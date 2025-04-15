package fpca

import (
	"net"
	"time"

	pb "github.com/LucaChot/pronto/src/message"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"gonum.org/v1/gonum/mat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (fp *FPCAAgent) AsClient() {
	aggAddr := findAggAddr()
	fp.connectToAgg(aggAddr)
}

func findAggAddr() net.IP {
	for {
		ips, err := net.LookupIP("agg-svc.basic-sched.svc.cluster.local")
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

func (fp *FPCAAgent) connectToAgg(aggAddr net.IP) {

    conn, err := grpc.NewClient(aggAddr.String()+":50052",
        grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"for":   "placement",
		}).Fatal("Could not connect to controller")
	}

	fp.aggStub = pb.NewAggregateMergeClient(conn)

}

func (fp *FPCAAgent) SendAggRequest(inM *mat.Dense) (*mat.Dense) {
    log.Debug("FPCA: REQUESTING AGGREGATION")
    ctx := context.Background()

    rows, cols := inM.Dims()
    outM, err := fp.aggStub.RequestAggMerge(ctx, &pb.DenseMatrix{ Rows: int64(rows),
        Cols: int64(cols),
        Data: inM.RawMatrix().Data,
    })

    if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Fatal("FAILED AGGREGATION")
	}

    log.Debug("FPCA: COMPLETED AGGREGATION")

    if outM == nil {
        return nil
    }

    return mat.NewDense(int(outM.Rows), int(outM.Cols), outM.Data)
}

