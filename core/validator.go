package core

import "fmt"

type Validator interface {
	ValidateBlock(*Block) error
	ValidateBlockLocked(*Block) error
}

type BlockValidator struct {
	bc *Blockchain
}

func NewBlockValidator(bc *Blockchain) *BlockValidator {
	return &BlockValidator{bc: bc}
}

func (v *BlockValidator) ValidateBlock(b *Block) error {
	return v.ValidateBlockLocked(b)
}

func (v *BlockValidator) ValidateBlockLocked(b *Block) error {
	if v.bc.hasBlockLocked(b.Header.Height) {
		return fmt.Errorf("chain already contains block (%d) with hash (%s)", b.Header.Height, b.Hash(BlockHasher{}))
	}

	if b.Header.Height != v.bc.heightLocked()+1 {
		return fmt.Errorf("Block (%s) with height(%d) is too high => current height (%d)", b.Hash(BlockHasher{}), b.Header.Height, v.bc.heightLocked())
	}

	prevHeader, err := v.bc.getHeaderLocked(b.Header.Height - 1)
	if err != nil {
		return err
	}

	hash := BlockHasher{}.Hash(prevHeader)
	if hash != b.Header.PrevBlockHash {
		return fmt.Errorf("the hash of the previous block (%s) is invalid ", b.Header.PrevBlockHash)
	}

	// Consensus specific verification
	if v.bc.consensus != nil {
		if err := v.bc.consensus.VerifyHeader(b.Header); err != nil {
			return err
		}
	}

	if err := b.Verify(); err != nil {
		return err
	}
	return nil
}
