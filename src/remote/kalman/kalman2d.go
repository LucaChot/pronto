package kalman

import "errors"

// KalmanFilter implements a 1-D Kalman filter to estimate the average signal drop per pod.
// It treats each scheduling event's impact as noise and learns a mean drop that adapts over time.
// Usage:
//   kf := NewKalmanFilter(0.1, 1.0, 0.001, 0.1)
//   // Before scheduling: reserve := kf.Reserve(mPods)
//   // After measurement arrives:
//   kf.Predict()
//   kf.Update(mPods, observedDrop)
//   // New estimate: kf.State()

type KalmanFilter2D struct {
    // state vector [β, γ]
    X [2]float64
    // covariance P
    P [2][2]float64
    // process noise Q
    Q [2][2]float64
    // measurement noise R
    R float64
}

// NewKalmanFilter2D initializes a Kalman filter.
// initialMu: starting guess for drop per pod
// initialP: starting covariance (e.g. 1.0)
// Q: process noise covariance (higher = more agility)
// R: measurement noise covariance (higher = trust measurements less)
func NewKalmanFilter2D(initX, initP, Q []float64, R float64) (KalmanFilter, error) {
    if len(initX) != 2 {
        return nil, errors.New("initX must have length 1")
    }
    if len(initP) != 4 {
        return nil, errors.New("initP must have length 1")
    }
    if len(Q) != 4 {
        return nil, errors.New("initQ must have length 1")
    }
	return &KalmanFilter2D{
        X:   [2]float64{initX[0], initX[1]},
		P:   [2][2]float64{
                {initP[0],initP[1]},
                {initP[2],initP[3]}},
		Q:   [2][2]float64{
                {Q[0],Q[1]},
                {Q[2],Q[3]}},
		R:   R,
	},nil
}

// Predict advances the filter state (time update).
func (kf *KalmanFilter2D) Predict() {
	// A = I so beta stays the same
	// P = P + Q
	for i := range 2 {
		for j := range 2 {
			kf.P[i][j] += kf.Q[i][j]
		}
	}
}

// Update incorporates a new measurement y with input u.
func (kf *KalmanFilter2D) Update(u, y float64) {
	// H = [1, u]
	// Compute innovation covariance S = H*P*H^T + R
	S := kf.P[0][0] + kf.P[0][1]*u + kf.P[1][0]*u + kf.P[1][1]*u*u + kf.R

	// Compute Kalman gain K = P*H^T/S
	K0 := (kf.P[0][0] + kf.P[0][1]*u) / S
	K1 := (kf.P[1][0] + kf.P[1][1]*u) / S

	// Residual = y - (beta0 + beta1*u)
	predY := kf.X[0] + kf.X[1]*u
	residual := y - predY

	// Update state
	kf.X[0] += K0 * residual
	kf.X[1] += K1 * residual

	// Update covariance: P = (I - K*H) * P
	M00 := 1 - K0*1
	M01 := -K0 * u
	M10 := -K1 * 1
	M11 := 1 - K1*u

	oldP := kf.P
	kf.P[0][0] = M00*oldP[0][0] + M01*oldP[1][0]
	kf.P[0][1] = M00*oldP[0][1] + M01*oldP[1][1]
	kf.P[1][0] = M10*oldP[0][0] + M11*oldP[1][0]
	kf.P[1][1] = M10*oldP[0][1] + M11*oldP[1][1]
}


// State returns the current estimates β (per-pod cost) and γ (base headroom).
func (kf *KalmanFilter2D) State() ([]float64) {
    return kf.X[:]
}

func (kf *KalmanFilter2D) ForceState(newX[]float64) (error) {
    if len(newX) != 2 {
        return errors.New("initX must have length 2")
    }
    kf.X[0] = newX[0]
    kf.X[1] = newX[1]
    return nil
}
