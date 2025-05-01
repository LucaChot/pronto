package remote

import (
	"context"
	"log"
	"os"

	"github.com/LucaChot/pronto/src/remote/fpca"
	"github.com/LucaChot/pronto/src/remote/metrics"
	"github.com/LucaChot/pronto/src/remote/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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


func GetKernelHostName() (string) {
    n, err :=  os.Hostname()
    if err != nil {
        log.Printf("unable to get hostname from kernel")
        return ""
    }
    return n
}

func GetHostNodeEnvironment() (string) {
    return os.Getenv("NODE_NAME")
}

type remoteOptions struct {
    podName                             string
    nodeName                            string
    namespace                           string
}

// Option configures a Scheduler
type Option func(*remoteOptions)

func WithNamespace(namespace string) Option {
	return func(o *remoteOptions) {
		o.namespace = namespace
	}
}

var defaultRemoteOptions = remoteOptions{
    podName: GetKernelHostName(),
    nodeName: GetHostNodeEnvironment(),
    namespace: metav1.NamespaceAll,
}



// New returns a Scheduler
func New(ctx context.Context,
	client clientset.Interface,
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

    c := cache.New(
        cache.WithClientSet(client),
        cache.WithNamespace(options.namespace),
        cache.WithNodeName(options.nodeName),
        cache.WithStopEverything(stopEverything))

    cpp := NewCostPerPodState()

    rmt := &RemoteScheduler{
        podName:        options.podName,
        nodeName:       options.nodeName,
        mc:             mc,
        fp:             fp,
        cache:          c,
        costPerPod:     cpp,
        client:         client,
        StopEverything: stopEverything,
    }

	return rmt, nil

}

