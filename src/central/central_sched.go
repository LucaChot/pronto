package central

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/LucaChot/pronto/src/message"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type CentralScheduler struct {
    name        string
    clientset   *kubernetes.Clientset

    nodeMap     map[string]int
    nodeNames   []string
    nodeSignals []atomic.Uint64

    nodes []*Node
    mu    sync.Mutex
    rand  *rand.Rand

    watchQueue  chan *v1.Pod
    bindQueue   chan *podBind
    retryQueue  chan *v1.Pod

    pb.UnimplementedSignalServiceServer
}

type Option func(*CentralScheduler)

func WithClientset(clientset *kubernetes.Clientset) Option {
	return func(s *CentralScheduler) {
		s.clientset = clientset
	}
}

/* Creates a new CentralScheduler */
func New(name string, opts ...Option) *CentralScheduler {
    /* Initialise scheduler values */
	ctl := &CentralScheduler{
		name: name,
        rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
    }

    for _, opt := range opts {
		opt(ctl)
	}
	return ctl
}

func (ctl *CentralScheduler) Init() error {
    err := ctl.findNodes()
    if err != nil {
        return fmt.Errorf("failed to find nodes: %w", err)
	}

    ctl.nodeSignals = make([]atomic.Uint64, len(ctl.nodeMap))
    for node := range(len(ctl.nodeMap)) {
        ctl.nodeSignals[node].Store(math.Float64bits(0.0))
    }

    return nil
}

func (ctl *CentralScheduler) Start(ctx context.Context) {
    ctl.startPlacementServer(ctx)
    ctl.startBindPodWorkers(ctx, 4)
    ctl.startWatchHandler(ctx)
}

/* Core Scheduling loop */
func (ctl *CentralScheduler) RunScheduler(ctx context.Context) error {
    log.Print("(sched) starting main scheduler loop")
    for {
        for {
            select {
            case <-ctx.Done():
                log.Print("(sched) context cancelled, exiting event loop")
                return nil
            case pod := <- ctl.retryQueue:
                log.Printf("(sched) %s <-retry", pod.Name)
                /* Find a node to place the pod */
                node, err := ctl.SchedulePod()
                if err != nil {
                    log.Printf("(sched) %s\n(sched) retry <-%s : total %d", err.Error(), pod.Name, len(ctl.retryQueue)+1)
                    ctl.retryQueue <- pod
                    time.Sleep(10 * time.Millisecond)
                    break
                }

                log.Printf("(sched) bind <-%s", pod.Name)
                ctl.bindQueue <- &podBind{
                    pod:    pod,
                    node:   node.ID,
                }
            case pod := <- ctl.watchQueue:
                log.Printf("(sched) %s <-watch", pod.Name)

                /* Find a node to place the pod */
                node, err := ctl.SchedulePod()
                if err != nil {
                    log.Printf("(sched) %s\n(sched) retry <-%s : total %d", err.Error(), pod.Name, len(ctl.retryQueue)+1)
                    ctl.retryQueue <- pod
                    time.Sleep(10 * time.Millisecond)
                    break
                }

                log.Printf("(sched) bind <-%s", pod.Name)
                ctl.bindQueue <- &podBind{
                    pod:    pod,
                    node:   node.ID,
                }
            }
        }
    }
}

