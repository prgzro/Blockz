package core

import (
	"fmt"

	"github.com/prgzro/Blockz/crypto"
	"github.com/prgzro/Blockz/types"
)

type Transaction struct {
	Data      []byte            // Data is the payload (contract bytecode or call data)
	To        types.Address     // Recipient address (zero address = contract creation)
	Value     uint64            // Amount of native currency to transfer
	Nonce     uint64            // Sender's nonce — prevents replay attacks
	GasLimit  uint64            // Maximum gas units this tx is willing to consume
	GasPrice  uint64            // Price per gas unit in native currency
	From      crypto.PublicKey  // PublicKey of the signer (set during signing)
	Signature *crypto.Signature // ECDSA signature of the transaction

	// cached version of the Tx Data Hash
	hash types.Hash
	// firstSeen is the timestamp of when this tx is first seen locally
	firstSeen int64
}

func NewTransaction(data []byte) *Transaction {
	return &Transaction{
		Data: data,
	}
}

func (tx *Transaction) Hash(hasher Hasher[*Transaction]) types.Hash {
	if tx.hash.IsZero() {
		tx.hash = hasher.Hash(tx)
	}
	return tx.hash
}

func (tx *Transaction) Sign(privKey crypto.PrivateKey) error { // Sign the transaction with the given private key
	sig, err := privKey.Sign(tx.Data) // Sign the transaction data
	if err != nil {
		return err
	}

	tx.From = privKey.PublicKey() // Set the public key
	tx.Signature = sig            // Set the signature

	return nil
}

func (tx *Transaction) Verify() error { // Verify the transaction's signature
	if tx.Signature == nil { // Check if the signature is nil
		return fmt.Errorf("transaction has no signature") // Return error if signature is nil
	}

	if !tx.Signature.Verify(tx.From, tx.Data) { // Verify the signature
		return fmt.Errorf("invalid transaction signature") // Return error if signature is invalid
	}
	return nil // Return nil if signature is valid
}

func (tx *Transaction) Decode(dec Decoder[*Transaction]) error {
	return dec.Decode(tx)
}

func (tx *Transaction) Encode(enc Encoder[*Transaction]) error {
	return enc.Encode(tx)
}

func (tx *Transaction) SetFirstSeen(t int64) {
	tx.firstSeen = t
}

func (tx *Transaction) FirstSeen() int64 {
	return tx.firstSeen
}
