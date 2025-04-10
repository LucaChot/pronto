package metrics

import (
	"sync/atomic"

	"gonum.org/v1/gonum/mat"
)

const (
    d = 2
    b = 10
)

type MetricsCollector struct {
    ys       []float64
    Y       atomic.Pointer[mat.VecDense]
    entries int
    output  chan *mat.Dense
}

/* Look at potentially parallelising the setup */
func New() *MetricsCollector {
	mc := MetricsCollector{
        ys:  make([]float64, b * d),
    }

	return &mc
}

func (mc *MetricsCollector) Collect() {
    for {
        for i := range b {
            row := d * i
            cpu := collectCPU()
            mem := collectRAM()

            mc.ys[row] = cpu
            mc.ys[row + 1] = mem

            mc.Y.Store(mat.NewVecDense(d, []float64{cpu, mem}))
        }
        bT := mat.NewDense(b, d, mc.ys)


        var B mat.Dense
        B.CloneFrom(bT.T())

        mc.output<- &B
    }
}

