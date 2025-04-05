package metrics

import (
	log "github.com/sirupsen/logrus"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

func (mc *LocalFPCAAgent) Collect() {
    mc.collectCPU()
    mc.collectRAM()
    mc.collectMemory()
    mc.collectNetwork()
}

func (mc *LocalFPCAAgent) collectCPU() {
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Fatal("FAILED TO READ /proc/stat")
	}
	mc.cpu = *stat
}

func (mc *LocalFPCAAgent) collectRAM() {
	stat, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Fatal("FAILED TO READ /proc/net/dev")
	}
	mc.mem = *stat
}

/*
TODO: Look into whether I should collect ReadDisk() vs ReadDiskStat()
*/
func (mc *LocalFPCAAgent) collectMemory() {
	stat, err := linuxproc.ReadDiskStats("/proc/diskstats")
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Fatal("FAILED TO READ /proc/net/dev")
	}
	mc.disk = stat
}

func (mc *LocalFPCAAgent) collectNetwork() {
	stat, err := linuxproc.ReadNetworkStat("/proc/net/dev")
	if err != nil {
		log.WithFields(log.Fields{
			"ERROR": err,
		}).Fatal("FAILED TO READ /proc/net/dev")
	}
	mc.net = stat
}
