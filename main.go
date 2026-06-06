package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prgzro/Blockz/consensus"
	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/crypto"
	"github.com/prgzro/Blockz/network"
	"github.com/prgzro/Blockz/types"
	"github.com/sirupsen/logrus"
)

const (
	// Version info
	clientName    = "Blockz"
	clientVersion = "1.0.0"
	chainID       = 100

	// Genesis allocation — pre-funded accounts (like Ethereum's genesis.json alloc)
	// These addresses get initial balances so we can immediately test transfers
	genesisAllocTotal = 1_000_000_000 // 1 billion native tokens

	// Block reward — miner/validator reward per block (like Ethereum's 2 ETH)
	blockReward = 5
)

// ══════════════════════════════════════════════════════════════════
//  CLI Flags — mirrors geth-style configuration
// ══════════════════════════════════════════════════════════════════

var (
	flagNodeID     = flag.String("id", "blockz-node-1", "Unique node identifier")
	flagListenAddr = flag.String("listen", ":9000", "P2P listen address")
	flagRPCAddr    = flag.String("rpc", ":8545", "JSON-RPC HTTP address (empty to disable)")
	flagMine       = flag.Bool("mine", true, "Enable block production (validator mode)")
	flagConsensus  = flag.String("consensus", "pow", "Consensus engine: pow or pos")
	flagDifficulty = flag.Uint64("difficulty", 8, "PoW mining difficulty (bits of leading zeros)")
	flagBlockTime  = flag.Duration("blocktime", 5*time.Second, "Target block interval")
	flagDataDir    = flag.String("datadir", "./blockz_data", "Directory for blockchain database")
	flagSeedNodes  = flag.String("seeds", "", "Comma-separated seed node addresses for discovery")
	flagLogLevel   = flag.String("loglevel", "info", "Log level: debug, info, warn, error")
	flagDevMode    = flag.Bool("dev", true, "Developer mode with pre-funded accounts and local transports")
)

func main() {
	flag.Parse()
	configureLogging(*flagLogLevel)
	printBanner()

	if *flagDevMode {
		runDevMode()
	} else {
		runProductionMode()
	}
}

// ══════════════════════════════════════════════════════════════════
//  Developer Mode — quick local multi-node setup for testing
// ══════════════════════════════════════════════════════════════════

func runDevMode() {
	logrus.Info("🔧 Starting in DEVELOPER mode with 4-node local network")
	logrus.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Generate the validator's keypair (like geth --mine with an etherbase)
	validatorKey := crypto.GeneratePrivateKey()
	validatorAddr := validatorKey.PublicKey().Address()
	logrus.Infof("⛏  Validator address: 0x%s", validatorAddr)

	// Genesis allocation — pre-fund the validator + dev accounts
	genesisAllocs := createGenesisAllocs(validatorAddr)
	logGenesisAllocs(genesisAllocs)

	// Create consensus engine
	engine := createConsensusEngine(*flagConsensus, *flagDifficulty, validatorAddr)

	// Create local transport mesh (simulates a 4-node P2P network)
	trLocal := network.NewLocalTransport("LOCAL")
	trRemoteA := network.NewLocalTransport("REMOTE_A")
	trRemoteB := network.NewLocalTransport("REMOTE_B")
	trRemoteC := network.NewLocalTransport("REMOTE_C")

	// Connect transports (full mesh for dev mode symmetrically)
	connectSym := func(t1, t2 *network.LocalTransport) {
		t1.Connect(t2)
		t2.Connect(t1)
	}
	connectSym(trLocal, trRemoteA)
	connectSym(trLocal, trRemoteB)
	connectSym(trLocal, trRemoteC)
	connectSym(trRemoteA, trRemoteB)
	connectSym(trRemoteA, trRemoteC)
	connectSym(trRemoteB, trRemoteC)

	// Boot remote (non-mining) nodes
	for i, tr := range []network.Transport{trRemoteA, trRemoteB, trRemoteC} {
		id := fmt.Sprintf("peer-%d", i+1)
		s := mustCreateServer(id, tr, nil, "", engine, core.NewMemoryStorage())
		go s.Start()
		logrus.Infof("🌐 Started peer node: %s", id)
	}

	// Send periodic test transactions from a remote node
	go txGeneratorLoop(trRemoteA, trLocal.Addr())

	// Boot the local (mining/validating) node with persistent storage + RPC
	storage := mustCreateStorage(*flagDataDir)
	localServer := mustCreateServer(*flagNodeID, trLocal, &validatorKey, *flagRPCAddr, engine, storage)

	logrus.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logrus.Infof("🚀 Node %s is LIVE", *flagNodeID)
	logrus.Infof("   📡 P2P listening on local transport")
	logrus.Infof("   🔌 JSON-RPC server at http://localhost%s", *flagRPCAddr)
	logrus.Infof("   ⛏  Mining: enabled (consensus=%s, difficulty=%d)", *flagConsensus, *flagDifficulty)
	logrus.Infof("   💾 Data directory: %s", *flagDataDir)
	logrus.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logrus.Info("")
	logrus.Info("💡 Test with:")
	logrus.Info("   curl -s -X POST http://localhost:8545 \\")
	logrus.Info("     -H 'Content-Type: application/json' \\")
	logrus.Info("     -d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}'")
	logrus.Info("")

	// Graceful shutdown
	go handleShutdown()

	localServer.Start() // blocks forever
}

