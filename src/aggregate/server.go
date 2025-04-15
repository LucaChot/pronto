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
    }).Debug("SERVER: STARTED AGGREGATE SERVER")

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
func (agg *Aggregator) RequestAggMerge(ctx context.Context, in *pb.DenseMatrix) (*pb.DenseMatrix, error) {
    inUSigma := mat.NewDense(int(in.Rows), int(in.Cols), in.Data)
    agg.matrices<- inUSigma

    aggU := agg.aggU.Load()

    if aggU == nil {
        log.Debug("SERVER: NO AGGREGATE RETURNED INPUT")
        return nil, nil
    }

    log.Debug("SERVER: RETURNED AGGREGATE")

    rows, cols := aggU.Dims()
    return &pb.DenseMatrix{
        Rows: int64(rows),
        Cols: int64(cols),
        Data: aggU.RawMatrix().Data,
    }, nil
}

/*
Move Merge into Aggregate Server
U, Sigma = pointer.read()

agg.channel <- (in.U, in.Sigma)
U', Sigma' := Merge(U, Sigma, in.U, in.Sigma, r)

return &Response {
    U: U'
    Sigma: Sigma'
}, nil
*/
