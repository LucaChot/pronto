package metrics

import (
	log "github.com/sirupsen/logrus"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

type LocalFPCAAgent struct {
	cpu  linuxproc.Stat
	mem  linuxproc.MemInfo
	disk []linuxproc.DiskStat
	net  []linuxproc.NetworkStat
}

func New() LocalFPCAAgent {
	mc := LocalFPCAAgent{}

    // mc.U = 0
    // mc.Sigma = 0
    // mc.B = []

	go mc.RunLocalUpdates()

	return mc
}

func (mc *LocalFPCAAgent) RunLocalUpdates() {
    for {
        mc.Collect()

        // If len(mc.B) = b
		//mc.FPCAEdge()

    }
}

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
}

func (mc *LocalFPCAAgent) Merge() {
    /*
    Z = U_1.transpose() * U_2
    Q, R = QR(U_2 - (U_1 * Z))

    U', Sigma'' = SVD( [[Sigma_1, Z * Sigma_2], [0, R * Sigma_2]], r)

    U'' = [U_1, Q]U'
    */
}

func (mc *LocalFPCAAgent) Rank() {
    /*
    total_variance = sum(variance(Sigma,i))
    Epsilon = variance(Sigma, r) / total_variance

    if Epsilon < alpha:
        U[:r-1], Sigma[:r-1]

    if Epsilon > beta:
        e = canonical(r+1)
        [U,e], Sigma[:r+1]
    */
}
