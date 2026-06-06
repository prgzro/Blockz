package core

import (
	"io"
	"testing"

	"github.com/go-kit/log"
	"github.com/prgzro/Blockz/types"
	"github.com/stretchr/testify/assert"
)

func TestAddBlock(t *testing.T) {
	bc := NewBlockchainWithGenesis(t)

	lenBlocks := 20
	for i := 0; i < lenBlocks; i++ {
		block := randomBlock(t, uint32(i+1), getPrevBlockHash(t, bc, uint32(i+1)))
		assert.Nil(t, bc.AddBlock(block))

	}
	assert.Equal(t, bc.Height(), uint32(lenBlocks))
	assert.Equal(t, len(bc.Headers), lenBlocks+1) // +1 for genesis block
	assert.NotNil(t, bc.AddBlock(randomBlock(t, 89, types.Hash{})))
}

func TestNewBlockchain(t *testing.T) {
	bc := NewBlockchainWithGenesis(t)
	assert.NotNil(t, bc.validator)
	assert.Equal(t, bc.Height(), uint32(0))
}
func TestHasBlock(t *testing.T) {
	bc := NewBlockchainWithGenesis(t)
	assert.True(t, bc.HasBlock(0))
	assert.False(t, bc.HasBlock(1))
	assert.False(t, bc.HasBlock(100))
}

func TestGetHeader(t *testing.T) {
	bc := NewBlockchainWithGenesis(t)
	lenBlocks := 20

	for i := 0; i < lenBlocks; i++ {
		block := randomBlock(t, uint32(i+1), getPrevBlockHash(t, bc, uint32(i+1)))
		assert.Nil(t, bc.AddBlock(block))
		header, err := bc.GetHeader(block.Header.Height)
		assert.Nil(t, err)
		assert.Equal(t, header, block.Header)
	}
}

func TestAddBlockToHigh(t *testing.T) {
	bc := NewBlockchainWithGenesis(t)

	assert.Nil(t, bc.AddBlock(randomBlock(t, 1, getPrevBlockHash(t, bc, uint32(1)))))
	assert.NotNil(t, bc.AddBlock(randomBlock(t, 3, types.Hash{})))
}

func NewBlockchainWithGenesis(t *testing.T) *Blockchain {
	logger := log.NewLogfmtLogger(io.Discard)
	bc, err := NewBlockchain(logger, randomBlock(t, 0, types.Hash{}), NewMemoryStorage())
	assert.Nil(t, err)
	return bc
}

func getPrevBlockHash(t *testing.T, bc *Blockchain, height uint32) types.Hash {
	prevHeader, err := bc.GetHeader(height - 1)
	assert.Nil(t, err)

	return BlockHasher{}.Hash(prevHeader)
}
