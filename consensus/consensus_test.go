package consensus

import (
	"math/big"
	"testing"
	"time"

	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/types"
	"github.com/stretchr/testify/assert"
)

func TestPoW(t *testing.T) {
	// Low difficulty for fast testing (4 bits = 1 leading hex zero)
	pow := NewPoWEngine(4)

	header := &core.Header{
		Version:   1,
		Height:    10,
		Timestamp: time.Now().UnixNano(),
	}
	block, _ := core.NewBlock(header, nil)
	// Prepare the block (sets difficulty)
	err := pow.Prepare(nil, block.Header)
	assert.Nil(t, err)

	// Should fail verification before sealing
	err = pow.VerifyHeader(block.Header)
	assert.NotNil(t, err)

	// Seal (Mine) the block
	err = pow.Seal(nil, block)
	assert.Nil(t, err)

	// Should pass verification after sealing
	err = pow.VerifyHeader(block.Header)
	assert.Nil(t, err)

	assert.Equal(t, uint64(4), block.Header.Difficulty)
}

func TestPoS(t *testing.T) {
	pos := NewPoSEngine()
	addr1 := types.AddressFromBytes(types.RandomBytes(20))
	addr2 := types.AddressFromBytes(types.RandomBytes(20))

	pos.AddValidator(addr1, 100)
	pos.AddValidator(addr2, 200)

	// Test deterministic selection
	p0, _ := pos.SelectProposer(0)
	p1, _ := pos.SelectProposer(1)
	p2, _ := pos.SelectProposer(2)

	assert.Equal(t, addr1, p0)
	assert.Equal(t, addr2, p1)
	assert.Equal(t, addr1, p2)

	header := &core.Header{
		Height:   0,
		Coinbase: addr1,
	}
	assert.Nil(t, pos.VerifyHeader(header))

	headerBad := &core.Header{
		Height:   0,
		Coinbase: types.AddressFromBytes(types.RandomBytes(20)),
	}
	assert.NotNil(t, pos.VerifyHeader(headerBad))
}

func TestHeaviestChain(t *testing.T) {
	rule := HeaviestChainRule{}

	c1 := ChainMetadata{
		Height:               10,
		CumulativeDifficulty: big.NewInt(100),
	}
	c2 := ChainMetadata{
		Height:               11,
		CumulativeDifficulty: big.NewInt(90),
	}

	// c1 is heavier despite being shorter
	assert.Equal(t, 1, rule.Compare(c1, c2))
}
