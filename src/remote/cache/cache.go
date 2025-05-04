package cache

import (
	"sync"

	clientset "k8s.io/client-go/kubernetes"
)

type Cache struct {
    mu          sync.Mutex
    podsRunning int
    informer    Informer
}


type Informer interface {
    Start()
    SetOnChange(func(count int))
}

func (c *Cache) UpdatedPodCount(count int) {
    c.mu.Lock()
    c.podsRunning += count
    c.mu.Unlock()
}

func (c *Cache) GetPodCount() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.podsRunning
}

func New(informer Informer) *Cache {
    c := &Cache{
        informer: informer,
    }

    c.informer.SetOnChange(c.UpdatedPodCount)
    c.informer.Start()
    return c
}

type InformerOptions struct {
    clientSet       clientset.Interface
    nodeName        string
    namespace       string
    stopEverything  <-chan struct{}
    socketPath      string
    newInformer     func(opt ...InformerOption) Informer
}

type InformerOption func(*InformerOptions)
