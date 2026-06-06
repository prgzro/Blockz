package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
)

// TCPTransport is a network transport mechanism that uses TCP connections.
type TCPTransport struct {
	addr      NetAddr
	listener  net.Listener
	consumeCh chan RPC

	mu    sync.RWMutex
	peers map[NetAddr]*TCPPeer
}

// TCPPeer represents a connection to a remote node.
type TCPPeer struct {
	net.Conn
	// Outbound indicates if we initiated the connection.
	Outbound bool
}

// NewTCPTransport creates a new TCP transport listening on the given address.
func NewTCPTransport(addr NetAddr) *TCPTransport {
	return &TCPTransport{
		addr:      addr,
		consumeCh: make(chan RPC, 1024),
		peers:     make(map[NetAddr]*TCPPeer),
	}
}

func (t *TCPTransport) Addr() NetAddr {
	return t.addr
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.consumeCh
}

// Connect implements the Transport interface.
// It dials the remote address and starts the read loop.
func (t *TCPTransport) Connect(tr Transport) error {
	// In the real world, tr would be a NetAddr.
	// But the existing interface takes a Transport.
	// For TCP, we'll dial the remote Addr().
	conn, err := net.Dial("tcp", string(tr.Addr()))
	if err != nil {
		return err
	}

	peer := &TCPPeer{
		Conn:     conn,
		Outbound: true,
	}

	t.mu.Lock()
	t.peers[NetAddr(conn.RemoteAddr().String())] = peer
	t.mu.Unlock()

	go t.readLoop(peer)

	return nil
}

// SendMessage sends a payload to a specific peer.
func (t *TCPTransport) SendMessage(to NetAddr, payload []byte) error {
	t.mu.RLock()
	peer, ok := t.peers[to]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("peer %s not found", to)
	}

	// We use a simple length-prefixed protocol or just direct payload?
	// For consistency with existing RPC decoding, we'll just send the bytes.
	_, err := peer.Write(payload)
	return err
}

// Broadcast sends a message to all connected peers.
func (t *TCPTransport) Broadcast(payload []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, peer := range t.peers {
		_, err := peer.Write(payload)
		if err != nil {
			fmt.Printf("failed to broadcast to %s: %s\n", peer.RemoteAddr(), err)
		}
	}
	return nil
}

// Listen starts the TCP listener and accepts incoming connections.
func (t *TCPTransport) Listen() error {
	ln, err := net.Listen("tcp", string(t.addr))
	if err != nil {
		return err
	}
	t.listener = ln

	go t.acceptLoop()

	return nil
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("accept error: %s\n", err)
			continue
		}

		peer := &TCPPeer{
			Conn:     conn,
			Outbound: false,
		}

		t.mu.Lock()
		t.peers[NetAddr(conn.RemoteAddr().String())] = peer
		t.mu.Unlock()

		go t.readLoop(peer)
	}
}

func (t *TCPTransport) readLoop(peer *TCPPeer) {
	defer func() {
		peer.Close()
		t.mu.Lock()
		delete(t.peers, NetAddr(peer.RemoteAddr().String()))
		t.mu.Unlock()
	}()

	decoder := gob.NewDecoder(peer.Conn)
	for {
		msg := new(Message)
		if err := decoder.Decode(msg); err != nil {
			return
		}

		// Re-encode to a buffer so the Server's RPCDecodeFunc can decode it independently
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			continue
		}

		t.consumeCh <- RPC{
			From:    NetAddr(peer.RemoteAddr().String()),
			Payload: buf,
		}
	}
}
