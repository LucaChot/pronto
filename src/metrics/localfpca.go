package metrics

import (
	log "github.com/sirupsen/logrus"

   "gonum.org/v1/gonum/mat"
	linuxproc "github.com/c9s/goprocinfo/linux"
    mt "github.com/LucaChot/pronto/src/matrix"
)

const BatchSize = 10
const r = 10

type LocalFPCAAgent struct {
	cpu         linuxproc.Stat
	mem         linuxproc.MemInfo
	disk        []linuxproc.DiskStat
	net         []linuxproc.NetworkStat
    U           *mat.Dense
    lastU           *mat.Dense
    Sigma       *mat.DiagDense
    B           *mat.Dense
    localU      *mat.Dense
    localSigma  *mat.DiagDense
    r           int
    forget      float64
    enhance     float64
    alpha       float64
    beta        float64
    epsilon     float64
}

func New() LocalFPCAAgent {
	mc := LocalFPCAAgent{
        r: r,
    }

	go mc.RunLocalUpdates()

	return mc
}

func (mc *LocalFPCAAgent) RunLocalUpdates() {
    for {
        mc.Collect()

        if _, c := mc.B.Dims(); c == BatchSize {
		    mc.FPCAEdge()
            mc.B.Reset()
        }

        if mat.EqualApprox(mc.U, mc.lastU, mc.epsilon) {
            //mc.RequestU
        }
        mc.U = mc.lastU
    }
}

/*
TODO: Ask andreas about the pseudocode of the paper, in the rank function, it assumes
that the sigma is of size r x r, so what does Sigma_[r+1] do?
TODO: Ask which version of Merge should I implement
*/
func (mc *LocalFPCAAgent) FPCAEdge() {
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

    if mc.localU == nil && mc.localSigma == nil {
        mc.localU, mc.localSigma = mt.SVDR(mc.B, mc.r)
    } else {
        _, bc := mc.B.Dims()
        identity := mat.NewDiagDense(bc, nil)
        mc.localU, mc.localSigma = mt.Merge(mc.localU, mc.localSigma, mc.B, identity, mc.r, mc.enhance, mc.forget)
    }

    /* Pass in rank r+1 so that we can increase the rank in the next step */
    tempU, tempSigma := mt.AggMerge(mc.U, mc.Sigma, mc.localU, mc.localSigma, mc.r)

    mc.U, mc.Sigma = mt.Rank(tempU, tempSigma, mc.r, mc.alpha, mc.beta)
}

