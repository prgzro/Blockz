package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/types"
)

type TXBroadcaster interface {
	BroadcastTransaction(*core.Transaction) error
}

// RPCServer provides a JSON-RPC 2.0 interface for the blockchain.
type RPCServer struct {
	chain       *core.Blockchain
	broadcaster TXBroadcaster
}

// JSONRPCRequest represents a standard JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      any               `json:"id"`
}

// JSONRPCResponse represents a standard JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
	ID      any    `json:"id"`
}

// NewRPCServer creates a new RPC server instance.
func NewRPCServer(chain *core.Blockchain, broadcaster TXBroadcaster) *RPCServer {
	return &RPCServer{
		chain:       chain,
		broadcaster: broadcaster,
	}
}

// Start launches the HTTP server on the given address.
func (s *RPCServer) Start(addr string) error {
	http.HandleFunc("/", s.handleRequest)
	return http.ListenAndServe(addr, nil)
}

func (s *RPCServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, nil, -32700, "Parse error")
		return
	}

	var (
		result any
		err    error
	)

	switch req.Method {
	case "eth_blockNumber":
		result = s.handleBlockNumber()
	case "eth_getBalance":
		result, err = s.handleGetBalance(req.Params)
	case "eth_getBlockByNumber":
		result, err = s.handleGetBlockByNumber(req.Params)
	case "eth_sendRawTransaction":
		result, err = s.handleSendRawTransaction(req.Params)
	case "net_version":
		result = "100" // Example network ID
	case "eth_chainId":
		result = "0x64" // Example chain ID (100 in hex)
	default:
		s.writeError(w, req.ID, -32601, "Method not found")
		return
	}

	if err != nil {
		s.writeError(w, req.ID, -32000, err.Error())
		return
	}

	s.writeResult(w, req.ID, result)
}

func (s *RPCServer) handleBlockNumber() string {
	return fmt.Sprintf("0x%x", s.chain.Height())
}

func (s *RPCServer) handleGetBalance(params []json.RawMessage) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("missing address parameter")
	}

	var addrStr string
	if err := json.Unmarshal(params[0], &addrStr); err != nil {
		return nil, err
	}

	addrBytes, err := hex.DecodeString(addrStr[2:]) // strip 0x
	if err != nil {
		return nil, err
	}

	addr := types.AddressFromBytes(addrBytes)
	balance := s.chain.WorldState.GetBalance(addr)

	return fmt.Sprintf("0x%x", balance), nil
}

func (s *RPCServer) handleGetBlockByNumber(params []json.RawMessage) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("missing block number parameter")
	}

	var numStr string
	if err := json.Unmarshal(params[0], &numStr); err != nil {
		return nil, err
	}

	var height uint64
	if numStr == "latest" {
		height = uint64(s.chain.Height())
	} else {
		// handle 0x... hex conversion
		fmt.Sscanf(numStr, "0x%x", &height)
	}

	header, err := s.chain.GetHeader(uint32(height))
	if err != nil {
		return nil, err
	}

	// In a real RPC, we'd return the full block object.
	// For simplicity, we'll return the header for now.
	return header, nil
}

func (s *RPCServer) handleSendRawTransaction(params []json.RawMessage) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("missing transaction data parameter")
	}

	var rawTx string
	if err := json.Unmarshal(params[0], &rawTx); err != nil {
		return nil, err
	}

	rawTxBytes, err := hex.DecodeString(rawTx[2:]) // strip 0x
	if err != nil {
		return nil, err
	}

	tx := new(core.Transaction)
	if err := tx.Decode(core.NewGobTxDecoder(bytes.NewReader(rawTxBytes))); err != nil {
		return nil, err
	}

	if err := s.broadcaster.BroadcastTransaction(tx); err != nil {
		return nil, err
	}

	txHash := sha256.Sum256(rawTxBytes)
	return fmt.Sprintf("0x%s", hex.EncodeToString(txHash[:])), nil
}

func (s *RPCServer) writeResult(w http.ResponseWriter, id any, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *RPCServer) writeError(w http.ResponseWriter, id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: map[string]any{
			"code":    code,
			"message": message,
		},
		ID: id,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
