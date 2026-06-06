package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/log"
	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/types"
	"github.com/stretchr/testify/assert"
)

type mockBroadcaster struct{}

func (m *mockBroadcaster) BroadcastTransaction(tx *core.Transaction) error {
	return nil
}

func TestRPCServer_BlockNumber(t *testing.T) {
	header := &core.Header{
		Version:   1,
		Height:    0,
		Timestamp: 100,
	}
	b, _ := core.NewBlock(header, nil)
	bc, _ := core.NewBlockchain(log.NewNopLogger(), b, core.NewMemoryStorage())

	server := NewRPCServer(bc, &mockBroadcaster{})

	reqBody := `{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":1}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	server.handleRequest(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp JSONRPCResponse
	json.NewDecoder(rr.Body).Decode(&resp)

	assert.Equal(t, "0x0", resp.Result)
}

func TestRPCServer_GetBalance(t *testing.T) {
	header := &core.Header{
		Version:   1,
		Height:    0,
		Timestamp: 100,
	}
	b, _ := core.NewBlock(header, nil)
	bc, _ := core.NewBlockchain(log.NewNopLogger(), b, core.NewMemoryStorage())

	addr := types.RandomAddress()
	bc.WorldState.AddBalance(addr, 1000)

	server := NewRPCServer(bc, &mockBroadcaster{})

	reqBody := fmt.Sprintf(`{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["0x%s"], "id":1}`, addr.String())
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	server.handleRequest(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp JSONRPCResponse
	json.NewDecoder(rr.Body).Decode(&resp)

	assert.Equal(t, "0x3e8", resp.Result) // 1000 = 0x3e8
}
