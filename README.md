<div align="center">

# ⛓️ Blockz

### Unchaining the EVM — A sovereign Layer-1 blockchain built from scratch in Go.

[![Go Report Card](https://goreportcard.com/badge/github.com/prgzro/Blockz)](https://goreportcard.com/report/github.com/prgzro/Blockz)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)

<br>

**Blockz** is not another toy chain.  
It's a **full-stack, production-grade Layer-1 blockchain** engineered entirely from the ground up — no frameworks, no shortcuts.  
Custom consensus. Custom VM. Custom networking. **100% Go. 0% fluff.**

[Getting Started](#-getting-started) · [Architecture](#-architecture) · [CLI Reference](#-cli-reference) · [API](#-interact-with-the-node) · [Testing](#-testing) · [Security Lab](#-security-testbed)

</div>

---

## ⚡ At a Glance

| Layer | What's Under the Hood |
| :--- | :--- |
| **Consensus** | Hybrid **Dual Engine** — Proof-of-Work (SHA-256 mining) & Proof-of-Stake (validator registry with automated slashing) |
| **Execution** | Stack-based **EVM-compatible VM** with 12+ opcodes, gas metering, persistent contract storage, and conditional jumps |
| **State** | Account-based **World State** with balances, nonces, contract code, and deterministic `StateRoot` hashing |
| **Storage** | High-performance block persistence via **LevelDB** with O(1) lookups by height, hash, or tx hash |
| **Networking** | Custom **TCP P2P layer** — handshakes, seed-node discovery, gossip propagation, and block sync |
| **Security** | PoS **Byzantine slashing** (double-sign detection), fork-choice rules, and evidence processing |
| **API** | **JSON-RPC 2.0** interface — `eth_blockNumber`, `eth_getBalance`, `eth_getBlockByNumber`, `eth_sendRawTransaction` |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        main.go                              │
│              CLI flags · Dev/Prod mode · Genesis            │
├──────────┬──────────┬──────────┬──────────┬─────────────────┤
│   api/   │  core/   │consensus/│ network/ │    crypto/      │
│          │          │          │          │                 │
│ JSON-RPC │Blockchain│ PoW Eng  │ Server   │ ECDSA Keypairs  │
│ Handler  │ World    │ PoS Eng  │ TxPool   │ Sign / Verify   │
│          │ State    │ Fork     │ TCP P2P  │                 │
│          │ VM       │ Choice   │ Gossip   │                 │
│          │ LevelDB  │ Slashing │ Sync     │                 │
├──────────┴──────────┴──────────┴──────────┴─────────────────┤
│                       types/                                │
│           Address · Hash · Generic Collections              │
└─────────────────────────────────────────────────────────────┘
```

### Core Modules

<details>
<summary><b>📦 core/</b> — The beating heart of the chain</summary>

- **`blockchain.go`** — Chain management, block addition with consensus validation, transaction execution, fork handling
- **`world_state.go`** — Account-based state model: balances, nonces, contract storage, deterministic `StateRoot`
- **`vm.go`** — Stack-based virtual machine with gas metering, arithmetic ops, storage ops (`STORE`/`GET`), and control flow (`JUMP`/`JUMPIF`)
- **`block.go`** — Block and header structures with Merkle-style hashing
- **`transaction.go`** — Transaction model with signature verification and value transfers
- **`leveldb_storage.go`** — Persistent block storage with LevelDB
- **`account.go`** — Account structure with deep copy support for speculative execution
- **`validator.go`** — Block validation: height continuity, previous-hash linkage, consensus rules
</details>

<details>
<summary><b>🔒 consensus/</b> — Dual consensus engine</summary>

- **`pow.go`** — SHA-256 Proof-of-Work with adjustable difficulty, nonce mining, and timeout protection
- **`pos.go`** — Proof-of-Stake with weighted validator selection, double-sign detection, and 50% stake slashing
- **`fork_choice.go`** — Heaviest-chain fork choice rule based on cumulative difficulty
</details>

<details>
<summary><b>🌐 network/</b> — P2P networking layer</summary>

- **`server.go`** — Node orchestrator: validator loop, block creation, message routing, peer management
- **`tcp_transport.go`** — TCP transport with connection pooling and handshake protocol
- **`txpool.go`** — Transaction mempool with deduplication and configurable capacity
- **`rpc.go`** — Message codec and protocol definitions (blocks, txs, status, peer exchange)
- **`local_transport.go`** — In-memory transport for testing and dev mode
</details>

<details>
<summary><b>🔑 crypto/</b> — Cryptographic primitives</summary>

- **`keypair.go`** — ECDSA key generation, signing, and verification over P-256
</details>

<details>
<summary><b>🌐 api/</b> — External interface</summary>

- **`rpc.go`** — JSON-RPC 2.0 server compatible with standard Web3 tooling
</details>

---

## 🚀 Getting Started

### Prerequisites

- **Go 1.25+** — [Install Go](https://go.dev/dl/)
- LevelDB is bundled as a Go dependency — no external install required.

### Build

```bash
git clone https://github.com/prgzro/Blockz.git
cd Blockz
make build
```

### Run in Developer Mode

Spin up a multi-node local mesh with pre-funded accounts (Hardhat-style addresses like `0xf39F...`) and automatic block production:

```bash
make dev
```

Or with custom parameters:

```bash
./bin/Blockz --dev --difficulty 8 --blocktime 3s --consensus pow
```

### Run in Production Mode

Connect to a real TCP network with seed-based peer discovery:

```bash
./bin/Blockz \
  --id my-validator \
  --listen :9000 \
  --rpc :8545 \
  --seeds "seed.blockz.org:9000" \
  --mine \
  --consensus pow \
  --difficulty 16
```

---

## 🎛️ CLI Reference

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--id` | `blockz-node-1` | Unique node identifier |
| `--listen` | `:9000` | P2P listen address |
| `--rpc` | `:8545` | JSON-RPC HTTP address (empty to disable) |
| `--seeds` | *(empty)* | Comma-separated seed node addresses |
| `--mine` | `false` | Enable block production |
| `--consensus` | `pow` | Consensus engine: `pow` or `pos` |
| `--difficulty` | `16` | PoW difficulty (bits of leading zeros) |
| `--blocktime` | `5s` | Target block interval |
| `--datadir` | `blockz_data` | LevelDB data directory |
| `--loglevel` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `--dev` | `true` | Dev mode with local transports and pre-funded accounts |

---

## 💻 Interact with the Node

Blockz exposes a standard **JSON-RPC 2.0** interface on port `8545`. Use `curl`, Postman, or any Web3 library.

**Get Current Block Height:**
```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

**Query Account Balance:**
```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"],"id":1}'
```

**Get Block by Number:**
```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest"],"id":1}'
```

**Send Raw Transaction:**
```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["0x<encoded-tx>"],"id":1}'
```

### Supported RPC Methods

| Method | Description |
| :--- | :--- |
| `eth_blockNumber` | Returns the current block height |
| `eth_getBalance` | Returns the balance for a given address |
| `eth_getBlockByNumber` | Returns a block header by height or `"latest"` |
| `eth_sendRawTransaction` | Broadcasts a signed transaction |
| `eth_chainId` | Returns the chain ID (`0x64`) |
| `net_version` | Returns the network version (`100`) |

---

## 🧪 Testing

Blockz has **14 test files** spanning every layer of the stack — from cryptographic primitives to P2P transport.

```bash
# Run the full test suite
make test

# Run tests matching a pattern
make test TestVM
make test TestBlockchain
make test TestTxPool

# Run with coverage report
make coverage
```

### Test Coverage Map

| Package | Test Files | What's Covered |
| :--- | :--- | :--- |
| `core/` | 5 tests | Block hashing, blockchain ops, VM execution, world state, storage |
| `consensus/` | 2 tests | PoW mining & validation, consensus interface |
| `network/` | 3 tests | Local transport, TCP transport, transaction pool |
| `crypto/` | 1 test | ECDSA key generation, signing, verification |
| `api/` | 1 test | JSON-RPC request/response handling |
| `types/` | 1 test | Hash operations |

---

## 🛡️ Security Testbed

Blockz is purpose-built as a **security research platform**. The clean, readable codebase makes it an ideal environment for studying:

| Research Area | What You Can Explore |
| :--- | :--- |
| **Consensus Attacks** | Simulate reorgs, forks, 51% attacks, and selfish mining strategies |
| **Mempool Manipulation** | Study transaction ordering, MEV extraction, and front-running |
| **VM Exploits** | Test gas limit attacks, stack overflows, and execution environment edge cases |
| **P2P Attacks** | Explore eclipse attacks, Sybil resistance, and gossip protocol weaknesses |
| **Slashing Logic** | Verify double-sign detection and Byzantine fault tolerance |
| **State Attacks** | Probe balance overflows, nonce manipulation, and state root corruption |

---

## 📁 Project Structure

```
Blockz/
├── main.go              # Node entry — CLI, genesis, dev/prod mode
├── Makefile             # Build system
├── go.mod / go.sum      # Dependencies
│
├── api/                 # JSON-RPC 2.0 server
├── consensus/           # PoW + PoS + Fork choice
├── core/                # Blockchain, VM, World State, Storage
├── crypto/              # ECDSA keypairs
├── network/             # P2P server, transports, mempool
└── types/               # Address, Hash, generic collections
```

---

## 📜 License

Distributed under the **MIT License**. See `LICENSE` for more information.

---

<div align="center">

*Forged from scratch. No frameworks. No shortcuts.*

**Built with ❤️ for the decentralized future.**

</div>
