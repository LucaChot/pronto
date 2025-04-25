package central

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	pb "github.com/LucaChot/pronto/src/message"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

const MAXSAMPLES = 4


type CentralScheduler struct {
    name        string
    clientset   *kubernetes.Clientset

    nodeMap     map[string]int
    nodeNames   []string
    nodeSignals []atomic.Uint64

    aliasTable  atomic.Pointer[CtlAliasTable]
    atLock      sync.Mutex
    atCond      *sync.Cond



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
    ctl.atCond = sync.NewCond(&ctl.atLock)

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
        log.Debug("ALIAS: STARTING ALIAS UPDATE")
        for {
            time.Sleep(10 * time.Millisecond)

            n := len(ctl.nodeSignals)
            signals := make([]float64,0, n)
            idxs := make([]int,0, n)

            for i := range n {
                signal := math.Float64frombits(ctl.nodeSignals[i].Load())
                if signal > 1e-3 {
                    signals = append(signals, signal)
                    idxs = append(idxs, i)
                }
            }

            cat, err := NewCtlAliasTable(signals)
            if err != nil {
                log.WithError(err)
                continue
            }
            if err := cat.WithIndxs(idxs); err != nil {
                log.WithError(err)
                continue
            }
            if err := cat.WithMaxSamples(MAXSAMPLES); err != nil {
                log.WithError(err)
                continue
            }

            ctl.aliasTable.Store(cat)
            ctl.atLock.Lock()
            ctl.atCond.Signal()
            ctl.atLock.Unlock()
        }
    }()
}


/*
Randlomly generates a node index depending on their probabilites in the
aliasTable
*/
func (ctl *CentralScheduler) selectNode() (string, error) {
    cat := ctl.aliasTable.Load()

    if cat == nil {
         return "", errors.New("alias table is not available")
    }

    idx, err := cat.Sample()
    if err != nil {
        return "", err
    }

    name := ctl.nodeNames[idx]
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

func (ctl *CentralScheduler) waitForAliasTable() {
    ctl.atLock.Lock()
    cat := ctl.aliasTable.Load()
    if cat == nil {
        ctl.atCond.Wait()
    } else {
        if left, err := cat.SamplesLeft(); !left || err != nil {
            ctl.atCond.Wait()
        }
    }
    ctl.atLock.Unlock()
}

/* Core Scheduling loop */
func (ctl *CentralScheduler) handleWatchEvents(ctx context.Context, watch watch.Interface) error {
    defer watch.Stop()

    for {
        ctl.waitForAliasTable()
        for {
            select {
            case <-ctx.Done():
                log.Info("CTL: CONTEXT CANCELLED, EXITING EVENT LOOP")
                return nil
            case pod := <- ctl.retryQueue:
                log.Debugf("%s <-retry", pod.Name)
                /* Find a node to place the pod */
                node, err := ctl.selectNode()
                if err != nil {
                    log.WithError(err)
                    log.Debugf("retry <-%s", pod.Name)
                    ctl.retryQueue <- pod
                    break
                }

                log.Debugf("bind <-%s", pod.Name)
                ctl.bindQueue <- &podBind{
                    pod:    pod,
                    node:   node,
                }
            case event, ok := <- watch.ResultChan():
                if !ok {
                    return fmt.Errorf("CTL: WATCH CHANNEL CLOSED")
                }

                if event.Type != "ADDED" {
                    continue
                }
                log.Info("CTL: FETCHING FROM WATCH QUEUE")

                pod, ok := event.Object.(*v1.Pod)
                if !ok {
                    log.Warn("CTL: UNEXPECTED EVENT OBJECT TYPE")
                    continue
                }
                log.Debugf("%s <-watch", pod.Name)

                /* Find a node to place the pod */
                node, err := ctl.selectNode()
                if err != nil {
                    log.WithError(err)
                    log.Debugf("retry <-%s", pod.Name)
                    ctl.retryQueue <- pod
                    break
                }

                log.Debugf("bind <-%s", pod.Name)
                ctl.bindQueue <- &podBind{
                    pod:    pod,
                    node:   node,
                }
            }
        }
    }
}

