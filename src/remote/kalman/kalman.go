package kalman

import (
	"log"
	"math"
)

// KalmanFilter implements a 1-D Kalman filter to estimate the average signal drop per pod.
// It treats each scheduling event's impact as noise and learns a mean drop that adapts over time.
// Usage:
//   kf := NewKalmanFilter(0.1, 1.0, 0.001, 0.1)
//   // Before scheduling: reserve := kf.Reserve(mPods)
//   // After measurement arrives:
//   kf.Predict()
//   kf.Update(mPods, observedDrop)
//   // New estimate: kf.State()

type KalmanFilter struct {
	mu, P float64  // state (mean drop) and its covariance
	Q, R float64   // process-noise and measurement-noise variances
}

// NewKalmanFilter initializes a Kalman filter.
// initialMu: starting guess for drop per pod
// initialP: starting covariance (e.g. 1.0)
// Q: process noise covariance (higher = more agility)
// R: measurement noise covariance (higher = trust measurements less)
func NewKalmanFilter(initialMu, initialP, Q, R float64) *KalmanFilter {
	return &KalmanFilter{
		mu: initialMu,
		P:   initialP,
		Q:   Q,
		R:   R,
	}
}

// Predict advances the filter state assuming a random walk: mu_{k|k-1} = mu_{k-1|k-1}
// and increases uncertainty: P = P + Q.
func (kf *KalmanFilter) Predict() {
	kf.P += kf.Q
}

// Update incorporates a new batch measurement.
// mPods: number of pods scheduled since last update.
// drop: observed total signal drop (pre-signal - post-signal).
func (kf *KalmanFilter) Update(mPods int, drop float64) {
	m := float64(mPods)
    if m == 0 {
        log.Printf("KalmanFilter.Update called with mPods=0: skipping update")
        return
    }

	// Innovation variance: S = H P H^T + R, where H = m
	S := m*m*kf.P + kf.R

	// Kalman gain: K = P H^T / S
	K := (kf.P * m) / S

	// Residual: r = z - H mu
	r := drop - m*kf.mu

	// State update
	kf.mu += K * r

	// Covariance update: P = (1 - K H) P
	kf.P *= (1 - K*m)

	// Residual monitoring: if the drop is outside 3-sigma, warn
	sigma := math.Sqrt(S)
	if math.Abs(r) > 3*sigma {
		log.Printf("KalmanFilter large residual: r=%.4f, 3σ=%.4f; consider tuning Q/R", r, 3*sigma)
	}
}

// Reserve returns the amount to subtract from the node signal before scheduling mPods.
func (kf *KalmanFilter) Reserve(mPods int) float64 {
	return float64(mPods) * kf.mu
}

// State returns the current mean-drop estimate and covariance.
func (kf *KalmanFilter) State() (mu, P float64) {
	return kf.mu, kf.P
}

// Optional: Simple exponential moving average fallback if you choose not to do KF.
// Uncomment and use in place of Update if desired.
/*
func (kf *KalmanFilter) EMAUpdate(mPods int, drop, alpha float64) {
	if mPods <= 0 { return }
	avgDrop := drop / float64(mPods)
	kf.mu = alpha*avgDrop + (1-alpha)*kf.mu
}
*/

