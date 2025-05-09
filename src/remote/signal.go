package remote

import (
	"errors"
	"math"
)

/*
TODO: Look at including information such as variance to determine the threshold
*/
func (rmt *RemoteScheduler) JobSignal() (float64, error) {
    y, err := rmt.mc.GetY()
    if err != nil {
        return 0.0, errors.New("y vector is not available")
    }

    sumProbUPtr := rmt.fp.SumProbU.Load()
    if sumProbUPtr == nil {
        return 0.0, errors.New("sumProbU matrix is not available")
    }

    sumProbU := *sumProbUPtr

    // 1. Check if any y[i] is already >= 1
	for _, yi := range y {
		if yi >= 0.95 {
			return 0.0, nil
		}
	}

    kMin := math.Inf(1)
	found := false
	for i, ui := range sumProbU {
		if ui > 0 {
			cand := (1.0 - y[i]) / ui
			if cand < kMin {
				kMin = cand
			}
			found = true
		}
	}

	// 3. If no positive slope found, never reaches 1
	if !found {
		return math.Inf(1), nil
	}

	// 4. Clamp at zero
	if kMin < 0 {
		return 0.0, nil
	}

	return kMin, nil
}
