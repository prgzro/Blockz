package consensus

import (
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/prgzro/Blockz/core"
	"github.com/prgzro/Blockz/types"
)

var (
	// maxTarget is the highest possible difficulty target (easiest to mine).
	// Calculated as 2^256 - 1.
	maxTarget = big.NewInt(1)
)

func init() {
	maxTarget.Lsh(maxTarget, 256)
	maxTarget.Sub(maxTarget, big.NewInt(1))
}

const (
	defaultDifficulty = 16 // Bits of zeros required at the start of the hash
	miningTimeout     = 10 * time.Second
)

// PoWEngine implements the Proof of Work consensus mechanism.
type PoWEngine struct {
	targetDifficulty uint64
}

// NewPoWEngine creates a new PoW consensus engine.
func NewPoWEngine(difficulty uint64) *PoWEngine {
	if difficulty == 0 {
		difficulty = defaultDifficulty
	}
	return &PoWEngine{
		targetDifficulty: difficulty,
	}
}

func (pow *PoWEngine) Type() core.ConsensusType {
	return core.PoW
}

// VerifyHeader checks if the header hash satisfies the difficulty target.
func (pow *PoWEngine) VerifyHeader(header *core.Header) error {
	hash := core.BlockHasher{}.Hash(header)

	// Convert hash to big.Int
	hashInt := new(big.Int).SetBytes(hash[:])

	// Calculate target based on difficulty bits
	// Target = maxTarget >> difficulty
	target := new(big.Int).Lsh(big.NewInt(1), uint(256-header.Difficulty))

	if hashInt.Cmp(target) > 0 {
		return fmt.Errorf("block hash %s does not satisfy difficulty %d", hash, header.Difficulty)
	}

	return nil
}

func (pow *PoWEngine) Author(header *core.Header) (types.Address, error) {
	return header.Coinbase, nil
}

func (pow *PoWEngine) Prepare(chain core.BlockchainReader, header *core.Header) error {
	header.Difficulty = pow.targetDifficulty
	// TODO: implement adaptive difficulty adjustment based on block times of past N blocks
	return nil
}

// Seal performs the mining operation by incrementing the nonce until a valid hash is found.
func (pow *PoWEngine) Seal(chain core.BlockchainReader, block *core.Block) error {
	header := block.Header
	target := new(big.Int).Lsh(big.NewInt(1), uint(256-pow.targetDifficulty))

	start := time.Now()
	var n uint64 = 0

	for n < math.MaxUint64 {
		header.Nonce = n
		hash := core.BlockHasher{}.Hash(header)
		hashInt := new(big.Int).SetBytes(hash[:])

		if hashInt.Cmp(target) <= 0 {
			// Found a valid hash!
			fmt.Printf("\n[PoW] Mined block at height %d with nonce %d in %v\n",
				header.Height, n, time.Since(start))
			return nil
		}
		n++

		// Check for timeout to prevent infinite loop if difficulty is too high for the test environment
		if n%100000 == 0 && time.Since(start) > miningTimeout {
			return fmt.Errorf("mining timed out after %v", miningTimeout)
		}
	}

	return fmt.Errorf("failed to find nonce")
}

// CalculateBlockWeight returns the cumulative difficulty of a chain.
// Used for the fork choice rule: highest cumulative difficulty wins.
func (pow *PoWEngine) CalculateBlockWeight(header *core.Header) *big.Int {
	// Weight = 2^Difficulty
	return new(big.Int).Lsh(big.NewInt(1), uint(header.Difficulty))
}
