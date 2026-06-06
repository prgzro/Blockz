package core

import (
	"testing"

	"github.com/prgzro/Blockz/types"
	"github.com/stretchr/testify/assert"
)

func TestAccount(t *testing.T) {
	acc := NewAccount(100)
	assert.Equal(t, uint64(100), acc.Balance)
	assert.Equal(t, uint64(0), acc.Nonce)
	assert.False(t, acc.IsContract())

	data, err := acc.Encode()
	assert.Nil(t, err)

	acc2, err := DecodeAccount(data)
	assert.Nil(t, err)
	assert.Equal(t, acc.Balance, acc2.Balance)
}

func TestWorldState(t *testing.T) {
	ws := NewWorldState()
	addr1 := types.AddressFromBytes(types.RandomBytes(20))
	addr2 := types.AddressFromBytes(types.RandomBytes(20))

	ws.AddBalance(addr1, 1000)
	assert.Equal(t, uint64(1000), ws.GetBalance(addr1))

	err := ws.Transfer(addr1, addr2, 400)
	assert.Nil(t, err)
	assert.Equal(t, uint64(600), ws.GetBalance(addr1))
	assert.Equal(t, uint64(400), ws.GetBalance(addr2))

	// Test insufficient funds
	err = ws.Transfer(addr1, addr2, 1000)
	assert.NotNil(t, err)

	// Test nonce
	assert.Equal(t, uint64(0), ws.GetNonce(addr1))
	ws.IncrementNonce(addr1)
	assert.Equal(t, uint64(1), ws.GetNonce(addr1))
}

func TestStateRootDeterministic(t *testing.T) {
	ws1 := NewWorldState()
	ws2 := NewWorldState()

	addr1 := types.AddressFromBytes(make([]byte, 20))
	addr1[0] = 0x1
	addr2 := types.AddressFromBytes(make([]byte, 20))
	addr2[0] = 0x2

	// Add in different order
	ws1.AddBalance(addr1, 100)
	ws1.AddBalance(addr2, 200)

	ws2.AddBalance(addr2, 200)
	ws2.AddBalance(addr1, 100)

	assert.Equal(t, ws1.StateRoot(), ws2.StateRoot())
}

func TestWorldStateCopy(t *testing.T) {
	ws := NewWorldState()
	addr := types.AddressFromBytes(types.RandomBytes(20))
	ws.AddBalance(addr, 500)

	wsCopy := ws.Copy()
	assert.Equal(t, ws.GetBalance(addr), wsCopy.GetBalance(addr))

	wsCopy.AddBalance(addr, 500)
	assert.Equal(t, uint64(500), ws.GetBalance(addr))
	assert.Equal(t, uint64(1000), wsCopy.GetBalance(addr))
}
