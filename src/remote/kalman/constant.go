package kalman

import (
	"errors"
)

type Constant struct {
	x [1]float64
}

func NewConstant(initX []float64, initP []float64, Q []float64, R float64) (KalmanFilter, error) {
    if len(initX) != 1 {
        return nil, errors.New("initX must have length 1")
    }
    return &Constant{
		x:   [1]float64{initX[0]},
	}, nil
}

func (kf *Constant) Predict() {}

func (kf *Constant) Update(u float64, y float64) {}

func (kf *Constant) State() ([]float64) {
    return kf.x[:]
}

func (kf *Constant) ForceState(newX[]float64) (error) {
    if len(newX) != 1 {
        return errors.New("initX must have length 1")
    }
    kf.x[0] = newX[0]
    return nil
}
