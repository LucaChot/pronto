package central

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/LucaChot/pronto/src/alias"
	pb "github.com/LucaChot/pronto/src/message"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type CentralScheduler struct {
    name        string
    clientset   *kubernetes.Clientset

    nodeMap     map[string]int
    nodeNames   []string
    nodeSignals []atomic.Uint64
    aliasTable  atomic.Pointer[alias.AliasTable]

    pb.UnimplementedPodPlacementServer
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
        ctl.nodeSignals[node].Store(math.Float64bits(1))
    }

    return nil
}

func (ctl *CentralScheduler) Start() {
    ctl.startAliasTableUpdater()
    ctl.startPlacementServer()
}

func (ctl *CentralScheduler) startAliasTableUpdater() {
    go func() {
        for {
            time.Sleep(1 * time.Second)
            log.Debug("CTL: UPDATING ALIAS TABLE")

            n := len(ctl.nodeSignals)
            signals := make([]float64, n)

            for i := range n {
                signals[i] = 1 / (math.Float64frombits(ctl.nodeSignals[i].Load()) + 1e-9)
            }

            at, err := alias.New(signals)
            if err != nil {
                log.WithFields(log.Fields{
                    "error": err,
                }).Debug("CTL: FAILED TO BUILD ALIAS TABLE")
                continue
            }
            ctl.aliasTable.Store(at)
        }
    }()
}


/*
Randlomly generates a node index depending on their probabilites in the
aliasTable TODO: Improve error handling when AliasTable is not yet available
for usage, i.e. use assign to a random node or add the pods to a queue to then
be dealt with after a timeout
*/
func (ctl *CentralScheduler) selectNode() string {
    at := ctl.aliasTable.Load()

    if at == nil {
        return ""
    }

    idx := at.Sample()
    name := ctl.nodeNames[idx]

    log.WithFields(log.Fields{
        "NODE": name,
    }).Debug("CTL: RANDOMLY ASSIGNED NODE")

    return name
}

/* Core Scheduling loop */
func (ctl *CentralScheduler) RunScheduler(ctx context.Context) error {
    log.Info("CTL: STARTING SCHEDULER LOOP")

    for {
        select {
        case <-ctx.Done():
            log.Info("CTL: RECEIVED SHUTDOWN SIGNAL")
            return nil
        default:
            /* Creates a watch interface for all pods that use this scheduler */
            watch, err := ctl.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
                FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", ctl.name),
            })

            if err != nil {
                log.WithFields(log.Fields{
                    "ERROR": err,
                }).Error("CTL: FAILED TO SET UP WATCHER")
                time.Sleep(1 * time.Second)
                continue
            }

            if err := ctl.handleWatchEvents(ctx, watch); err != nil {
                log.WithFields(log.Fields{
					"ERROR": err,
				}).Error("CTL: ERROR HANDLING WATCH EVENTS")
				time.Sleep(1 * time.Second) // backoff on error
            }
        }
    }
}

/* Core Scheduling loop */
func (ctl *CentralScheduler) handleWatchEvents(ctx context.Context, watch watch.Interface) error {
    defer watch.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Info("CTL: CONTEXT CANCELLED, EXITING EVENT LOOP")
            return nil
        case event, ok := <- watch.ResultChan():
            if !ok {
                return fmt.Errorf("CTL: WATCH CHANNEL CLOSED")
            }

            if event.Type != "ADDED" {
                continue
            }

            start := time.Now().UTC()

            p, ok := event.Object.(*v1.Pod)
            if !ok {
                log.Warn("CTL: UNEXPECTED EVENT OBJECT TYPE")
                continue
            }

            log.WithFields(log.Fields{
                "namespace": p.Namespace,
                "pod":       p.Name,
            }).Debug("BEGIN POD SCHEDULE")


            /* Find a node to place the pod */
            node := ctl.selectNode()
            if node == "" {
                log.Debug("FAILED TO FIND SUITABLE NODE")
                continue
            }

            ctl.placePodToNode(p, node)

            /* Collect information for event */
            end := time.Now().UTC()
            nanosecondsSpent := end.Sub(start).Nanoseconds()
            annotations := map[string]string{
                "scheduler/nanoseconds": fmt.Sprintf("%d", nanosecondsSpent),
            }

            /* Creates a new event alerting the binding of the pod */
            if err := ctl.createSchedEvent(p, node, end, annotations); err != nil {
                log.WithFields(log.Fields{
                    "err": err,
                }).Debug("FAILED TO CREATE EVENT")
            }
        }
    }
}

