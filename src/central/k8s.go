package central

import (
	"context"
	"fmt"
	"log"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func (ctl *CentralScheduler) findNodes() error {
	// TODO add informer to get the list of nodeList
	nodeList, err := ctl.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s!=%s", "node-role.kubernetes.io/control-plane", ""),
	})

    if err != nil {
        return err
    }

    n := len(nodeList.Items)
	nMap := make(map[string]int)
    nodeNames := make([]string, n)
    nodes := make([]*Node, n)


	for i, node := range nodeList.Items {
		nMap[node.Name] = i
        nodeNames[i] = node.Name
        nodes[i] = NewNode(node.Name, 0.1, 10, 2)
	}

	ctl.nodeMap = nMap
    ctl.nodeNames = nodeNames
    ctl.nodes = nodes

    return nil
}

func (ctl *CentralScheduler) startWatchHandler(ctx context.Context) {
    ctl.watchQueue = make(chan *v1.Pod, 100)
    go func() {
        for {
            select {
            case <-ctx.Done():
                log.Print("(api) received shutdown signal")
                return
            default:
                /* Creates a watch interface for all pods that use this scheduler */
                watch, err := ctl.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
                    FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", ctl.name),
                })

                if err != nil {
                    log.Printf("(api) failed to set up watcher: %s", err.Error())
                    time.Sleep(1 * time.Second)
                    continue
                }

                if err := handleWatchEvents(ctx, ctl.watchQueue, watch); err != nil {
                    log.Printf("(api) error handling watch events: %s", err.Error())
                    time.Sleep(1 * time.Second) // backoff on error
                }
            }
        }
    }()
}

func handleWatchEvents(ctx context.Context, eventCh chan<- *v1.Pod, watch watch.Interface) error {
    for {
        select {
        case <-ctx.Done():
            log.Printf("(api) context cancelled, exiting event loop")
            return nil
        case event, ok := <- watch.ResultChan():
            if !ok {
                return fmt.Errorf("watch channel closed")
            }

            if event.Type != "ADDED" {
                continue
            }

            pod, ok := event.Object.(*v1.Pod)
            if !ok {
                log.Print("(api) unexpected event object type")
                continue
            }

            log.Printf("(api) watch <- %s", pod.Name)
            eventCh <- pod
        }
    }
}
