package fpca

import (
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	mt "github.com/LucaChot/pronto/src/matrix"
	pb "github.com/LucaChot/pronto/src/message"
	"gonum.org/v1/gonum/mat"
)

const (
    d = 2
    r = 2
)

type USigmaPair struct {
    U *mat.Dense
    Sigma *mat.DiagDense
}

type FPCAAgent struct {
    inB         <-chan *mat.Dense
    USIgma      atomic.Pointer[USigmaPair]

    adaptive    bool

    b           *mat.Dense
    u           *mat.Dense
    lastU       *mat.Dense
    sigma       *mat.DiagDense
    localU      *mat.Dense
    localSigma  *mat.DiagDense

    r           int
    forget      float64
    enhance     float64
    alpha       float64
    beta        float64
    epsilon     float64

    aggStub     pb.AggregateMergeClient
}

func New(ch <-chan *mat.Dense) *FPCAAgent {
	fp := FPCAAgent{
        inB: ch,
        adaptive: false,
        r: r,
        enhance: 1.1,
        forget: 0.9,
        epsilon: 0,
    }

    fp.u = mat.NewDense(d, r, nil)
    fp.sigma = mat.NewDiagDense(r, nil)
    fp.lastU = fp.u

    fp.USIgma.Store(&USigmaPair{
        U: fp.u,
        Sigma: fp.sigma,
    })

    fp.AsClient()

	go fp.RunLocalUpdates()

	return &fp
}

func (fp *FPCAAgent) RunLocalUpdates() {
    for {
        log.Debug("FPCA: WAITING ON B")
        fp.b = <-fp.inB
        log.Debug("FPCA: RECIEVED B AND BEGINNING FPCA")

		fp.FPCAEdge()

        if !mat.EqualApprox(fp.u, fp.lastU, fp.epsilon) {
            var uSigma mat.Dense
            uSigma.Mul(fp.u, fp.sigma)
            aggUSigma := fp.RequestAgg(&uSigma)
            fp.u, fp.sigma = mt.AggMerge(aggUSigma, &uSigma, fp.r)
        }

        fp.USIgma.Store(&USigmaPair{
            U: fp.u,
            Sigma: fp.sigma,
        })
        log.Debug("FPCA: UPDATED U AND SIGMA")

        fp.lastU = fp.u
    }
}

/*
TODO: Ask andreas about the pseudocode of the paper, in the rank function, it assumes
that the sigma is of size r x r, so what does Sigma_[r+1] do?
TODO: Ask which version of Merge should I implement
*/
func (fp *FPCAAgent) FPCAEdge() {
    /*
    * Update embedding estimates *
    if mc.localU, mc.localSigma = 0,0:
        mc.LocalU, mc.LocalSigma := SVD(B,r)
    else:
        mc.LocalU, mc.LocalSigma := Merge(mc.LocalU, mc.localSigma, B, I, r)

    * Merge with previous estimate *
    mc.GlobalU, mc.GlobalSigma := Merge(mc.LocalU, mc.localSigma, mc.GlobalU, mc.GlobalSigma, r)

    * Adjust the rank based on information it has seen so far *
    mc.GlobalU, mc.GlobalSigma := Rank(mc.GlobalU, mc.GlobalSigma, alpha, beta, r)
    */

    if fp.localU == nil && fp.localSigma == nil {
        fp.localU, fp.localSigma = mt.SVDR(fp.b, fp.r)
    } else {
        _, bc := fp.b.Dims()
        identity := mat.NewDiagDense(bc, nil)
        fp.localU, fp.localSigma = mt.Merge(fp.localU, fp.localSigma, fp.b, identity, fp.r, fp.enhance, fp.forget)
    }

    var uSigma, localUSigma mat.Dense
    uSigma.Mul(fp.u, fp.sigma)
    localUSigma.Mul(fp.localU, fp.localSigma)


    if fp.adaptive {
        /* Pass in rank r+1 so that we can increase the rank in the next step */
        tempU, tempSigma := mt.AggMerge(&uSigma, &localUSigma, fp.r + 1)
        fp.u, fp.sigma = mt.Rank(tempU, tempSigma, fp.r, fp.alpha, fp.beta)
    } else {
        fp.u, fp.sigma = mt.AggMerge(&uSigma, &localUSigma, fp.r)
    }
}

