package central

import (
	"context"
	"errors"
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

type CtlAliasTable struct {
    at      *alias.AliasTable
    idxs    []int
}


type CentralScheduler struct {
    name        string
    clientset   *kubernetes.Clientset

    nodeMap     map[string]int
    nodeNames   []string
    nodeSignals []atomic.Uint64
    aliasTable  atomic.Pointer[CtlAliasTable]

    bindQueue   chan *podBind
    retryQueue  chan *v1.Pod

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
    ctl.startBindPodWorkers(4)
}

/*
TODO: Maybe handle the different errors of the aliasTable separately
E.g. zero length weights slice vs invalide values
*/
func (ctl *CentralScheduler) startAliasTableUpdater() {
    go func() {
        for {
            time.Sleep(1 * time.Second)
            log.Debug("CTL: UPDATING ALIAS TABLE")

            n := len(ctl.nodeSignals)
            signals := make([]float64,0, n)
            idxs := make([]int,0, n)

            for i := range n {
                signal := math.Float64frombits(ctl.nodeSignals[i].Load())
                if signal < 1 {
                    w := 1 / math.Max(signal, 1e-3)
                    signals = append(signals, w)
                    idxs = append(idxs, i)
                }
            }

            at, err := alias.New(signals)
            if err != nil {
                log.WithFields(log.Fields{
                    "error": err,
                }).Debug("CTL: ALL REMOTE NODES ARE OVERLOADED")
                ctl.aliasTable.Store(nil)
                continue
            }
            ctl.aliasTable.Store(&CtlAliasTable{
                at: at,
                idxs: idxs,
            })
        }
    }()
}


/*
Randlomly generates a node index depending on their probabilites in the
aliasTable
*/
func (ctl *CentralScheduler) selectNode() (string, error) {
    ctlAt := ctl.aliasTable.Load()

    if ctlAt == nil {
         return "", errors.New("alias table is not available")
    }

    idx := ctlAt.at.Sample()
    name := ctl.nodeNames[ctlAt.idxs[idx]]

    log.WithFields(log.Fields{
        "NODE": name,
    }).Debug("CTL: RANDOMLY ASSIGNED NODE")

    return name, nil
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

            pod, ok := event.Object.(*v1.Pod)
            if !ok {
                log.Warn("CTL: UNEXPECTED EVENT OBJECT TYPE")
                continue
            }

            log.WithFields(log.Fields{
                "namespace": pod.Namespace,
                "pod":       pod.Name,
            }).Debug("BEGIN POD SCHEDULE")


            /* Find a node to place the pod */
            node, err := ctl.selectNode()
            if err != nil {
                log.WithFields(log.Fields{
                    "error": err,
                }).Debug("CTL: FAILED TO FIND SUITABLE NODE")
                ctl.retryQueue <- pod
                continue
            }

            ctl.bindQueue <- &podBind{
                pod:    pod,
                node:   node,
            }
        case p := <- ctl.retryQueue:
            numPodsWaiting := len(ctl.retryQueue)
            podsWaiting := make([]*v1.Pod, numPodsWaiting + 1)
            podsWaiting[0] = p

            for i := 1; i < numPodsWaiting + 1; i++ {
                podsWaiting[i] = <-ctl.retryQueue
            }

            for _, pod := range podsWaiting {
                log.WithFields(log.Fields{
                    "namespace": pod.Namespace,
                    "pod":       pod.Name,
                }).Debug("BEGIN POD SCHEDULE")


                /* Find a node to place the pod */
                node, err := ctl.selectNode()
                if err != nil {
                    log.WithFields(log.Fields{
                        "error": err,
                    }).Debug("CTL: FAILED TO FIND SUITABLE NODE")
                    ctl.retryQueue <- pod
                    continue
                }

                ctl.bindQueue <- &podBind{
                    pod:    pod,
                    node:   node,
                }
            }
        }
    }
}

