package central

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type podBind struct {
    pod     *v1.Pod
    node    string
}

/* Binds Pod p to Node n */
func (ctl *CentralScheduler) startBindPodWorkers(n int) {
    ctl.bindQueue = make(chan *podBind, 2 * n)
    ctl.retryQueue = make(chan *v1.Pod, 4 * n)

    for range n {
        go ctl.bindPodWorker()
    }
}

type BindInfo struct {
    bind        *v1.Binding
    event       *v1.Event
    createOpt   *metav1.CreateOptions
    deleteOpt   *metav1.DeleteOptions
}

func NewBindInfo() (*BindInfo) {
    bd := BindInfo{
        bind: &v1.Binding{},
        createOpt: &metav1.CreateOptions{},
        deleteOpt: &metav1.DeleteOptions{},
        event: &v1.Event{},
    }

    return &bd
}

func (bd *BindInfo) Reset() {
    *bd.bind = v1.Binding{
        ObjectMeta: metav1.ObjectMeta{},
        Target: v1.ObjectReference{
            APIVersion: "v1",
            Kind:       "Node",
        },
    }

    *bd.createOpt = metav1.CreateOptions{}
    *bd.deleteOpt = metav1.DeleteOptions{}

    *bd.event = v1.Event{
        Action:         "Binding",
        Reason:         "Scheduled",
        Type:           "Normal",
        InvolvedObject: v1.ObjectReference{
            Kind:      "Pod",
        },
        ObjectMeta: metav1.ObjectMeta{},
    }
}

func (bd *BindInfo) withReportingController(name string) {
    bd.event.ReportingController = name
}

func (bd *BindInfo) withReportingInstance(name string) {
    bd.event.ReportingInstance = name
}


func (bd *BindInfo) withBind(pod *v1.Pod, node string) {
    bd.bind.ObjectMeta.Name = pod.Name
    bd.bind.ObjectMeta.Namespace = pod.Namespace
    bd.bind.Target.Name = node

    bd.event.Message = fmt.Sprintf("Successfully assigned %s/%s to %s", pod.Namespace, pod.Name, node)

    bd.event.InvolvedObject.Name = pod.Name
    bd.event.InvolvedObject.Namespace = pod.Namespace
    bd.event.InvolvedObject.UID = pod.UID

    bd.event.ObjectMeta.GenerateName = pod.Name + "-"
}

/*
TODO: Look into making this loop less brittle, e.g. have goroutines work on
separate pointers
*/
func (ctl * CentralScheduler) bindPodWorker() {
    bd := NewBindInfo()
    bd.withReportingController(ctl.name)
    bd.withReportingInstance(fmt.Sprintf("%s-dev-k8s-lc869-00", ctl.name))

    bindResult  := make(chan error, 1)
    eventResult := make(chan *v1.Event, 1)

    for item := range ctl.bindQueue {
        pod := item.pod
        node := item.node
        bd.Reset()
        bd.withBind(pod, node)

        ctx, cancel := context.WithCancel(context.Background())
        // 3) Start event‑creation loop (infinite backoff until success or cancel)
        go func() {
            backoff := time.Second
            limit := time.Minute / 2
            for {
                select {
                case <-ctx.Done():
                    eventResult <- nil  // non‑nil
                    return
                default:
                }

                evt, err := ctl.clientset.CoreV1().
                    Events(pod.Namespace).
                    Create(ctx, bd.event, *bd.createOpt)
                if err == nil {
                    eventResult <- evt       // success!
                    return
                }

                time.Sleep(backoff)
                backoff *= 2
                if backoff > limit {
                    backoff = limit
                }
            }
        }()

        // 4) Start bind‑with‑max‑3‑tries loop
        go func() {
            backoff := time.Second
            var err error
            for range 3 {
                err = ctl.clientset.CoreV1().
                    Pods(pod.Namespace).
                    Bind(context.TODO(), bd.bind, *bd.createOpt)
                if err == nil {
                    break
                }
                time.Sleep(backoff)
                backoff *= 2
            }
            bindResult <- err
        }()

        var bindErr error
        bindErr = <-bindResult
        if bindErr != nil {
            // Bind gave up after 3 tries: stop event attempts
            cancel()
            eventErr := <-eventResult

            // If event did ever succeed, delete it
            if eventErr != nil {
                // retry deletion forever (or until you want a limit)
                backoff := time.Second
                limit := time.Minute / 2
                for {
                    if err := ctl.clientset.CoreV1().
                        Events(pod.Namespace).
                        Delete(context.TODO(), eventErr.Name, *bd.deleteOpt); err == nil {
                        break
                    }
                    time.Sleep(backoff)
                    backoff *= 2
                    if backoff > limit {
                        backoff = limit
                    }
                }
            }

            // now requeue this pod for a later bind attempt
            ctl.retryQueue <- pod
            continue
        }

        <-eventResult
        cancel()
    }
}

