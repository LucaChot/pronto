package remote

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

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

type RemoteScheduler struct {
    hostname string
    onNode *v1.Node

    mc *metrics.MetricsCollector
    fp *fpca.FPCAAgent

    clientset   *kubernetes.Clientset
    ctlSignalStub  pb.SignalServiceClient
    signalStream    pb.SignalService_StreamSignalsClient

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
    pods, err := rmt.clientset.CoreV1().Pods("basic-sched").List(context.TODO(), metav1.ListOptions{
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
    rmt := &RemoteScheduler{}

    /* Run metrics collection */
    var sender <-chan *mat.Dense
    rmt.mc, sender = metrics.New()
    log.Debug("RMT: INITIALISE METRIC COLLECTOR")

    /* Run fpca */
    rmt.fp = fpca.New(sender)
    log.Debug("RMT: INITIALISE FPCA")


    /* Set the remote scheduler variables */
    rmt.SetClientset()
    rmt.SetHostname()
    rmt.SetOnNode()
    rmt.AsClient()

    log.Debug("RMT: FINISHED INITIALISATION")
	return rmt
}

/*
TODO: Look at including information such as variance to determine the threshold
*/
func (rmt *RemoteScheduler) JobSignal() (float64, error) {
    yPtr := rmt.mc.Y.Load()
    if yPtr == nil {
        return 0.0, errors.New("y vector is not available")
    }

    sumProbUPtr := rmt.fp.SumProbU.Load()
    if sumProbUPtr == nil {
        return 0.0, errors.New("sumProbU matrix is not available")
    }

    y := *yPtr
    sumProbU := *sumProbUPtr

    // 1. Check if any y[i] is already >= 1
	for _, yi := range y {
		if yi >= 0.95 {
			return 0.0, nil
		}
	}

    kMin := math.Inf(1)
	found := false
	for i, ui := range sumProbU {
		if ui > 0 {
			cand := (1.0 - y[i]) / ui
			if cand < kMin {
				kMin = cand
			}
			found = true
		}
	}

	// 3. If no positive slope found, never reaches 1
	if !found {
		return math.Inf(1), nil
	}

	// 4. Clamp at zero
	if kMin < 0 {
		return 0.0, nil
	}

	return kMin, nil
}

/* Core Scheduling loop */
/*
TODO: Implement error handling
*/
func (rmt *RemoteScheduler) Schedule() {
    ticker := time.NewTicker(time.Millisecond)
    defer ticker.Stop()
    for {
        <-ticker.C

        signal, err := rmt.JobSignal()
        if err != nil {
            log.WithError(err)
        }
        rmt.RequestPod(signal)
	}
}