// ══════════════════════════════════════════════════════════════════
//  Production Mode — TCP networking with real peer discovery
// ══════════════════════════════════════════════════════════════════

func runProductionMode() {
	logrus.Info("🚀 Starting in PRODUCTION mode")

	validatorKey := crypto.GeneratePrivateKey()
	validatorAddr := validatorKey.PublicKey().Address()
	logrus.Infof("⛏  Validator address: 0x%s", validatorAddr)

	engine := createConsensusEngine(*flagConsensus, *flagDifficulty, validatorAddr)
	storage := mustCreateStorage(*flagDataDir)
	tcpTransport := network.NewTCPTransport(network.NetAddr(*flagListenAddr))

	if err := tcpTransport.Listen(); err != nil {
		logrus.Fatalf("Failed to start P2P listener: %s", err)
	}

	var pk *crypto.PrivateKey
	if *flagMine {
		pk = &validatorKey
	}

	seeds := []string{}
	if *flagSeedNodes != "" {
		seeds = strings.Split(*flagSeedNodes, ",")
	}

	opts := network.ServerOpts{
		ID:         *flagNodeID,
		Transports: []network.Transport{tcpTransport},
		PrivateKey: pk,
		Consensus:  engine,
		RPCAddr:    *flagRPCAddr,
		Storage:    storage,
		BlockTime:  *flagBlockTime,
		SeedNodes:  seeds,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		logrus.Fatalf("Failed to create server: %s", err)
	}

	logrus.Infof("   📡 P2P listening on %s", *flagListenAddr)
	logrus.Infof("   🔌 JSON-RPC server at http://localhost%s", *flagRPCAddr)
	logrus.Infof("   ⛏  Mining: %v", *flagMine)

	go handleShutdown()
	s.Start()
}

// ══════════════════════════════════════════════════════════════════
//  Genesis Configuration
// ══════════════════════════════════════════════════════════════════

func createGenesisAllocs(validatorAddr types.Address) map[types.Address]uint64 {
	allocs := make(map[types.Address]uint64)

	// Fund the validator (like setting etherbase balance)
	allocs[validatorAddr] = genesisAllocTotal / 2 // 500M tokens

	// Create well-known dev accounts (like Hardhat/Anvil dev accounts)
	devAddresses := []string{
		"f39Fd6e51aad88F6F4ce6aB8827279cffFb92266", // Dev account #0
		"70997970C51812dc3A010C7d01b50e0d17dc79C8", // Dev account #1
		"3C44CdDdB6a900fa2b585dd299e03d12FA4293BC", // Dev account #2
		"90F79bf6EB2c4f870365E785982E1f101E93b906", // Dev account #3
		"15d34AAf54267DB7D7c367839AAf71A00a2C6A65", // Dev account #4
	}

	perDevAccount := (genesisAllocTotal / 2) / uint64(len(devAddresses))
	for _, addrHex := range devAddresses {
		decoded, err := hex.DecodeString(addrHex)
		if err != nil {
			logrus.Warnf("Skipping invalid dev address: %s", addrHex)
			continue
		}
		if len(decoded) == 20 {
			addr := types.AddressFromBytes(decoded)
			allocs[addr] = perDevAccount
		}
	}

	return allocs
}

func logGenesisAllocs(allocs map[types.Address]uint64) {
	logrus.Info("📜 Genesis Allocation:")
	for addr, balance := range allocs {
		logrus.Infof("   💰 0x%s  →  %d tokens", addr, balance)
	}
}

// ══════════════════════════════════════════════════════════════════
//  Consensus Factory
// ══════════════════════════════════════════════════════════════════

