package central

import (
	"io"
	"math"
	"net"

	pb "github.com/LucaChot/pronto/src/message"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)



func (ctl *CentralScheduler) startPlacementServer() {
    lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve start server")
	}

	s := grpc.NewServer()
    pb.RegisterSignalServiceServer(s, ctl)

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

func (ctl *CentralScheduler) StreamSignals(stream pb.SignalService_StreamSignalsServer) error {
    var m pb.Signal
    var node string
    for {
        err := stream.RecvMsg(&m)
        if err != nil {
            if node != "" {
                index := ctl.nodeMap[node]
                ctl.nodeSignals[index].Store(math.Float64bits(0.0))
            }
            if err == io.EOF {
                log.Printf("Client %s disconnected gracefully.", node)
                return nil
            }
            log.Printf("Error receiving stream from client %s: %v", node, err)
			return status.Errorf(codes.Internal, "error receiving stream: %v", err)
        }

        if node == "" {
            node = m.GetNode()
        }
        index := ctl.nodeMap[node]
        ctl.nodeSignals[index].Store(math.Float64bits(m.GetSignal()))
    }
}

