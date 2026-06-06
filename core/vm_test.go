package core

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/prgzro/Blockz/types"
	"github.com/stretchr/testify/assert"
)

func TestEVM_BasicExecution(t *testing.T) {
	state := NewWorldState()
	contractAddr := types.RandomAddress()
	callerAddr := types.RandomAddress()

	// Bytecode: PUSH1 0x05, PUSH1 0x01, ADD, PUSH1 0x00, SSTORE
	// Stores 6 at storage key 0
	code := []byte{
		0x60, 0x05, // PUSH1 5
		0x60, 0x01, // PUSH1 1
		0x01,       // ADD
		0x60, 0x00, // PUSH1 0
		0x55, // SSTORE
	}

	evm := NewEVM(code, nil, state, contractAddr, callerAddr, big.NewInt(0), 100000)
	_, err := evm.Run()
	assert.NoError(t, err)

	// Fetch storage key 0 (SSTORE uses sha256 for key lookup internally)
	padded := make([]byte, 32)
	keyHash := sha256.Sum256(padded)
	valBytes, err := state.GetAccountState(contractAddr, types.Hash(keyHash))
	assert.NoError(t, err)
	// Alternatively, verify gas used
	assert.True(t, evm.GasUsed() > 0)
	assert.NotNil(t, valBytes)
}
