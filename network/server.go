package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"time"

	"sync"

	"github.com/go-kit/log"
	"github.com/prgzro/Blockz/api"
	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/crypto"
	"github.com/prgzro/Blockz/types"
)

var defaultBlockTime = 5 * time.Second

const defaultTxPoolMaxLength = 10000 // You can make this configurable later

type ServerOpts struct { // ServerOpts holds configuration options for the Server
	ID            string
	Logger        log.Logger
	RPCDecodeFunc RPCDecodeFunc // RPCHandler processes incoming RPC messages
	RPCProcessor  RPCProcessor
	Transports    []Transport // Transports is a list of Transport interfaces for network communication
	BlockTime     time.Duration
	PrivateKey    *crypto.PrivateKey
	Consensus     core.ConsensusEngine
	RPCAddr       string
	Storage       core.Storage
	SeedNodes     []string
}

type Server struct {
	ServerOpts
	memPool     *TxPool
	chain       *core.Blockchain
	isValidator bool
	consensus   core.ConsensusEngine
	rpcCh       chan RPC
	quitCh      chan struct{}

	peerLock sync.RWMutex
	peers    map[NetAddr]Transport
}

func NewServer(opts ServerOpts) (*Server, error) { // NewServer creates a new Server instance with the provided options

	if opts.BlockTime == time.Duration(0) {
		opts.BlockTime = defaultBlockTime
	}

	if opts.RPCDecodeFunc == nil {
		opts.RPCDecodeFunc = DefaultRPCDecodeFunc
	}

	if opts.Logger == nil {
		opts.Logger = log.NewLogfmtLogger(os.Stderr)
		opts.Logger = log.With(opts.Logger, "ID", opts.ID)
	}

	if opts.Storage == nil {
		opts.Storage = core.NewMemoryStorage()
	}

	chain, err := core.NewBlockchain(opts.Logger, genesisBlock(), opts.Storage)
	if err != nil {
		return nil, err
	}

	// Use the new TxPool with max length limit and FIFO pruning
	txPool := NewTxPool(defaultTxPoolMaxLength)

	s := &Server{ // Return a pointer to a new Server struct
		ServerOpts:  opts, // Embed the provided ServerOpts
		memPool:     txPool,
		chain:       chain,
		isValidator: opts.PrivateKey != nil,
		consensus:   opts.Consensus,
		rpcCh:       make(chan RPC),         // Channel for incoming RPC messages
		quitCh:      make(chan struct{}, 1), // Buffered channel to signal server shutdown
		peers:       make(map[NetAddr]Transport),
	}

	if s.consensus != nil {
		s.chain.SetConsensus(s.consensus)
	}

	// IF we got any processor from the server options, we going to user
	// the server as default.
	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}

	if s.isValidator {
		go s.ValidatorLoop()
	}

	if s.RPCAddr != "" {
		s.initRPCServer()
	}

	return s, nil
}

func (s *Server) Start() { // Start initializes the server and begins processing RPC messages
	s.initTransports() // Initialize the transports
	go s.discoveryLoop()

free: // Label to break out of the for loop
	for { // Infinite loop to process incoming RPC messages and other events
		select { // Select statement to handle multiple channels
		case rpc := <-s.rpcCh: // Case for receiving an RPC message
			msg, err := s.RPCDecodeFunc(rpc) // Decode the RPC message using the provided decode function
			if err != nil {
				s.Logger.Log("error", err)
			}
			s.Logger.Log("msg", "processing RPC", "from", rpc.From)
			if err := s.RPCProcessor.ProcessMessage(msg); err != nil {
				s.Logger.Log("error", err)
			}
			//
		case <-s.quitCh: // Case for receiving a shutdown signal
			break free // Break out of the loop to stop the server
		}
	}

	s.Logger.Log("msg", "Server is shutting down")
}

func (s *Server) ValidatorLoop() {
	ticker := time.NewTicker(s.BlockTime)
	s.Logger.Log("msg", "Starting validator loop", "blockTime", s.BlockTime)

	for {
		<-ticker.C
		s.createNewBlock()
	}
}

func (s *Server) ProcessMessage(msg *DecodedMessage) error {
	switch t := msg.Data.(type) {
	case *core.Transaction:
		return s.ProcessTransaction(t)
	case *core.Block:
		return s.processBlock(msg.From, t)
	case *GetBlocksMessage:
		return s.processGetBlocks(msg.From, t)
	case *BlocksMessage:
		return s.processBlocks(msg.From, t)
	case *StatusMessage:
		return s.processStatusMessage(msg.From, t)
	case *GetStatusMessage:
		return s.processGetStatusMessage(msg.From, t)
	case *GetPeersMessage:
		return s.handleGetPeers(msg.From)
	case *SharePeersMessage:
		return s.handleSharePeers(t)
	}

	return nil
}