func createConsensusEngine(mode string, difficulty uint64, validatorAddr types.Address) core.ConsensusEngine {
	switch strings.ToLower(mode) {
	case "pos":
		logrus.Info("🔐 Consensus: Proof of Stake (PoS)")
		pos := consensus.NewPoSEngine()
		pos.AddValidator(validatorAddr, 1_000_000) // Bootstrap validator with initial stake
		return pos
	default:
		logrus.Infof("⛏  Consensus: Proof of Work (PoW) — difficulty=%d", difficulty)
		return consensus.NewPoWEngine(difficulty)
	}
}

// ══════════════════════════════════════════════════════════════════
//  Server & Storage Factories
// ══════════════════════════════════════════════════════════════════

func mustCreateStorage(dataDir string) core.Storage {
	storage, err := core.NewLevelDBStore(dataDir)
	if err != nil {
		logrus.Fatalf("❌ Failed to open database at %s: %s", dataDir, err)
	}
	logrus.Infof("💾 Opened database: %s", dataDir)
	return storage
}

func mustCreateServer(id string, tr network.Transport, pk *crypto.PrivateKey, rpcAddr string, engine core.ConsensusEngine, storage core.Storage) *network.Server {
	opts := network.ServerOpts{
		ID:         id,
		Transports: []network.Transport{tr},
		PrivateKey: pk,
		Consensus:  engine,
		RPCAddr:    rpcAddr,
		Storage:    storage,
		BlockTime:  *flagBlockTime,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		logrus.Fatalf("❌ Failed to create server %s: %s", id, err)
	}
	return s
}

// ══════════════════════════════════════════════════════════════════
//  Transaction Generator (simulates network activity)
// ══════════════════════════════════════════════════════════════════

func txGeneratorLoop(tr network.Transport, to network.NetAddr) {
	logrus.Info("🔄 Transaction generator started (every 2s)")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := sendTestTransaction(tr, to); err != nil {
			logrus.WithError(err).Warn("Failed to send test transaction")
		}
	}
}

func sendTestTransaction(tr network.Transport, to network.NetAddr) error {
	privKey := crypto.GeneratePrivateKey()

	// Create a simple smart contract: push 2, push 3, add, store result as "FOO"
	tx := core.NewTransaction(sampleContract())
	tx.GasLimit = 10000
	tx.GasPrice = 1
	if err := tx.Sign(privKey); err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		return err
	}

	msg := network.NewMessage(network.MessageTypeTx, buf.Bytes())
	return tr.SendMessage(to, msg.Bytes())
}

func sampleContract() []byte {
	// True EVM bytecode
	// PUSH1 0x05 (value), PUSH1 0x00 (key), SSTORE
	// PUSH1 0x00 (key), SLOAD
	// PUSH1 0x00 (offset), MSTORE
	// PUSH1 0x20 (size), PUSH1 0x00 (offset), RETURN
	data := []byte{
		0x60, 0x05, // PUSH1 5
		0x60, 0x00, // PUSH1 0
		0x55,       // SSTORE
		0x60, 0x00, // PUSH1 0
		0x54,       // SLOAD -> 5 is on stack
		0x60, 0x00, // PUSH1 0
		0x52,       // MSTORE (saves 5 at memory 0)
		0x60, 0x20, // PUSH1 0x20 (32 bytes)
		0x60, 0x00, // PUSH1 0 (offset)
		0xf3, // RETURN
	}
	return data
}

// ══════════════════════════════════════════════════════════════════
//  Utilities
// ══════════════════════════════════════════════════════════════════

func configureLogging(level string) {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
		ForceColors:     true,
	})

	switch strings.ToLower(level) {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func handleShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logrus.Infof("\n⏹  Received %s — shutting down gracefully...", sig)
	os.Exit(0)
}

func printBanner() {
	banner := `
  ____  _            _        
 | __ )| | ___   ___| | __ ____
 |  _ \| |/ _ \ / __| |/ /|_  /
 | |_) | | (_) | (__|   <  / / 
 |____/|_|\___/ \___|_|\_\/___| v%s

  ⛓  Layer-1 Blockchain Node
  🔐 Dual PoW/PoS Consensus  
  💻 EVM-Compatible Execution  
  🌐 P2P Network + JSON-RPC
  
  Chain ID:  %d (0x%x)
  Node ID:   %s
`
	fmt.Printf(banner, clientVersion, chainID, chainID, *flagNodeID)
	fmt.Println(strings.Repeat("━", 52))
}
