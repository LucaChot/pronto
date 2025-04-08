package fpca

import (

	pb "github.com/LucaChot/pronto/src/message"
   "gonum.org/v1/gonum/mat"
    mt "github.com/LucaChot/pronto/src/matrix"
    mets "github.com/LucaChot/pronto/src/metrics"
)

const BatchSize = 10
const r = 10

type FPCAAgent struct {
    Mc         *mets.MetricsCollector

    U           *mat.Dense
    lastU       *mat.Dense
    Sigma       *mat.DiagDense
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

func New() *FPCAAgent {
	fp := FPCAAgent{
        r: r,
    }

	go fp.RunLocalUpdates()

	return &fp
}

func (fp *FPCAAgent) RunLocalUpdates() {
    for {
        fp.Mc.Collect()

        if _, c := fp.Mc.B.Dims(); c == BatchSize {
		    fp.FPCAEdge()
            fp.Mc.B.Reset()
        }

        if !mat.EqualApprox(fp.U, fp.lastU, fp.epsilon) {
            var uSigma *mat.Dense
            uSigma.Mul(fp.U, fp.Sigma)
            aggUSigma := fp.RequestAgg(uSigma)
            fp.U, fp.Sigma = mt.AggMerge(aggUSigma, uSigma, fp.r)
        }
        fp.U = fp.lastU
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
        fp.localU, fp.localSigma = mt.SVDR(fp.Mc.B, fp.r)
    } else {
        _, bc := fp.Mc.B.Dims()
        identity := mat.NewDiagDense(bc, nil)
        fp.localU, fp.localSigma = mt.Merge(fp.localU, fp.localSigma, fp.Mc.B, identity, fp.r, fp.enhance, fp.forget)
    }

    var uSigma, localUSigma *mat.Dense
    uSigma.Mul(fp.U, fp.Sigma)
    localUSigma.Mul(fp.localU, fp.localSigma)

    /* Pass in rank r+1 so that we can increase the rank in the next step */
    tempU, tempSigma := mt.AggMerge(uSigma, localUSigma, fp.r + 1)

    fp.U, fp.Sigma = mt.Rank(tempU, tempSigma, fp.r, fp.alpha, fp.beta)
}

