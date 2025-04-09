package remote

import (
	"context"
	"fmt"
	"math"
	"os"

	log "github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/mat"

	"github.com/LucaChot/pronto/src/fpca"
	pb "github.com/LucaChot/pronto/src/message"
	"github.com/LucaChot/pronto/src/metrics"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
    TR = 1
)

type RemoteScheduler struct {
    hostname string
    onNode *v1.Node

    mc *metrics.MetricsCollector
    fp *fpca.FPCAAgent

    tr float64

    clientset   *kubernetes.Clientset
    ctlPlStub  pb.PodPlacementClient
}

func (rmt *RemoteScheduler) SetClientset() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Error("CONFIG ERROR")
	}

	clientset, err := kubernetes.NewForConfig(config)
	rmt.clientset = clientset
}

func (rmt *RemoteScheduler) SetHostname() {
	hostname, err := os.Hostname()
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Error("HOSTNAME ERROR")
	}
    rmt.hostname = hostname;
}

func (rmt *RemoteScheduler) SetOnNode() {
    pods, err := rmt.clientset.CoreV1().Pods("pronto").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", rmt.hostname),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Error("LOCATING POD ERROR")
	}

	n, err := rmt.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "kubernetes.io/hostname", pods.Items[0].Spec.NodeName),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Error("LOCATING NODE ERROR")
	}

    rmt.onNode = &n.Items[0]
}

/* Creates a new CentralScheduler */
func New() *RemoteScheduler {

    /* Initialise scheduler values */
    rmt := &RemoteScheduler{
        tr: TR,
    }

    /* Run metrics collection */
    rmt.mc = metrics.New()
    go rmt.mc.Collect()

    /* Run fpca */
    rmt.fp = fpca.New()
    go rmt.fp.RunLocalUpdates()


    /* Set the remote scheduler variables */
    rmt.SetClientset()
    rmt.SetHostname()
    rmt.SetOnNode()
    rmt.AsClient()

	return rmt
}

func absFunc(i, j int, v float64) (float64) {
    return math.Abs(v)
}


func (rmt *RemoteScheduler) JobSignal() float64 {
    /* TODO: How to ensure that the B, U and Sigma we load are for the same
    * timestep. Will have to use an atomic pointer that points to be U and
    * Sigma */
    y := rmt.mc.Y.Load()

    uSigmaPair := rmt.fp.USIgma.Load()
    u := uSigmaPair.U
    sigma := uSigmaPair.Sigma

    var temp, p, wP *mat.Dense
    temp.Mul(y.T(), u)
    p.Apply(absFunc, temp)

    wP.Mul(sigma, p)

    return mat.Sum(wP)
}

/* Core Scheduling loop */
/*
TODO: Determine whether I calculate this on pod event or whether I send signal
periodically
TODO: Change to periodic as this will reduce delay, scheduler can use the
latest value received
*/
func (rmt *RemoteScheduler) Schedule() {

    /* Creates a watch interface for all pods that use this scheduler */
	watch, _ := rmt.clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", "pronto"),
	})

    /* Loops over all new pod events we detect */
	for event := range watch.ResultChan() {
        /* Ignore events where pods have been added */
		if event.Type != "ADDED" {
			continue
		}

		p := event.Object.(*v1.Pod)
		log.WithFields(log.Fields{
			"namespace": p.Namespace,
			"pod":       p.Name,
		}).Debug("BEGIN POD REQUEST")

        signal := rmt.JobSignal()
        if signal < rmt.tr {
            rmt.RequestPod(signal)
        }
	}
}

