package remote

import (
	"net"
	"time"

	pb "github.com/LucaChot/pronto/src/message"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (rmt *RemoteScheduler) AsClient() {
	ctrlAddr := findCtlAddr()
	rmt.connectToPl(ctrlAddr)
}

func findCtlAddr() net.IP {
	for {
		ips, err := net.LookupIP("central-svc.basic-sched.svc.cluster.local")
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Debug("Could not get IPs")
			time.Sleep(time.Second * 5)
		} else {
			return ips[0]
		}
	}
}

func (rmt *RemoteScheduler) connectToPl(ctlAddr net.IP) {

    conn, err := grpc.NewClient(ctlAddr.String()+":50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"for":   "placement",
		}).Fatal("Could not connect to controller")
	}

	rmt.ctlPlStub = pb.NewPodPlacementClient(conn)

}

func (rmt *RemoteScheduler) RequestPod(signal float64) {
    ctx := context.Background()
    rmt.ctlPlStub.RequestPod(ctx, &pb.PodRequest{
        Node:   rmt.onNode.Name,
        Signal: signal,
    })
    log.WithFields(log.Fields{
        "SIGNAL":   signal,
        "NODE":     rmt.onNode.Name,
    }).Debug("RMT: SENT POD REQUEST")
}
