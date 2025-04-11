package central

import (
	"context"
	"math"
	"net"

	pb "github.com/LucaChot/pronto/src/message"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)



func (ctl *CentralScheduler) ctlStartPlacementServer() {
    lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve start server")
	}

	s := grpc.NewServer()
    pb.RegisterPodPlacementServer(s, ctl)

	log.WithFields(log.Fields{
		"ADDRESS": lis.Addr(),
	}).Debug("STARTED PLACEMENT SERVER")

	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("failed to serve start server")
		}
	}()
}

func (ctl *CentralScheduler) RequestPod(ctx context.Context, in *pb.PodRequest) (*pb.EmptyReply, error) {
    log.WithFields(log.Fields{
        "SIGNAL":      in.Signal,
        "NODE":     in.Node,
    }).Debug("RECEIVED JOB SIGNAL")

    index := ctl.nodeMap[in.Node]
    ctl.nodeSignals[index].Store(math.Float64bits(in.Signal))

    return &pb.EmptyReply{}, nil
}

