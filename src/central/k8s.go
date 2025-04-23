package central

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ctl *CentralScheduler) findNodes() error {
	// TODO add informer to get the list of nodes
	nodes, err := ctl.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s!=%s", "node-role.kubernetes.io/control-plane", ""),
	})

    if err != nil {
        return err
    }

	nMap := make(map[string]int)
    nodeNames := make([]string, len(nodes.Items))

	for i, node := range nodes.Items {
		nMap[node.Name] = i
        nodeNames[i] = node.Name
	}

	ctl.nodeMap = nMap
    ctl.nodeNames = nodeNames

    return nil
}
