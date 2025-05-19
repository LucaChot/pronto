package remote

import (
	"log"
	"sync"
	"time"

	pb "github.com/LucaChot/pronto/src/message"
	"github.com/LucaChot/pronto/src/remote/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Publisher struct {
    mu          sync.Mutex
    nodeName    string
    Signal      float64
    Capacity    float64
    OverProvision   float64
    signalStub pb.SignalServiceClient
    stream grpc.ClientStreamingClient[pb.Signal, pb.SignalAck]
}

func (pub *Publisher) SetSignal(signal float64) {
    pub.Signal = signal
}

func (pub *Publisher) SetCapacity(capacity float64) {
    pub.Capacity = capacity
}

func (pub *Publisher) SetOverProvision(op float64) {
    pub.OverProvision = op
}


func (pub *Publisher) SetUp(ctx context.Context, nodeName string) {
    pub.mu.Lock()
    defer pub.mu.Unlock()
    pub.nodeName = nodeName
    var conn *grpc.ClientConn
    var err error
	for {
        conn, err = grpc.NewClient("pronto-svc.kube-system.svc.cluster.local:50051",
            grpc.WithTransportCredentials(insecure.NewCredentials()))
        if err != nil {
            log.Fatalf("could not connect to controller: %v", err)
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	pub.signalStub = pb.NewSignalServiceClient(conn)
    pub.stream, err = pub.signalStub.StreamSignals(ctx)
    if err != nil {
        log.Fatalf("error creating stream: %v", err)
    }

    go func() {
        <-ctx.Done()
        _, err := pub.stream.CloseAndRecv()
        if err != nil {
            log.Fatalf("error on CloseAndRecv: %v", err)
        }
    }()
}


func (pub *Publisher) Publish(opts ...types.PublishOptions) {
    pub.mu.Lock()
    for _, opt := range opts {
        opt(pub)
    }
    data :=  &pb.Signal{
        Node: pub.nodeName,
        Signal: pub.Signal,
        Capacity: pub.Capacity,
        Overprovision: pub.OverProvision,
    }
    pub.mu.Unlock()

    log.Printf("(publish) data: %#v\n", data)

    if err := pub.stream.Send(data); err != nil {
      log.Fatalf("failed to send data %v: %v", data, err)
    }
}
