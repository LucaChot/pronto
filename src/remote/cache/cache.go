package cache

import (
	"sync"
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
