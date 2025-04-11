package metrics

import (
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
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
func New() (*MetricsCollector, <-chan *mat.Dense) {
	mc := MetricsCollector{
        ys:  make([]float64, b * d),
        output: make(chan *mat.Dense),
    }
    mc.Y.Store(mat.NewVecDense(d, nil))


    go mc.Collect()

	return &mc, mc.output
}

func (mc *MetricsCollector) Collect() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    for {
        for i := range b {
            <-ticker.C
            row := d * i
            cpu := collectCPU()
            mem := collectRAM()

            mc.ys[row] = cpu
            mc.ys[row + 1] = mem

            mc.Y.Store(mat.NewVecDense(d, []float64{cpu, mem}))
            log.WithFields(log.Fields{
                "CPU" : cpu,
                "MEM" : mem,
            }).Debug("METRIC: SENT Y")
        }
        bT := mat.NewDense(b, d, mc.ys)


        var B mat.Dense
        B.CloneFrom(bT.T())

        mc.output<- &B
        log.Debug("METRIC: SENT B")

    }
}

