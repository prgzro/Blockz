package consensus

import (
	"math/big"
)

// ForkChoiceRule defines how to select the canonical chain when multiple forks exist.
type ForkChoiceRule interface {
	// Compare returns 1 if chain A is preferred, -1 if chain B is preferred, 0 if equal.
	Compare(chainA, chainB ChainMetadata) int
}

// ChainMetadata holds the metrics used to evaluate a chain's preference.
type ChainMetadata struct {
	Height               uint32
	CumulativeDifficulty *big.Int // Used for PoW
	TotalStake           *big.Int // Used for PoS
}

// LongestChainRule is the standard fork choice rule for basic chains.
type LongestChainRule struct{}

func (r LongestChainRule) Compare(a, b ChainMetadata) int {
	if a.Height > b.Height {
		return 1
	}
	if b.Height > a.Height {
		return -1
	}
	return 0
}

// HeaviestChainRule is the fork choice rule for PoW (highest total difficulty).
type HeaviestChainRule struct{}

func (r HeaviestChainRule) Compare(a, b ChainMetadata) int {
	cmp := a.CumulativeDifficulty.Cmp(b.CumulativeDifficulty)
	if cmp != 0 {
		return cmp
	}
	// Tie-break with height
	if a.Height > b.Height {
		return 1
	}
	if b.Height > a.Height {
		return -1
	}
	return 0
}
