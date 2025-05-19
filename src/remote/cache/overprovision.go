package cache

import (
	"log"
	"math"
	"sort"
	//"time"
)

// BaselineEstimator tracks a lower-percentile baseline and detects oversaturation
// based on mean-of-squares in a sliding window of completion times, using incremental updates.
type BaselineEstimator struct {
	windowSize int
	alphaDown  float64  // EMA rate for downward adjustments
	alphaUp    float64  // EMA rate for upward adjustments
	beta       float64  // slowdown tolerance factor
	percentile float64  // e.g. 0.10 for 10th percentile

	window []float64  // circular buffer of recent S samples
	sorted []float64  // circular buffer of recent S samples
	pos    int        // next insert position
	nFull  int        // count of samples inserted (capped at windowSize)
	sumSq  float64    // running sum of squares of samples

	SEst float64  // current EMA baseline
}

// NewBaselineEstimator creates an estimator with the given parameters:
// - windowSize: number of recent samples to keep
// - alphaDown:  EMA weight for downward adjustments (e.g. 0.2)
// - alphaUp:    EMA weight for upward adjustments (e.g. 0.05)
// - beta:       allowed slowdown multiplier (e.g. 1.2)
// - percentile: lower-tail percentile for baseline (e.g. 0.10)
func (e *BaselineEstimator) SetBaselineEstimator(windowSize int, alphaDown, alphaUp, beta, percentile float64) {
	e.windowSize = windowSize
	e.alphaDown =  alphaDown
	e.alphaUp =    alphaUp
	e.beta =       beta
	e.percentile = percentile
	e.window =     make([]float64, windowSize)
	e.sorted =     make([]float64, windowSize)
}

// AddSample adds a new observed completion time S (in seconds) and returns:
// - baseline:  the baseline used for threshold (pre-update)
// - recentLow: the computed lower-percentile of the window (only when full)
// - meanSq:    the mean of S^2 over the window (only when full)
// - oversat:   true if meanSq > (beta * baseline)^2 indicating overload (only when full)
func (e *BaselineEstimator) AddSample(S float64) bool {
	// Remove oldest sample from sumSq when buffer is full
	if e.nFull == e.windowSize {
		old := e.window[e.pos]
		e.sumSq -= old * old
	} else {
		// Still filling the buffer
		e.nFull++
	}

	// Insert new sample and update sumSq
	e.window[e.pos] = S
	e.sumSq += S * S

	// Advance position circularly
	e.pos = (e.pos + 1) % e.windowSize

	// Before buffer is full, initialize baseline and skip detection
	if e.nFull < e.windowSize {
        e.SEst += S / float64(e.windowSize)
        log.Printf("(oversat) not enough samples, SESt: %.4f", e.SEst)
		return true
	}

	// Buffer is full: compute window stats
	n := e.windowSize

	// Compute lower-percentile sample for baseline update
	copy(e.sorted, e.window)
	sort.Float64s(e.sorted)
	idx := int(math.Ceil(e.percentile*float64(n))) - 1
	if idx < 0 {
		idx = 0
	} else if idx >= n {
		idx = n - 1
	}
    recentLow := e.sorted[idx]

	// Compute mean of squares incrementally
    meanSq := e.sumSq / float64(n)

	// Use current (pre-update) baseline for threshold
    baseline := e.SEst
	threshold := math.Pow(e.beta*baseline, 2)
    log.Printf("(oversat) baseline: %.4f threshold: %.4f meanSq: %.4f", baseline, threshold, meanSq)
    oversat := meanSq > threshold

	// Now update EMA baseline based on recentLow
	if e.SEst == 0 {
		e.SEst = recentLow
	} else {
		var alpha float64
		if recentLow < e.SEst {
			alpha = e.alphaDown
		} else {
			alpha = e.alphaUp
		}
		e.SEst = alpha*recentLow + (1-alpha)*e.SEst
	}

	return oversat
}

