package core

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"

	"github.com/prgzro/Blockz/types"
)

// WorldState manages the global account state of the blockchain.
// It maps addresses to accounts, tracking balances, nonces, and contract data.
// This is the equivalent of Ethereum's "state trie" in a simplified form.
type WorldState struct {
	mu           sync.RWMutex
	accounts     map[types.Address]*Account
	contractCode map[types.Hash][]byte // codeHash -> bytecode
}

// NewWorldState creates a new empty world state.
func NewWorldState() *WorldState {
	return &WorldState{
		accounts:     make(map[types.Address]*Account),
		contractCode: make(map[types.Hash][]byte),
	}
}

// CreateGenesisState returns a WorldState pre-funded with genesis accounts.
// This bootstraps the chain with initial token distribution.
func CreateGenesisState(allocs map[types.Address]uint64) *WorldState {
	ws := NewWorldState()
	for addr, balance := range allocs {
		ws.accounts[addr] = NewAccount(balance)
	}
	return ws
}

// GetAccount returns the account at the given address.
// Returns nil if no account exists at that address.
func (ws *WorldState) GetAccount(addr types.Address) *Account {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.accounts[addr]
}

// GetOrCreateAccount returns the account at the given address, creating it if needed.
func (ws *WorldState) GetOrCreateAccount(addr types.Address) *Account {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		acc = NewAccount(0)
		ws.accounts[addr] = acc
	}
	return acc
}

// GetBalance returns the balance for the given address. Returns 0 if account doesn't exist.
func (ws *WorldState) GetBalance(addr types.Address) uint64 {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		return 0
	}
	return acc.Balance
}

// GetNonce returns the nonce for the given address. Returns 0 if account doesn't exist.
func (ws *WorldState) GetNonce(addr types.Address) uint64 {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		return 0
	}
	return acc.Nonce
}

// IncrementNonce increments the nonce for the given address.
func (ws *WorldState) IncrementNonce(addr types.Address) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		acc = NewAccount(0)
		ws.accounts[addr] = acc
	}
	acc.Nonce++
}

// Transfer moves value from one address to another.
// Returns an error if the sender has insufficient balance.
// This is a critical function — must be atomic (both debit and credit succeed or neither does).
func (ws *WorldState) Transfer(from, to types.Address, value uint64) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// Can't transfer zero — prevents spam and state bloat
	if value == 0 {
		return fmt.Errorf("cannot transfer zero value")
	}

	fromAcc, ok := ws.accounts[from]
	if !ok {
		return fmt.Errorf("sender account %s does not exist", from)
	}

	if fromAcc.Balance < value {
		return fmt.Errorf("insufficient balance: account %s has %d, tried to send %d",
			from, fromAcc.Balance, value)
	}

	toAcc, ok := ws.accounts[to]
	if !ok {
		toAcc = NewAccount(0)
		ws.accounts[to] = toAcc
	}

	// Atomic: debit sender and credit receiver
	fromAcc.Balance -= value
	toAcc.Balance += value

	return nil
}

// AddBalance adds the given amount to an account's balance.
// Used for mining rewards, staking rewards, etc.
func (ws *WorldState) AddBalance(addr types.Address, amount uint64) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		acc = NewAccount(0)
		ws.accounts[addr] = acc
	}
	acc.Balance += amount
}

// SubBalance subtracts the given amount from an account's balance.
// Returns error if insufficient balance.
func (ws *WorldState) SubBalance(addr types.Address, amount uint64) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		return fmt.Errorf("account %s does not exist", addr)
	}
	if acc.Balance < amount {
		return fmt.Errorf("insufficient balance: account %s has %d, tried to subtract %d",
			addr, acc.Balance, amount)
	}
	acc.Balance -= amount
	return nil
}

// SetCode stores contract bytecode for an account.
func (ws *WorldState) SetCode(addr types.Address, code []byte) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		acc = NewAccount(0)
		ws.accounts[addr] = acc
	}

	codeHash := sha256.Sum256(code)
	acc.CodeHash = types.Hash(codeHash)
	ws.contractCode[acc.CodeHash] = code
}

// GetCode retrieves the contract bytecode for an account.
func (ws *WorldState) GetCode(addr types.Address) []byte {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		return nil
	}
	return ws.contractCode[acc.CodeHash]
}

// StateRoot computes a deterministic hash of the entire world state.
// Used as the StateRoot in block headers to commit to a specific state.
func (ws *WorldState) StateRoot() types.Hash {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	// Sort addresses for deterministic ordering
	addrs := make([]types.Address, 0, len(ws.accounts))
	for addr := range ws.accounts {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		for k := 0; k < 20; k++ {
			if addrs[i][k] != addrs[j][k] {
				return addrs[i][k] < addrs[j][k]
			}
		}
		return false
	})

	// Hash all accounts in sorted order
	var combined []byte
	for _, addr := range addrs {
		acc := ws.accounts[addr]
		data, err := acc.Encode()
		if err != nil {
			continue
		}
		combined = append(combined, addr.ToSlice()...)
		combined = append(combined, data...)
	}

	h := sha256.Sum256(combined)
	return types.Hash(h)
}

// Copy creates a deep copy of the world state.
// Used for speculative execution (eth_call) without modifying canonical state.
func (ws *WorldState) Copy() *WorldState {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	newState := NewWorldState()
	for addr, acc := range ws.accounts {
		newState.accounts[addr] = acc.DeepCopy()
	}
	for hash, code := range ws.contractCode {
		codeCopy := make([]byte, len(code))
		copy(codeCopy, code)
		newState.contractCode[hash] = codeCopy
	}
	return newState
}

// AccountCount returns the number of accounts in the state.
func (ws *WorldState) AccountCount() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return len(ws.accounts)
}

func (ws *WorldState) SetAccountState(addr types.Address, key types.Hash, value []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		acc = NewAccount(0)
		ws.accounts[addr] = acc
	}

	acc.Storage[key] = value
	return nil
}

func (ws *WorldState) GetAccountState(addr types.Address, key types.Hash) ([]byte, error) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	acc, ok := ws.accounts[addr]
	if !ok {
		return nil, fmt.Errorf("account %s not found", addr)
	}

	val, ok := acc.Storage[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found in account %s", key, addr)
	}

	return val, nil
}
