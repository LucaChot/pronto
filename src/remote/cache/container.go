package cache

import (
	"context"
	"log"
	"time"

	"github.com/containerd/containerd"
	eventsapi "github.com/containerd/containerd/api/events"
	tasks "github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/namespaces"
	typeurl "github.com/containerd/typeurl/v2"
)

type ContainerInformer struct {
    onChange        func(count int)
    socketPath      string
}

type ContainerInformerOptions struct {
    socketPath      string
}

// Option configures a Scheduler
type ContainerInformerOption func(*ContainerInformerOptions)

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithSocketPath(socketPath string) ContainerInformerOption {
	return func(o *ContainerInformerOptions) {
		o.socketPath = socketPath
	}
}

var defaultRemoteOptions = ContainerInformerOptions{
    socketPath: "/run/containerd/containerd.sock",
}

func NewContainerInformer(opts ...ContainerInformerOption) *ContainerInformer {
    options := ContainerInformerOptions{}
	for _, opt := range opts {
		opt(&options)
	}


    ci := &ContainerInformer{
        socketPath: options.socketPath,
    }

    return ci
}


func (ci *ContainerInformer) SetOnChange(onChange func(count int)) {
    ci.onChange = onChange
}

func (ci *ContainerInformer) Start() {
    go ci.watchPodCount()
}
// watchPodCount connects to containerd, calculates an initial
// count of unique pod UIDs running here, then keeps that count
// up-to-date by listening to TaskStart and TaskExit events.
//
// onChange is called whenever the current pod count increases or decreases.
func (ci *ContainerInformer) watchPodCount() error {
    log.Print("(container) start watching function")
    client, err := containerd.New("/run/containerd/containerd.sock")
    if err != nil {
        log.Printf("(container) could not connect to containerd socket %v", err)
        return err
    }
    defer client.Close()

    ctx := namespaces.WithNamespace(context.Background(), "k8s.io")

    // Maps podUID → number of running containers in that pod
    podCounts := make(map[string]int)
    activePods := 0

    // 1) INITIAL SYNC: list all running tasks
    listResp, err := client.TaskService().List(ctx, &tasks.ListTasksRequest{})
    if err != nil {
        log.Printf("(container) error listing services: %v", err)
        return err
    }
    for _, t := range listResp.Tasks {
        info, err := client.ContainerService().Get(ctx, t.ContainerID)
        if err != nil {
            log.Printf("(container) initial container info: %v", err)
            continue
        }
        if uid, ok := info.Labels["io.kubernetes.pod.uid"]; ok {
            if podCounts[uid] == 0 {
                activePods++
            }
            podCounts[uid]++
        }
    }
    // fire initial count
    ci.onChange(activePods)

    // 2) SUBSCRIBE to all task events
    eventsCh, errsCh := client.EventService().Subscribe(ctx,
    `topic=="/containers/create"`,
    `topic=="/tasks/exit"`,
    )
    log.Print("(container) received subscribe stream")

    // 3) EVENT LOOP
    for {
        select {
        case msg, ok :=  <-eventsCh:
            if !ok {
              log.Println("event stream closed")
              return nil
            }
            //log.Print("(container) received event")
            ev, err := typeurl.UnmarshalAny(msg.Event)
            if err != nil {
                log.Printf("(container) unmarshal event: %v", err)
                continue
            }

            // We’ll mutate podCounts and activePods under lock
            switch e := ev.(type) {
            case *eventsapi.ContainerCreate:
                //log.Print("(container) detected container start")
                info, err := client.ContainerService().Get(ctx, e.ID)
                if err != nil {
                    log.Printf("TaskStart: container info: %v", err)
                    break
                }
                if uid, ok := info.Labels["io.kubernetes.pod.uid"]; ok {
                    prev := podCounts[uid]
                    podCounts[uid] = prev + 1
                    if prev == 0 {
                        ts := msg.Timestamp.Format(time.RFC3339Nano)
                        log.Printf("(container) detected new pod: %s", ts)
                        ci.onChange(1)
                    }
                }

            case *eventsapi.TaskExit:
                //log.Print("(container) detected container termination")
                info, err := client.ContainerService().Get(ctx, e.ContainerID)
                if err != nil {
                    log.Printf("TaskExit: container info: %v", err)
                    break
                }
                if uid, ok := info.Labels["io.kubernetes.pod.uid"]; ok {
                    cnt := podCounts[uid] - 1
                    if cnt <= 0 {
                        ts := msg.Timestamp.Format(time.RFC3339Nano)
                        log.Printf("(container) detected pod termination : %s", ts)
                        ci.onChange(-1)
                        delete(podCounts, uid)
                    } else {
                        podCounts[uid] = cnt
                    }
                }
            }
        case err := <-errsCh:
            log.Fatalf("subscribe failed: %v", err)
            return err
        }
    }
}