func (s *Server) broadcast(payload []byte) error {
	for _, tr := range s.Transports {
		if err := tr.Broadcast(payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) processBlock(from NetAddr, b *core.Block) error {
	if err := s.chain.AddBlock(b); err != nil {
		// If the block is too high, it means we are behind and need to sync
		if strings.Contains(err.Error(), "too high") {
			s.Logger.Log("msg", "received block with height too high, triggering sync", "height", b.Header.Height, "currentHeight", s.chain.Height())
			return s.requestBlocksSync(from)
		}

		// If we already have the block (normal in P2P gossip), just silently drop it
		if strings.Contains(err.Error(), "chain already contains block") {
			return nil
		}

		return err
	}

	go s.broadcastBlock(b)

	return nil
}

func (s *Server) requestBlocksSync(from NetAddr) error {
	msg := GetBlocksMessage{
		From: s.chain.Height() + 1,
		To:   0, // 0 means give me everything you have from 'From'
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	return s.send(from, NewMessage(MessageTypeGetBlocks, buf.Bytes()))
}

func (s *Server) ProcessTransaction(tx *core.Transaction) error {

	hash := tx.Hash(core.TxHasher{})

	if s.memPool.Contains(hash) {
		return nil
	}

	if err := tx.Verify(); err != nil {
		return err
	}

	tx.SetFirstSeen(time.Now().UnixNano())

	// s.Logger.Log(
	// 	"msg", "adding new tx to mempool",
	// 	"hash", hash,
	// 	"mempoolPending", s.memPool.PendingCount(),
	// )

	go s.broadcastTx(tx)

	s.memPool.Add(tx)

	return nil
}

func (s *Server) BroadcastTransaction(tx *core.Transaction) error {
	return s.ProcessTransaction(tx)
}

func (s *Server) broadcastBlock(b *core.Block) error {
	buf := &bytes.Buffer{}

	if err := b.Encode(core.NewGobBlockEncoder(buf)); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeBlock, buf.Bytes())

	return s.broadcast(msg.Bytes())
}

func (s *Server) broadcastTx(tx *core.Transaction) error {
	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeTx, buf.Bytes())
	return s.broadcast(msg.Bytes())
}

func (s *Server) initTransports() {
	for _, tr := range s.Transports {
		go func(tr Transport) {
			for rpc := range tr.Consume() {
				s.addPeer(rpc.From, tr)
				s.rpcCh <- rpc
			}
		}(tr)
	}
}

func (s *Server) addPeer(addr NetAddr, tr Transport) {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	if _, ok := s.peers[addr]; !ok {
		s.Logger.Log("msg", "adding peer", "addr", addr)
		s.peers[addr] = tr
	}
}

func (s *Server) createNewBlock() error {
	currentHeader, err := s.chain.GetHeader(s.chain.Height())
	if err != nil {
		return err
	}

	txx := s.memPool.Pending()

	block, err := core.NewBlockFromPrevHeader(currentHeader, txx)
	if err != nil {
		return err
	}

	// 1. Prepare consensus fields (e.g. difficulty adjustment, coinbase)
	if s.consensus != nil {
		if err := s.consensus.Prepare(s.chain, block.Header); err != nil {
			return err
		}
	}

	// 2. Set coinbase to validator's address
	if s.isValidator {
		block.Header.Coinbase = s.PrivateKey.PublicKey().Address()
	}

	// 3. Seal the block (e.g. mining for PoW)
	if s.consensus != nil {
		if err := s.consensus.Seal(s.chain, block); err != nil {
			return err
		}
	}

	if err := block.Sign(*s.PrivateKey); err != nil {
		return err
	}

	if err := s.chain.AddBlock(block); err != nil {
		return err
	}

	s.memPool.ClearPending()

	go s.broadcastBlock(block)

	return nil
}

func (s *Server) processStatusMessage(from NetAddr, msg *StatusMessage) error {
	s.Logger.Log("msg", "received status message", "from", msg.ID, "height", msg.Height)

	if msg.Height > s.chain.Height() {
		s.Logger.Log("msg", "peer is ahead of us, syncing", "peerHeight", msg.Height, "ourHeight", s.chain.Height())

		getBlocks := &GetBlocksMessage{
			From: s.chain.Height() + 1,
			To:   msg.Height,
		}

		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(getBlocks); err != nil {
			return err
		}

		s.Logger.Log("msg", "requesting blocks", "from", getBlocks.From, "to", getBlocks.To)
		return s.send(from, NewMessage(MessageTypeGetBlocks, buf.Bytes()))
	}

	return nil
}

func (s *Server) processGetStatusMessage(from NetAddr, msg *GetStatusMessage) error {
	s.Logger.Log("msg", "received get status message")

	status := &StatusMessage{
		ID:      s.ID,
		Version: 1,
		Height:  s.chain.Height(),
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(status); err != nil {
		return err
	}

	return s.send(from, NewMessage(MessageTypeStatus, buf.Bytes()))
}

func (s *Server) processGetBlocks(from NetAddr, msg *GetBlocksMessage) error {
	s.Logger.Log("msg", "received get blocks request", "from", msg.From, "to", msg.To)

	blocks := []*core.Block{}

	to := msg.To
	if to == 0 || to > s.chain.Height() {
		to = s.chain.Height()
	}

	for i := msg.From; i <= to; i++ {
		block, err := s.chain.GetBlock(i)
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
	}

	blocksMsg := &BlocksMessage{
		Blocks: blocks,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(blocksMsg); err != nil {
		return err
	}

	return s.send(from, NewMessage(MessageTypeBlocks, buf.Bytes()))
}

func (s *Server) processBlocks(from NetAddr, msg *BlocksMessage) error {
	s.Logger.Log("msg", "received blocks batch", "count", len(msg.Blocks))

	for _, b := range msg.Blocks {
		if err := s.chain.AddBlock(b); err != nil {
			s.Logger.Log("error", "failed to add block during sync", "height", b.Header.Height, "err", err)
			return err
		}
	}

	return nil
}

func (s *Server) send(addr NetAddr, msg *Message) error {
	s.peerLock.RLock()
	tr, ok := s.peers[addr]
	s.peerLock.RUnlock()

	if !ok {
		// Try to see if we have it under a different name (e.g. 127.0.0.1 vs localhost)
		// This is a bit of a hack but helps in local dev
		if strings.Contains(string(addr), "localhost") {
			altAddr := NetAddr(strings.Replace(string(addr), "localhost", "127.0.0.1", 1))
			s.peerLock.RLock()
			tr, ok = s.peers[altAddr]
			s.peerLock.RUnlock()
		}
	}

	if !ok {
		return fmt.Errorf("peer %s not found", addr)
	}

	return tr.SendMessage(addr, msg.Bytes())
}

func (s *Server) initRPCServer() {
	rpcServer := api.NewRPCServer(s.chain, s)
	go rpcServer.Start(s.RPCAddr)
	s.Logger.Log("msg", "Starting RPC server", "addr", s.RPCAddr)
}

func genesisBlock() *core.Block {
	header := &core.Header{
		Version:   1,
		DataHash:  types.Hash{},
		Height:    0,
		Timestamp: 000000,
	}

	b, _ := core.NewBlock(header, nil)
	return b
}

func (s *Server) discoveryLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial discovery
	s.bootstrap()

	for {
		select {
		case <-ticker.C:
			s.bootstrap()
		case <-s.quitCh:
			return
		}
	}
}

func (s *Server) bootstrap() {
	for _, addr := range s.SeedNodes {
		// connect to seed node if not already connected
		if !s.isPeerConnected(NetAddr(addr)) {
			s.Logger.Log("msg", "connecting to seed node", "addr", addr)
			// Assuming we use the first transport for outgoing connections
			// Note: TCPTransport.Connect takes another Transport just to get its Addr().
			// We can pass a dummy one or update Connect signature.
			// Current TCPTransport.Connect(tr Transport) uses tr.Addr().
			if err := s.Transports[0].Connect(NewTCPTransport(NetAddr(addr))); err != nil {
				s.Logger.Log("error", err)
				continue
			}
		}

		// Resolve localhost to 127.0.0.1 for consistency
		pAddr := NetAddr(addr)
		if strings.Contains(string(pAddr), "localhost") {
			pAddr = NetAddr(strings.Replace(string(pAddr), "localhost", "127.0.0.1", 1))
		}

		// Ask for peers and share our status
		s.sendGetStatus(pAddr)
		s.sendGetPeers(pAddr)
	}
}

func (s *Server) sendGetStatus(addr NetAddr) error {
	msg := NewMessage(MessageTypeGetStatus, nil)
	return s.send(addr, msg)
}

func (s *Server) isPeerConnected(addr NetAddr) bool {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()
	_, ok := s.peers[addr]
	return ok
}

func (s *Server) sendGetPeers(to NetAddr) error {
	msg := Message{
		Header: MessageTypeGetPeers,
		Data:   []byte{},
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	return s.Transports[0].SendMessage(to, buf.Bytes())
}

func (s *Server) handleGetPeers(from NetAddr) error {
	s.peerLock.RLock()
	peers := []NetAddr{}
	for addr := range s.peers {
		// Don't send the requester back their own address
		if addr != from {
			peers = append(peers, addr)
		}
	}
	s.peerLock.RUnlock()

	msg := SharePeersMessage{Peers: peers}
	payload := new(bytes.Buffer)
	if err := gob.NewEncoder(payload).Encode(msg); err != nil {
		return err
	}

	fullMsg := Message{
		Header: MessageTypeSharePeers,
		Data:   payload.Bytes(),
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(fullMsg); err != nil {
		return err
	}

	return s.Transports[0].SendMessage(from, buf.Bytes())
}

func (s *Server) handleSharePeers(msg *SharePeersMessage) error {
	for _, addr := range msg.Peers {
		if !s.isPeerConnected(addr) && addr != s.Transports[0].Addr() {
			s.Logger.Log("msg", "discovered new peer", "addr", addr)
			// Connect to newly discovered peer
			go s.Transports[0].Connect(NewTCPTransport(addr))
		}
	}
	return nil
}
