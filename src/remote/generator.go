package remote

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/LucaChot/pronto/src/remote/kalman"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const epsilon = 1e-3

type CostPerPodState struct {
    filter          *kalman.KalmanFilter2D
    lastSignal      float64
    lastBeta        float64
    lastPodCount    int
}

type costPerPodStateOption struct {
    initialMu       float64
    initialP        float64
    Q               float64
    R               float64
    startSignal     float64
    startPodCount   int
}

// Option configures a Scheduler
type CostPerPodOption func(*costPerPodStateOption)

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithInitialMu(initialMu float64) CostPerPodOption {
	return func(o *costPerPodStateOption) {
		o.initialMu = initialMu
	}
}

func WithInitialP(initialP float64) CostPerPodOption {
	return func(o *costPerPodStateOption) {
		o.initialP = initialP
	}
}

func WithQ(q float64) CostPerPodOption {
	return func(o *costPerPodStateOption) {
		o.Q = q
	}
}

func WithR(r float64) CostPerPodOption {
	return func(o *costPerPodStateOption) {
		o.R = r
	}
}

var defaultCostPerPodState = costPerPodStateOption{
    initialMu:      0.1,
    initialP:       1,
    Q:              0.01,
    R:              0.1,
    startSignal:     1.4,
    startPodCount:    0,
}


func NewCostPerPodState(opts ...CostPerPodOption) *CostPerPodState{
    options := defaultCostPerPodState
	for _, opt := range opts {
		opt(&options)
	}

    //kf := kalman.NewKalmanFilter1D(
        //options.initialMu,
        //options.initialP,
        //options.Q,
        //options.R)

    // initial guess: β=0.1, γ=1.0
    initB := [2]float64{0, -0.1}
    // initial uncertainty
    initP := [2][2]float64{{0,0},{0,1e3}}
    // small process noise
    Q  := [2][2]float64{{0,0},{0,1e-2}}
    // measurement noise
    R  := 1e-1
    kf := kalman.NewKalmanFilter2D(initB, initP, Q, R)

    cpp := &CostPerPodState{
        filter:     kf,
        lastSignal: options.startSignal,
        lastPodCount: options.startPodCount,
    }

    return cpp
}

func (cpp *CostPerPodState) UpdatePodCost1D(signal float64, podCount int) {
    // 1) compute how many new pods arrived
    deltaPods := podCount - cpp.lastPodCount

    // 2) decide whether to update β (per-pod cost)
    //    only if
    //      a) there's been at least one new pod
    //      b) the node had some headroom before (lastSignal > ε)
    if deltaPods != 0 && cpp.lastSignal > epsilon {
        // feed the filter the NEW (s, p) pair
        // internally this does predict+update on [β,γ]
        cpp.filter.Update(signal, float64(podCount))
    } else {
        // otherwise just update γ (the baseline) by doing
        // a “measurement” step with deltaPods = 0
        // that still corrects γ to your current signal.
        cpp.filter.Update(signal, float64(podCount))
        // then immediately clamp β back to its last value
        // so you don’t let the no-headroom step drift it downward
        beta, gamma := cpp.filter.State()
        cpp.filter.ForceState(math.Max(beta, cpp.lastBeta), gamma)
    }

    // 3) record for next time
    cpp.lastSignal   = signal
    cpp.lastPodCount = podCount
    cpp.lastBeta, _  = cpp.filter.State()
}

func (cpp *CostPerPodState) UpdatePodCost2D(signal float64, podCount int) {
    y := signal - cpp.lastSignal
    u := podCount - cpp.lastPodCount
    cpp.lastSignal = signal
    cpp.lastPodCount = podCount

    // Predict every interval
    cpp.filter.Predict()

    // Only update for pod-start events with positive drop
    // Skip if no pods started or signal floor-limited
    //if u <= 0 || 3 * y >= float64(u) * cpp.lastBeta {
    if u <= 0 || y >= -epsilon {
        return
    }

    // Optional: skip if measurement indicates no drop (censored)
    if y >= 0 {
        return
    }

    cpp.filter.Update(float64(u), y)
    cost, _ := cpp.filter.State()
    if cost > -0.08 {
        cpp.filter.ForceState(0, -0.08)
    }
    if cost < -0.12 {
        cpp.filter.ForceState(0, -0.12)
    }
}

/*
TODO: Look at including information such as variance to determine the threshold
*/
func (rmt *RemoteScheduler) JobSignal() (float64, error) {
    yPtr := rmt.mc.Y.Load()
    if yPtr == nil {
        return 0.0, errors.New("y vector is not available")
    }

    sumProbUPtr := rmt.fp.SumProbU.Load()
    if sumProbUPtr == nil {
        return 0.0, errors.New("sumProbU matrix is not available")
    }

    y := *yPtr
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

/* Core Scheduling loop */
/*
TODO: Implement error handling
*/
func (rmt *RemoteScheduler) Start() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    for {
        <-ticker.C
        log.Printf("running generation")

        ctx := context.Background()


        signal, err := rmt.JobSignal()
        if err != nil {
            log.Printf("error generating signal: %s", err)
            continue
        }

        log.Printf("(generator) signal = %.4f", signal)
        rmt.costPerPod.UpdatePodCost2D(signal, rmt.cache.GetPodCount())
        cost, constant := rmt.costPerPod.filter.State()
        log.Printf("(generator) per-pod cost = %.4f constant = %.4f\n", cost, constant)

        patch := fmt.Sprintf(`{"metadata":{"annotations":{"pronto/signal":"%f","pronto/pod-cost" : "%f"}}}`, signal, math.Abs(cost))

        _, err = rmt.client.CoreV1().Nodes().Patch(ctx, rmt.nodeName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
        if err != nil {
            log.Printf("Failed to patch node: %v", err)
            continue
        }
    }
}
