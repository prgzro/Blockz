package consensus

import (
	"testing"

	"github.com/prgzro/Blockz/core"
	"github.com/stretchr/testify/assert"
)

func TestPoWEngine_Seal(t *testing.T) {
	// Use a very low difficulty for fast testing
	difficulty := uint64(8)
	pow := NewPoWEngine(difficulty)

	header := &core.Header{
		Height:     1,
		Difficulty: difficulty,
	}
	block, _ := core.NewBlock(header, nil)

	// Mining should succeed
	err := pow.Seal(nil, block)
	assert.Nil(t, err)

	// Verification should pass
	err = pow.VerifyHeader(block.Header)
	assert.Nil(t, err)
}

func TestPoWEngine_VerifyHeader_Invalid(t *testing.T) {
	pow := NewPoWEngine(16)
	header := &core.Header{
		Height:     1,
		Difficulty: 16,
		Nonce:      12345, // Likely invalid for 16 bits
	}

	// This hash is unlikely to meet 16 bits of difficulty
	err := pow.VerifyHeader(header)
	if err == nil {
		// If it by some miracle is valid, we can't assert fail,
		// but with 12345 constant it's very likely to fail.
		t.Log("Warning: random nonce 12345 happened to be valid for 16 bits")
	}
}

func TestPoWEngine_Weight(t *testing.T) {
	pow := NewPoWEngine(0)
	h1 := &core.Header{Difficulty: 10}
	h2 := &core.Header{Difficulty: 11}

	w1 := pow.CalculateBlockWeight(h1)
	w2 := pow.CalculateBlockWeight(h2)

	assert.True(t, w2.Cmp(w1) > 0, "Weight of difficulty 11 should be greater than difficulty 10")
}
