package aggregate

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)



func (ctl *Aggregator) startAggregateServer() {
    lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve start server")
	}

	s := grpc.NewServer()
    //pb.RegisterAggregateServer(s, agg)

	log.WithFields(log.Fields{
		"ADDRESS": lis.Addr(),
	}).Debug("STARTED AGGREGATE SERVER")

	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"ERR": err,
			}).Fatalf("FAILED TO START SERVER")
		}
	}()
}

func (ctl *Aggregator) RequestGlobalMerge() {
    /*
    log.WithFields(log.Fields{
        "NODE":     in.Node,
    }).Debug("RECEIVED POD REQUEST")
    */

    /*
    * Read the pointer to the struct containing both the U and Sigma *
    * Uses atomic.Pointer[T] to ensure atomicity *
    * Research into Golang's memory model *

    U, Sigma = pointer.read()

    agg.channel <- (in.U, in.Sigma)
    U', Sigma' := Merge(U, Sigma, in.U, in.Sigma, r)

    return &Response {
        U: U'
        Sigma: Sigma'
    }, nil
    */
}

