package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/LucaChot/pronto/src/remote"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	log "github.com/sirupsen/logrus"
)

func init() {
	flag.Parse()

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

func init() {
	flag.Parse()


	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

    //log.SetOutput(io.Discard)
}

func GetInClusterClientset() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
        return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(80, 100)

	return kubernetes.NewForConfig(config)
}


func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()


    clientset, err := GetInClusterClientset()
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}


	rmt, err := remote.New(
        ctx,
        clientset,
        remote.WithNamespace("basic-sched"))
    if err != nil {
        log.Fatal(err)
    }

    rmt.Start()
}
