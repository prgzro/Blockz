package consensus

import (
	"fmt"
	"sync"

	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/types"
)

// PoSEngine implements the Proof of Stake consensus mechanism.
type PoSEngine struct {
	mu          sync.RWMutex
	validators  map[types.Address]uint64 // address -> stake amount
	activeSet   []types.Address
	seenHeaders map[uint32]map[types.Address]types.Hash // height -> validator -> block hash
}

// NewPoSEngine creates a new PoS consensus engine.
func NewPoSEngine() *PoSEngine {
	return &PoSEngine{
		validators:  make(map[types.Address]uint64),
		seenHeaders: make(map[uint32]map[types.Address]types.Hash),
	}
}

func (pos *PoSEngine) Type() core.ConsensusType {
	return core.PoS
}

// AddValidator adds an account to the validator set with a given stake.
func (pos *PoSEngine) AddValidator(addr types.Address, stake uint64) {
	pos.mu.Lock()
	defer pos.mu.Unlock()
	pos.validators[addr] = stake
	pos.activeSet = append(pos.activeSet, addr)
}

func (pos *PoSEngine) VerifyHeader(header *core.Header) error {
	pos.mu.Lock()
	defer pos.mu.Unlock()

	// 1. Proposer check
	expected, err := pos.selectProposerLocked(header.Height)
	if err != nil {
		return err
	}
	if expected != header.Coinbase {
		return fmt.Errorf("unexpected proposer %s at height %d, expected %s", header.Coinbase, header.Height, expected)
	}

	// 2. Double-signing check (sliding window or in-memory for recent blocks)
	height := header.Height
	if pos.seenHeaders[height] == nil {
		pos.seenHeaders[height] = make(map[types.Address]types.Hash)
	}

	headerHash := header.Hash()
	if existingHash, seen := pos.seenHeaders[height][header.Coinbase]; seen {
		if existingHash != headerHash {
			// DOUBLE SIGNING DETECTED!
			// In a production system, we'd trigger slashing here or via a dedicated evidence path.
			// For simplicity, we'll return an error and mark for slashing.
			return fmt.Errorf("validator %s double-signed at height %d", header.Coinbase, height)
		}
	}

	pos.seenHeaders[height][header.Coinbase] = headerHash

	// 3. Finality check (TODO: 2/3 signatures)
	return nil
}

func (pos *PoSEngine) Slash(addr types.Address, reason string) {
	pos.mu.Lock()
	defer pos.mu.Unlock()

	if stake, ok := pos.validators[addr]; ok {
		// Slash 50% of stake
		pos.validators[addr] = stake / 2

		// Remove from active set
		newSet := []types.Address{}
		for _, a := range pos.activeSet {
			if a != addr {
				newSet = append(newSet, a)
			}
		}
		pos.activeSet = newSet
	}
}

func (pos *PoSEngine) Author(header *core.Header) (types.Address, error) {
	return header.Coinbase, nil
}

func (pos *PoSEngine) Prepare(chain core.BlockchainReader, header *core.Header) error {
	pos.mu.Lock()
	defer pos.mu.Unlock()

	// Selection logic: who is the proposer for this height?
	proposer, err := pos.selectProposerLocked(header.Height)
	if err != nil {
		return err
	}
	header.Coinbase = proposer
	return nil
}

func (pos *PoSEngine) Seal(chain core.BlockchainReader, block *core.Block) error {
	// In PoS, sealing is just signing the block.
	// The proposer is already set in Prepare.
	// Actual signing happens in the server's validator loop.
	return nil
}

// SelectProposer picks a validator to produce a block at a given height.
// Uses a simple weighted selection (conceptually).
func (pos *PoSEngine) SelectProposer(height uint32) (types.Address, error) {
	pos.mu.RLock()
	defer pos.mu.RUnlock()

	return pos.selectProposerLocked(height)
}

func (pos *PoSEngine) selectProposerLocked(height uint32) (types.Address, error) {
	if len(pos.activeSet) == 0 {
		return types.Address{}, fmt.Errorf("no active validators")
	}

	// Simplified: deterministic selection based on height
	index := int(height) % len(pos.activeSet)
	return pos.activeSet[index], nil
}

// Slashing: Detect double signing
func (pos *PoSEngine) CheckSlashing(h1, h2 *core.Header) *types.Address {
	if h1.Height == h2.Height && h1.Coinbase == h2.Coinbase {
		// Same validator signed two different blocks at the same height
		return &h1.Coinbase
	}
	return nil
}
