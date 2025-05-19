package kalman

import (
	"errors"
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

type KalmanFilter1D struct {
	x [1]float64
    P float64  // state (mean drop) and its covariance
	Q, R float64   // process-noise and measurement-noise variances
}

// NewKalmanFilter1D initializes a Kalman filter.
// initialMu: starting guess for drop per pod
// initialP: starting covariance (e.g. 1.0)
// Q: process noise covariance (higher = more agility)
// R: measurement noise covariance (higher = trust measurements less)
func NewKalmanFilter1D(initX, initP, Q []float64, R float64) (KalmanFilter, error) {
    if len(initX) != 1 {
        return nil, errors.New("initX must have length 1")
    }
    if len(initP) != 1 {
        return nil, errors.New("initP must have length 1")
    }
    if len(Q) != 1 {
        return nil, errors.New("initQ must have length 1")
    }

    return &KalmanFilter1D{
		x:   [1]float64{initX[0]},
		P:   initP[0],
		Q:   Q[0],
		R:   R,
	}, nil
}

// Predict advances the filter state assuming a random walk: mu_{k|k-1} = mu_{k-1|k-1}
// and increases uncertainty: P = P + Q.
func (kf *KalmanFilter1D) Predict() {
	kf.P += kf.Q
}

// Update incorporates a new batch measurement.
// mPods: number of pods scheduled since last update.
// drop: observed total signal drop (pre-signal - post-signal).
func (kf *KalmanFilter1D) Update(u float64, y float64) {
    if u == 0 {
        log.Printf("KalmanFilter.Update called with mPods=0: skipping update")
        return
    }

	// Innovation variance: S = H P H^T + R, where H = m
	S := u*u*kf.P + kf.R

	// Kalman gain: K = P H^T / S
	K := (kf.P * u) / S

	// Residual: r = z - H mu
	r := y - u*kf.x[0]

	// State update
	kf.x[0] += K * r

	// Covariance update: P = (1 - K H) P
	kf.P *= (1 - K*u)

	// Residual monitoring: if the drop is outside 3-sigma, warn
	sigma := math.Sqrt(S)
	if math.Abs(r) > 3*sigma {
		log.Printf("KalmanFilter large residual: r=%.4f, 3σ=%.4f; consider tuning Q/R", r, 3*sigma)
	}
}

// State returns the current mean-drop estimate and covariance.
func (kf *KalmanFilter1D) State() ([]float64) {
    return kf.x[:]
}

func (kf *KalmanFilter1D) ForceState(newX[]float64) (error) {
    if len(newX) != 1 {
        return errors.New("initX must have length 1")
    }
    kf.x[0] = newX[0]
    return nil
}

func (kf *KalmanFilter1D) Revert() { }

// Optional: Simple exponential moving average fallback if you choose not to do KF.
// Uncomment and use in place of Update if desired.
/*
func (kf *KalmanFilter) EMAUpdate(mPods int, drop, alpha float64) {
	if mPods <= 0 { return }
	avgDrop := drop / float64(mPods)
	kf.mu = alpha*avgDrop + (1-alpha)*kf.mu
}
*/

