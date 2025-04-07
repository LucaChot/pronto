package aggregate

import (
	"context"
	"net"

	pb "github.com/LucaChot/pronto/src/message"
	log "github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/mat"
	"google.golang.org/grpc"
)



func (agg *Aggregator) startAggregateServer() {
    lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve start server")
	}

	s := grpc.NewServer()
    pb.RegisterAggregateMergeServer(s, agg)

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

/*
Read the pointer to the struct containing both the U and Sigma
Uses atomic.Pointer[T] to ensure atomicity
Research into Golang's memory model
*/
func (agg *Aggregator) RequestGlobalMerge(ctx context.Context, in *pb.DenseMatrix) (*pb.DenseMatrix, error) {
    log.Debug("RECEIVED AGGREGATE REQUEST")
    /*
    U, Sigma = pointer.read()

    agg.channel <- (in.U, in.Sigma)
    U', Sigma' := Merge(U, Sigma, in.U, in.Sigma, r)

    return &Response {
        U: U'
        Sigma: Sigma'
    }, nil
    */
    inUSigma := mat.NewDense(int(in.Rows), int(in.Cols), in.Data)
    agg.matrices<- inUSigma

    aggUSigma := agg.aggregate.Load()

    rows, cols := aggUSigma.Dims()
    return &pb.DenseMatrix{
        Rows: int64(rows),
        Cols: int64(cols),
        Data: aggUSigma.RawMatrix().Data,
    }, nil
}

