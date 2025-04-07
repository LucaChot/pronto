package main

import (
	"flag"

	"github.com/LucaChot/pronto/src/remote"

	log "github.com/sirupsen/logrus"
)

func init() {
	flag.Parse()

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

func main() {
	rmt := remote.New()
    rmt.Schedule()
}
