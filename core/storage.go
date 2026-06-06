package core

import (
	"fmt"
	"sync"

	"github.com/prgzro/Blockz/types"
)

// Storage is the interface for persisting blocks.
type Storage interface {
	Put(*Block) error
	Get(height uint32) (*Block, error)
	GetByHash(hash types.Hash) (*Block, error)
	GetTxByHash(hash types.Hash) (*Transaction, error)
}

// MemoryStore implements Storage with in-memory maps for fast lookups.
// Indexes blocks by both height and hash for O(1) retrieval.
type MemoryStore struct {
	mu          sync.RWMutex
	blocks      map[uint32]*Block     // height -> block
	blockByHash map[types.Hash]*Block // block hash -> block
	txByHash    map[types.Hash]*Transaction
}

func NewMemoryStorage() *MemoryStore {
	return &MemoryStore{
		blocks:      make(map[uint32]*Block),
		blockByHash: make(map[types.Hash]*Block),
		txByHash:    make(map[types.Hash]*Transaction),
	}
}

func (s *MemoryStore) Put(b *Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := b.Hash(BlockHasher{})
	s.blocks[b.Header.Height] = b
	s.blockByHash[hash] = b

	// Index all transactions in this block for later retrieval
	for _, tx := range b.Transactions {
		txHash := tx.Hash(&TxHasher{})
		s.txByHash[txHash] = tx
	}

	return nil
}

func (s *MemoryStore) Get(height uint32) (*Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	block, ok := s.blocks[height]
	if !ok {
		return nil, fmt.Errorf("block at height %d not found", height)
	}
	return block, nil
}

func (s *MemoryStore) GetByHash(hash types.Hash) (*Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	block, ok := s.blockByHash[hash]
	if !ok {
		return nil, fmt.Errorf("block with hash %s not found", hash)
	}
	return block, nil
}

func (s *MemoryStore) GetTxByHash(hash types.Hash) (*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, ok := s.txByHash[hash]
	if !ok {
		return nil, fmt.Errorf("transaction with hash %s not found", hash)
	}
	return tx, nil
}
