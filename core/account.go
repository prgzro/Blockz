package core

import (
	"bytes"
	"encoding/gob"

	"github.com/prgzro/Blockz/types"
)

type Account struct {
	Nonce       uint64                // Number of transactions sent from this account (prevents replay attacks)
	Balance     uint64                // Account balance in the native currency (wei-equivalent)
	CodeHash    types.Hash            // Hash of the contract bytecode (zero for EOA)
	StorageRoot types.Hash            // Root hash of the account's storage trie (zero for EOA)
	Storage     map[types.Hash][]byte // The account's persistent storage trie (simplified as map)
}

// NewAccount creates a new account with the given balance.
func NewAccount(balance uint64) *Account {
	return &Account{
		Balance: balance,
		Storage: make(map[types.Hash][]byte),
	}
}

func (a *Account) DeepCopy() *Account {
	storage := make(map[types.Hash][]byte)
	for k, v := range a.Storage {
		storage[k] = v
	}
	return &Account{
		Nonce:       a.Nonce,
		Balance:     a.Balance,
		CodeHash:    a.CodeHash,
		StorageRoot: a.StorageRoot,
		Storage:     storage,
	}
}

// IsContract returns true if this account has contract code deployed.
func (a *Account) IsContract() bool {
	return !a.CodeHash.IsZero()
}

// Encode serializes the account to bytes using gob encoding.
func (a *Account) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(a); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeAccount deserializes an account from bytes.
func DecodeAccount(data []byte) (*Account, error) {
	acc := &Account{}
	buf := bytes.NewReader(data)
	if err := gob.NewDecoder(buf).Decode(acc); err != nil {
		return nil, err
	}
	return acc, nil
}
