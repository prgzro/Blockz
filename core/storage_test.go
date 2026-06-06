package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStorage()
	block := randomBlock(t, 1, [32]byte{})

	err := store.Put(block)
	assert.Nil(t, err)

	decoded, err := store.Get(1)
	assert.Nil(t, err)
	assert.Equal(t, block.Header.Height, decoded.Header.Height)

	decodedByHash, err := store.GetByHash(block.Hash(BlockHasher{}))
	assert.Nil(t, err)
	assert.Equal(t, block.Header.Height, decodedByHash.Header.Height)
}

func TestLevelDBStore(t *testing.T) {
	dbPath := "./test_db"
	defer os.RemoveAll(dbPath)

	store, err := NewLevelDBStore(dbPath)
	assert.Nil(t, err)
	defer store.Close()

	block := randomBlock(t, 1, [32]byte{})

	err = store.Put(block)
	assert.Nil(t, err)

	decoded, err := store.Get(1)
	assert.Nil(t, err)
	assert.Equal(t, block.Header.Height, decoded.Header.Height)

	// Test persistence by closing and reopening
	store.Close()
	store, err = NewLevelDBStore(dbPath)
	assert.Nil(t, err)

	decodedAfterRestart, err := store.Get(1)
	assert.Nil(t, err)
	assert.Equal(t, block.Header.Height, decodedAfterRestart.Header.Height)
}
