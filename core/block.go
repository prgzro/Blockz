package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/prgzro/Blockz/crypto"
	"github.com/prgzro/Blockz/types"
)

/*
*
What is a Block Header?

Every block in a blockchain has two main parts:

Block header — small but very important part that summarizes what’s inside the block.

Block body (or data) — the list of transactions and other details.

So, the block header is like the “ID card” of the block.
It contains only the key information needed to:

# Verify the block’s identity

Link it to the previous block (to form the chain)

Check its integrity and proof of work
*/
type Header struct {
	Version       uint32        // Version of the block
	DataHash      types.Hash    // Hash of the block's data (transactions)
	PrevBlockHash types.Hash    // Hash of the previous block in the chain
	StateRoot     types.Hash    // Root hash of the world state after executing this block
	Height        uint32        // Height of the block in the blockchain
	Timestamp     int64         // Timestamp of when the block was created
	Nonce         uint64        // PoW nonce — miners iterate this to find valid hash
	Difficulty    uint64        // PoW difficulty target for this block
	Coinbase      types.Address // Address of the miner/validator who produced this block
}

func (h *Header) Bytes() []byte {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	enc.Encode(h)

	return buf.Bytes()
}

func (h *Header) Hash() types.Hash {
	return BlockHasher{}.Hash(h)
}

type Block struct {
	Header       *Header           // Pointer to the block header
	Transactions []*Transaction    // List of transactions in the block
	Evidence     []*Header         // Slashing evidence (headers signed by others)
	Validator    crypto.PublicKey  // Public key of the block validator
	Signature    *crypto.Signature // Signature of the block that was created(signed by his privKey) by the validator
	// Cached Version of the header hash
	hash types.Hash
}

func NewBlock(h *Header, txx []*Transaction) (*Block, error) { // Constructor for creating a new Block instance
	return &Block{
		Header:       h,
		Transactions: txx,
	}, nil
}

func NewBlockFromPrevHeader(prevHeader *Header, txx []*Transaction) (*Block, error) {
	dataHash, err := CalculateDataHash(txx, nil)
	if err != nil {
		return nil, err
	}

	header := &Header{
		Version:       1,
		DataHash:      dataHash,
		PrevBlockHash: BlockHasher{}.Hash(prevHeader),
		Height:        prevHeader.Height + 1,
		Timestamp:     time.Now().UnixNano(),
	}
	return NewBlock(header, txx)
}

func (b *Block) AddTransaction(tx *Transaction) {
	b.Transactions = append(b.Transactions, tx)
}

func (b *Block) Sign(privKey crypto.PrivateKey) error { // Method to sign the block using a private key
	sig, err := privKey.Sign(b.Header.Bytes()) // Sign the block's header_data's bytes
	if err != nil {                            // Check for errors during signing
		return err
	}
	b.Validator = privKey.PublicKey() // Set the validator's public key
	b.Signature = sig                 // Set the block's signature
	return nil
}
func (b *Block) Verify() error { // Method to verify the block's signature
	if b.Signature == nil {
		return fmt.Errorf("block has no signature")
	}
	if !b.Signature.Verify(b.Validator, b.Header.Bytes()) { // Verify the signature using the validator's public key
		return fmt.Errorf("block has invalid signature")
	}

	for _, tx := range b.Transactions {
		if err := tx.Verify(); err != nil {
			return err
		}
	}

	dataHash, err := CalculateDataHash(b.Transactions, b.Evidence)
	if err != nil {
		return err
	}
	if dataHash != b.Header.DataHash {
		return fmt.Errorf("block (%s) has an invalid data hash", b.Hash(BlockHasher{}))
	}

	return nil
}

func (b *Block) Decode(dec Decoder[*Block]) error {
	return dec.Decode(b)
}

func (b *Block) Encode(enc Encoder[*Block]) error {
	return enc.Encode(b)
}

func (b *Block) Hash(hasher Hasher[*Header]) types.Hash {
	if b.hash.IsZero() {
		b.hash = hasher.Hash(b.Header)
	}
	return b.hash
}

func CalculateDataHash(txx []*Transaction, evidence []*Header) (hash types.Hash, err error) {
	buf := &bytes.Buffer{}

	for _, tx := range txx {
		if err = tx.Encode(NewGobTxEncoder(buf)); err != nil {
			return
		}
	}
	for _, h := range evidence {
		buf.Write(h.Bytes())
	}
	hash = sha256.Sum256(buf.Bytes())
	return
}
