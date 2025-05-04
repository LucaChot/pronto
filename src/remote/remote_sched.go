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

    cache       *cache.Cache
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
    podName                             string
    nodeName                            string
}

// Option configures a Scheduler
type Option func(*remoteOptions)

var defaultRemoteOptions = remoteOptions{
    podName: GetPodName(),
    nodeName: GetNodeName(),
}



// New returns a Scheduler
func New(ctx context.Context,
	client clientset.Interface,
    cache   *cache.Cache,
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

	return rmt, nil
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
        rmt.costPerPod.Update(rmt.cache.GetPodCount(), signal)
        cost := rmt.costPerPod.GetPodCost()
        log.Printf("(generator) per-pod cost = %.4f", cost)

        patch := fmt.Sprintf(`{"metadata":{"annotations":{"pronto/signal":"%f","pronto/pod-cost" : "%f"}}}`, signal, cost)

        _, err = rmt.client.CoreV1().Nodes().Patch(ctx, rmt.nodeName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
        if err != nil {
            log.Printf("Failed to patch node: %v", err)
            continue
        }
    }
}

