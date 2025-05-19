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

type KalmanFilter interface {
    Predict()
    Update(u float64, y float64)
    State() []float64
    ForceState([]float64) (error)
    Revert()
}

type KalmanConfig struct {
  InitX     []float64 `yaml:"initX"`
  InitP     []float64 `yaml:"initP"`
  Q         []float64 `yaml:"Q"`
  R         float64 `yaml:"R"`
}
