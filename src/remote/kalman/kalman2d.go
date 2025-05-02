package kalman

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
    B [2]float64
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
func NewKalmanFilter2D(initB [2]float64, initP, Q [2][2]float64, R float64) *KalmanFilter2D {
	return &KalmanFilter2D{
		B:   initB,
		P:   initP,
		Q:   Q,
		R:   R,
	}
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
	predY := kf.B[0] + kf.B[1]*u
	residual := y - predY

	// Update state
	kf.B[0] += K0 * residual
	kf.B[1] += K1 * residual

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
func (kf *KalmanFilter2D) State() (beta, gamma float64) {
    return kf.B[1], kf.B[0]
}

func (kf *KalmanFilter2D) ForceState(beta, gamma float64) {
    kf.B[0] = beta
    kf.B[1] = gamma
}
