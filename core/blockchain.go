package core

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/go-kit/log"
	"github.com/prgzro/Blockz/types"
)

type Blockchain struct {
	logger          log.Logger
	store           Storage
	lock            sync.RWMutex
	Headers         []*Header
	consensus       ConsensusEngine
	validator       Validator
	WorldState      *WorldState // Account-based world state (balances, nonces, code)
	totalDifficulty *big.Int
}

func NewBlockchain(l log.Logger, genesis *Block, store Storage) (*Blockchain, error) {
	bc := &Blockchain{
		Headers:         []*Header{},
		store:           store,
		logger:          l,
		WorldState:      NewWorldState(),
		totalDifficulty: big.NewInt(0),
	}
	bc.validator = NewBlockValidator(bc)

	// If store is empty, add genesis.
	// Otherwise load headers from store.
	if _, err := bc.store.Get(0); err != nil {
		err := bc.addBlockWithoutValidation(genesis)
		if err != nil {
			return nil, err
		}
	} else {
		if err := bc.loadHeaders(); err != nil {
			return nil, err
		}
	}

	return bc, nil
}

// NewBlockchainWithState creates a blockchain with a pre-initialized world state.
// Used for genesis initialization with pre-funded accounts.
func NewBlockchainWithState(l log.Logger, genesis *Block, ws *WorldState) (*Blockchain, error) {
	bc := &Blockchain{
		Headers:    []*Header{},
		store:      NewMemoryStorage(),
		logger:     l,
		WorldState: ws,
	}
	bc.validator = NewBlockValidator(bc)
	err := bc.addBlockWithoutValidation(genesis)

	return bc, err
}

func (bc *Blockchain) SetValidator(v Validator) {
	bc.validator = v
}

func (bc *Blockchain) SetConsensus(c ConsensusEngine) {
	bc.consensus = c
}

func (bc *Blockchain) AddBlock(b *Block) error {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if err := bc.validator.ValidateBlockLocked(b); err != nil {
		return err
	}

	// Execute transactions and apply state transitions
	for _, tx := range b.Transactions {
		if err := bc.executeTransaction(tx); err != nil {
			return err
		}
	}

	// Process slashing evidence
	for _, evidence := range b.Evidence {
		if bc.consensus != nil {
			if slasher, ok := bc.consensus.(interface {
				Slash(types.Address, string)
			}); ok {
				slasher.Slash(evidence.Coinbase, "Double signing evidence")
			}
		}
	}

	// Mint block reward to coinbase
	bc.WorldState.AddBalance(b.Header.Coinbase, 5)

	return bc.addBlockWithoutValidationLocked(b)
}

func (bc *Blockchain) addBlockWithoutValidation(b *Block) error {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	return bc.addBlockWithoutValidationLocked(b)
}

func (bc *Blockchain) addBlockWithoutValidationLocked(b *Block) error {
	bc.Headers = append(bc.Headers, b.Header)

	bc.logger.Log(
		"msg", "new block",
		"hash", b.Hash(BlockHasher{}),
		"height", b.Header.Height,
		"transactions", len(b.Transactions),
	)
	if bc.consensus != nil {
		weight := bc.consensus.(interface {
			CalculateBlockWeight(*Header) *big.Int
		}).CalculateBlockWeight(b.Header)
		bc.totalDifficulty.Add(bc.totalDifficulty, weight)
	}

	return bc.store.Put(b)
}

// executeTransaction processes a single transaction:
// 1. If it has Data (contract code), run through the VM
// 2. If it has Value, transfer native currency between accounts
func (bc *Blockchain) executeTransaction(tx *Transaction) error {
	// Run contract code through the VM if present
	if len(tx.Data) > 0 {
		bc.logger.Log("msg", "executing code", "len", len(tx.Data), "hash", tx.Hash(&TxHasher{}))
		fromAddr := tx.From.Address()

		vm := NewEVM(tx.Data, nil, bc.WorldState, tx.To, fromAddr, new(big.Int).SetUint64(tx.Value), tx.GasLimit)
		if _, err := vm.Run(); err != nil {
			return err
		}
	}

	// Apply value transfer if present
	if tx.Value > 0 {
		fromAddr := tx.From.Address()
		if err := bc.WorldState.Transfer(fromAddr, tx.To, tx.Value); err != nil {
			return fmt.Errorf("transfer failed: %w", err)
		}
	}

	// Increment sender's nonce to prevent replay
	if tx.Signature != nil {
		fromAddr := tx.From.Address()
		bc.WorldState.IncrementNonce(fromAddr)
	}

	return nil
}

func (bc *Blockchain) GetHeader(height uint32) (*Header, error) {
	if height > bc.Height() {
		return nil, fmt.Errorf("given height (%d) Too high", height)
	}

	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.Headers[height], nil
}

func (bc *Blockchain) getHeaderLocked(height uint32) (*Header, error) {
	if height >= uint32(len(bc.Headers)) {
		return nil, fmt.Errorf("given height (%d) too high", height)
	}
	return bc.Headers[height], nil
}

// GetBlock returns the full block at the given height.
func (bc *Blockchain) GetBlock(height uint32) (*Block, error) {
	return bc.store.Get(height)
}

// GetBlockByHash returns the block with the given hash.
func (bc *Blockchain) GetBlockByHash(hash types.Hash) (*Block, error) {
	return bc.store.GetByHash(hash)
}

// GetTxByHash returns the transaction with the given hash.
func (bc *Blockchain) GetTxByHash(hash types.Hash) (*Transaction, error) {
	return bc.store.GetTxByHash(hash)
}

func (bc *Blockchain) HasBlock(height uint32) bool {
	return height <= bc.Height()
}

func (bc *Blockchain) hasBlockLocked(height uint32) bool {
	return height <= bc.heightLocked()
}

// Block height = the number of blocks from the genesis block (the very first block) up to the current block.
// [0, 1, 2, 3] ==> Len 4
// Height = Len - 1 = 3
// because the genesis block is at height 0.
func (bc *Blockchain) Height() uint32 {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.heightLocked()
}

func (bc *Blockchain) heightLocked() uint32 {
	return uint32(len(bc.Headers) - 1)
}

func (bc *Blockchain) loadHeaders() error {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	for i := uint32(0); ; i++ {
		block, err := bc.store.Get(i)
		if err != nil {
			break
		}
		bc.Headers = append(bc.Headers, block.Header)
	}

	bc.logger.Log("msg", "loaded headers from store", "count", len(bc.Headers))
	return nil
}
