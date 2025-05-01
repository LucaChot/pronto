package scheduler

import (
	"errors"
	"math"
	"sync/atomic"
	"time"
)

// Node represents a compute node with a smoothed willingness signal and a token bucket for burst throttling.
type Node struct {
    ID         string
    weight     atomic.Uint64   // EWMA-smoothed willingness signal
    alpha      float64   // EWMA smoothing factor (0 < alpha <= 1)
    tokens     float64   // current number of tokens
    rate       float64   // refill rate (tokens per second)
    capacity   float64   // maximum tokens in bucket
    lastRefill time.Time // last time tokens were refilled
}

// NewNode constructs a Node with given parameters.
func NewNode(id string, alpha, rate, capacity float64) *Node {
    now := time.Now()
    n := Node{
        ID:         id,
        alpha:      alpha,
        tokens:     capacity,
        rate:       rate,
        capacity:   capacity,
        lastRefill: now,
    }
    n.weight.Store(math.Float64bits(0.0))
    return &n
}

// UpdateWillingness updates the node's EWMA-smoothed willingness signal.
func (n *Node) UpdateWillingness(newSignal float64) {
    weight := n.weight.Load()
    newWeight := n.alpha*newSignal + (1-n.alpha)* math.Float64frombits(weight)
    n.weight.Store(math.Float64bits(newWeight))
}

// refill replenishes tokens based on elapsed time.
func (n *Node) refill(now time.Time) {
    elapsed := now.Sub(n.lastRefill).Seconds()
    n.tokens += elapsed * n.rate
    if n.tokens > n.capacity {
        n.tokens = n.capacity
    }
    n.lastRefill = now
}

// AcquireToken attempts to consume one token; returns true if successful.
func (n *Node) AcquireToken(now time.Time) bool {
    // Refill first
    n.refill(now)

    if n.tokens >= 1 {
        n.tokens--
        return true
    }
    return false
}

func (n *Node) GetWeight() float64 {
    return math.Float64frombits(n.weight.Load())
}

// SchedulePod selects a node for the next pod using:
// 1) Power-of-two sampling with weighted probabilities (smoothed willingness)
// 2) Burst-aware token bucket throttling
// Returns the chosen node or an error if no node is available.
func (ctl *Scheduler) SchedulePod() (*Node, error) {
    now := time.Now()

    // Compute total weight
    totalWeight := 0.0
    for _, n := range ctl.nodes {
        totalWeight += n.GetWeight()
    }

    if totalWeight <= 0 {
        return nil, errors.New("(sched) no available nodes (all weights are zero)")
    }

    // Helper to sample one node proportionally to weight
    sample := func() *Node {
        threshold := ctl.rand.Float64() * totalWeight
        sum := 0.0
        for _, n := range ctl.nodes {
            w := n.GetWeight()
            sum += w
            if sum >= threshold {
                return n
            }
        }
        // fallback in case of rounding
        return ctl.nodes[len(ctl.nodes)-1]
    }

    // Power-of-two sampling
    n1 := sample()
    n2 := sample()

    // Choose candidate with higher smoothed weight
    var chosen, other *Node
    w1 := n1.GetWeight()
    w2 := n2.GetWeight()

    if w1 >= w2 {
        chosen, other = n1, n2
    } else {
        chosen, other = n2, n1
    }

    // Try to acquire token on chosen node
    if chosen.AcquireToken(now) {
        return chosen, nil
    }
    // Fallback: try the other candidate
    if other.AcquireToken(now) {
        return other, nil
    }

    return nil, errors.New("(sched) all candidate nodes are currently rate-limited")
}
