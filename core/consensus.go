package core

import (
	"github.com/prgzro/Blockz/types"
)

// ConsensusType identifies the consensus mechanism being used.
type ConsensusType int

const (
	PoW ConsensusType = iota // Proof of Work
	PoS                      // Proof of Stake
)

// ConsensusEngine is the interface that all consensus mechanisms must implement.
type ConsensusEngine interface {
	// Type returns the consensus type.
	Type() ConsensusType

	// VerifyHeader checks if the header satisfies the consensus rules.
	VerifyHeader(header *Header) error

	// Author returns the address of the account that produced the block.
	Author(header *Header) (types.Address, error)

	// Prepare initializes the consensus fields of a header.
	Prepare(chain BlockchainReader, header *Header) error

	// Seal finalizes the block by finding the consensus proof.
	Seal(chain BlockchainReader, block *Block) error
}

// BlockchainReader is a subset of the Blockchain interface needed for consensus.
type BlockchainReader interface {
	Height() uint32
	GetHeader(height uint32) (*Header, error)
}
