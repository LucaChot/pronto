package cache

import (
	"context"
	"log"
	"time"

	//"time"

	"github.com/containerd/containerd"
	eventsapi "github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/namespaces"
	typeurl "github.com/containerd/typeurl/v2"
)

type ContainerEventInformer struct {
    onEvent        func(name string)
}

func NewContainerEventInformer() EventInformer {
    log.Print("created container informer")
    ci := &ContainerEventInformer{}

    return ci
}


func (ci *ContainerEventInformer) SetOnEvent(onEvent func(name string)) {
    ci.onEvent = onEvent
}

func (ci *ContainerEventInformer) Start() {
    go ci.watchEvents()
}

// watchPodCount connects to containerd, calculates an initial
// count of unique pod UIDs running here, then keeps that count
// up-to-date by listening to TaskStart and TaskExit events.
//
// onChange is called whenever the current pod count increases or decreases.
func (ci *ContainerEventInformer) watchEvents() error {
    log.Print("(container) start watching function")
    client, err := containerd.New("/run/containerd/containerd.sock")
    if err != nil {
        log.Printf("(container) could not connect to containerd socket %v", err)
        return err
    }
    defer client.Close()

    ctx := namespaces.WithNamespace(context.Background(), "k8s.io")

    // 2) SUBSCRIBE to all task events
    eventsCh, errsCh := client.EventService().Subscribe(ctx,
    `topic=="/tasks/start"`,
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
            case *eventsapi.TaskStart:
                //log.Print("(container) detected container start")
                info, err := client.ContainerService().Get(ctx, e.ContainerID)
                if err != nil {
                    log.Printf("TaskStart: container info: %v", err)
                    break
                }
                if uid, ok := info.Labels["io.kubernetes.pod.uid"]; ok {
                    ts := msg.Timestamp.Format(time.RFC3339Nano)
                    log.Printf("(container) detected new pod: %s", ts)
                    ci.onEvent(uid)
                }

            case *eventsapi.TaskExit:
                //log.Print("(container) detected container termination")
                info, err := client.ContainerService().Get(ctx, e.ContainerID)
                if err != nil {
                    log.Printf("TaskExit: container info: %v", err)
                    break
                }
                if uid, ok := info.Labels["io.kubernetes.pod.uid"]; ok {
                    ts := msg.Timestamp.Format(time.RFC3339Nano)
                    log.Printf("(container) detected pod termination : %s", ts)
                    ci.onEvent(uid)
                }
            }
        case err := <-errsCh:
            log.Fatalf("subscribe failed: %v", err)
            return err
        }
    }
}
