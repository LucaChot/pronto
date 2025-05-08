package remote

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/LucaChot/pronto/src/remote/cache"
	"github.com/LucaChot/pronto/src/remote/fpca"
	"github.com/LucaChot/pronto/src/remote/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
)

type RemoteScheduler struct {
    podName     string
    nodeName    string

    mc *metrics.MetricsCollector
    fp *fpca.FPCAAgent

    cache       *cache.EventCache
    costPerPod  *CostPerPodState

    client   clientset.Interface
    // Close this to shut down the scheduler.
	StopEverything <-chan struct{}

}


func GetPodName() (string) {
    n, err :=  os.Hostname()
    if err != nil {
        log.Printf("unable to get hostname from kernel")
        return ""
    }
    return n
}

func GetNodeName() (string) {
    return os.Getenv("NODE_NAME")
}

type remoteOptions struct {
    podName     string
    nodeName    string
    trigger     bool
}

// Option configures a Scheduler
type Option func(*remoteOptions)

func WithTrigger() Option {
	return func(o *remoteOptions) {
		o.trigger = true
	}
}

var defaultRemoteOptions = remoteOptions{
    podName: GetPodName(),
    nodeName: GetNodeName(),
    trigger: false,
}

// New returns a Scheduler
func New(ctx context.Context,
	client clientset.Interface,
    cache   *cache.EventCache,
    cpp     *CostPerPodState,
	opts ...Option) (*RemoteScheduler, error) {

	stopEverything := ctx.Done()

	options := defaultRemoteOptions
	for _, opt := range opts {
		opt(&options)
	}

    /* Run metrics collection */
    mc, sender := metrics.New()

    /* Run fpca */
    fp := fpca.New(sender)

    rmt := &RemoteScheduler{
        podName:        options.podName,
        nodeName:       options.nodeName,
        mc:             mc,
        fp:             fp,
        cache:          cache,
        costPerPod:     cpp,
        client:         client,
        StopEverything: stopEverything,
    }

    if options.trigger {
        cache.SetSignal(rmt.JobSignal)
        cache.SetUpdateCost(cpp.Update)
        cache.SetPublish(rmt.publish)
    }

	return rmt, nil
}

/* Core Scheduling loop */
/*
TODO: Implement error handling
*/

func (rmt *RemoteScheduler) publish(signal, cost float64) {
    go func() {
        ctx := context.Background()
        patch := fmt.Sprintf(`{"metadata":{"annotations":{"pronto/signal":"%f","pronto/pod-cost" : "%f"}}}`, signal, cost)

        _, err := rmt.client.CoreV1().Nodes().Patch(ctx, rmt.nodeName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
        if err != nil {
            log.Printf("Failed to patch node: %v", err)
        }
    }()
}


func (rmt *RemoteScheduler) Start() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    for {
        <-ticker.C
        if rmt.cache.IsWaiting() {
            signal := rmt.cache.GetLastSignal()
            podDiff := rmt.cache.GetChangeInPodCount()
            cost := rmt.costPerPod.GetPodCost()
            log.Printf("(remote) signal = %.4f", signal - (cost * float64(podDiff)))
            log.Printf("(remote) per-pod cost = %.4f", cost)
            rmt.publish(signal - (cost * float64(podDiff)), cost)
            continue
        }


        signal, err := rmt.JobSignal()
        if err != nil {
            log.Printf("error generating signal: %s", err)
            continue
        }
        log.Printf("(remote) signal = %.4f", signal)
        cost := rmt.costPerPod.GetPodCost()
        log.Printf("(remote) per-pod cost = %.4f", cost)
        rmt.publish(signal, cost)

    }
}

