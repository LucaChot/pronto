package metrics

import (
	log "github.com/sirupsen/logrus"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func collectCPU() float64 {
    stat, err := cpu.Percent(0, false)
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Fatal("FAILED TO READ /proc/stat")
	}
    return stat[0] / 100
}

func collectRAM() (float64) {
    stat, err := mem.VirtualMemory()
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Fatal("FAILED TO READ /proc/meminfo")
	}
	return stat.UsedPercent / 100
}

/*
TODO: Look into whether I should collect ReadDisk() vs ReadDiskStat()
*/
func collectMemory() {
    /* TODO: Look into how to include Disk Statistics */
    /* Potentially attach the node directories to the pods */
}

func collectNetwork() {
    /* TODO: Look into how to include Net Statistics without knowing capacity*/
    /* Look into number of packets dropped */
}
