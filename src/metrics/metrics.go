package metrics

import (
   "gonum.org/v1/gonum/mat"
	linuxproc "github.com/c9s/goprocinfo/linux"
)

const BatchSize = 10
const r = 10

type MetricsCollector struct {
	cpu         linuxproc.Stat
	mem         linuxproc.MemInfo
	disk        []linuxproc.DiskStat
	net         []linuxproc.NetworkStat

    B           *mat.Dense
}

/* Look at potentially parallelising the setup */
func New() *MetricsCollector {
	mc := MetricsCollector{}

	return &mc
}

func (mc *MetricsCollector) Collect() {
    mc.collectCPU()
    mc.collectRAM()
    mc.collectMemory()
    mc.collectNetwork()
}

