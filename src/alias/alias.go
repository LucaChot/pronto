package alias

import (
	"errors"
	"math/rand/v2" // Using the newer math/rand/v2 package
)


// AliasTable holds the precomputed tables for the Alias Method.
type AliasTable struct {
	n     int       // Number of outcomes
	prob  []float64 // Probability table
	alias []int     // Alias table
}

// New creates a new AliasTable from a slice of weights.
// The weights must be non-negative and sum to a positive value.
func New(weights []float64) (*AliasTable, error) {
	n := len(weights)
	if n == 0 {
		return nil, errors.New("weights slice cannot be empty")
	}

	// Calculate sum and check for negative weights
	var sum float64
	for _, w := range weights {
		if w < 0 {
			return nil, errors.New("weights cannot be negative")
		}
		sum += w
	}

	if sum <= 0 {
		return nil, errors.New("sum of weights must be positive")
	}

	// Initialize tables and normalize probabilities
	prob := make([]float64, n)
	alias := make([]int, n)
	normProb := make([]float64, n)

	// Create worklists for items with probabilities smaller or larger than 1.0
	small := make([]int, 0, n)
	large := make([]int, 0, n)

    // Normalize probabilities so the average probability is 1.0
	for i, p := range weights {
        normP := p * float64(n) / sum
        normProb[i] = normP
		if normP < 1.0 {
			small = append(small, i)
		} else {
			large = append(large, i)
		}
	}

	// Populate the probability and alias tables
	for len(small) > 0 && len(large) > 0 {
		s := small[len(small)-1]
		small = small[:len(small)-1]
		l := large[len(large)-1]
		large = large[:len(large)-1]

		prob[s] = normProb[s]
		alias[s] = l

        // Update the probability of the large item and reclassify the large
        // item based on its updated probability
		normProb[l] = normProb[l] - (1.0 - normProb[s])
        if normProb[l] < 1.0 {
			small = append(small, l)
		} else {
			large = append(large, l)
		}
	}

	// Handle remaining items (due to floating point inaccuracies)
	// Any remaining items should have probability 1.0
    for _, idx := range large {
		prob[idx] = 1.0
	}
    for _, idx := range small {
		prob[idx] = 1.0
	}

	return &AliasTable{
		n:     n,
		prob:  prob,
		alias: alias,
	}, nil
}

// Sample returns a randomly sampled index based on the original weights.
func (at *AliasTable) Sample() int {
	// 1. Choose a column (bin) uniformly at random
	i := rand.IntN(at.n)

	// 2. Flip a biased coin for that column
	if rand.Float64() < at.prob[i] {
		// Return the primary item in the bin
		return i
	}
	// Return the alias item in the bin
	return at.alias[i]
}
