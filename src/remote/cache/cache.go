package cache

import (
	"sync"
)

type Cache struct {
    mu          sync.Mutex
    podsRunning int
    informer    PodCountInformer
    signal      func()
}


type PodCountInformer interface {
    Start()
    SetOnChange(func(count int))
}

func (c *Cache) UpdatedPodCount(count int) {
    c.mu.Lock()
    c.podsRunning += count
    go c.signal()
    c.mu.Unlock()
}

func (c *Cache) GetPodCount() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.podsRunning
}

func (c *Cache) SetSignal(signal func()) {
    c.mu.Lock()
    c.signal = signal
    c.mu.Unlock()
}

func New(informer PodCountInformer) *Cache {

    c := &Cache{
        informer: informer,
        signal: func() {},
    }

    c.informer.SetOnChange(c.UpdatedPodCount)
    c.informer.Start()
    return c
}
