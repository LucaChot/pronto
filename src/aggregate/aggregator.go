package aggregate

/*
Core Components:
- Aggregator thread
- gRPC threads

Idea:
Aggregator thread takes a queue of (U,Sigma)s and performs the merge operation.

The gRPC threads from the remote schedulers read the address of the pointer to
the current aggregated matrix. It then performs its on merge operation.


This method reduces overall latency. We can avoid gRPC requests from queuing.
In addition, if we did not make gRPC requests queue, merge operations could be lost.

So we use a combination of central thread and gRPC calls to reduce latency while
ensuring consistency


FAULT TOLERANCE:
We could add fault tolerance by having the remote schedulers use the pod service
and so when a pod restarts, they are still able to communicate with this pod.
*/

type Aggregator struct {
    // pipe
    //pb.UnimplementedAggregateServer
}

func New() (Aggregator) {
    agg := Aggregator {}

    agg.startAggregateServer()

    return agg
}

func (agg *Aggregator) Aggregate()  {
    /*
    for {
        inU, inSigma <- agg.ch

        * Uses sync/atomic pointer *
        U, Sigma = pointer.read()

        U', Sigma' := Merge(U, Sigma, inU, inSigma, r)

        pointer.update(U', Sigma')
    }
    */
}

