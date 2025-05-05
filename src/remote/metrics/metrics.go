package metrics

import (
	"sync/atomic"
	"time"

	"gonum.org/v1/gonum/mat"
)

const (
    d = 2
    b = 10
)

type MetricsCollector struct {
    ys       []float64
    Y       atomic.Pointer[[]float64]
    entries int
    output  chan *mat.Dense
}

/* Look at potentially parallelising the setup */
func New() (*MetricsCollector, <-chan *mat.Dense) {
	mc := MetricsCollector{
        ys:  make([]float64, b * d),
        output: make(chan *mat.Dense),
    }
    mc.Y.Store(nil)


    go mc.Collect()

	return &mc, mc.output
}

/*
TODO: Investigate difference between ticker and time.Sleep()
*/
func (mc *MetricsCollector) Collect() {
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()
    for {
        for i := range b {
            <-ticker.C
            row := d * i
            cpu := collectCPU()
            mem := collectRAM()

            mc.ys[row] = cpu
            mc.ys[row + 1] = mem

            mc.Y.Store(&[]float64{cpu, mem})
        }
        bT := mat.NewDense(b, d, mc.ys)


        var B mat.Dense
        B.CloneFrom(bT.T())

        mc.output<- &B
    }
}

