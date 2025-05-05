package cache

import (
	"sync"
	"time"
)

type EventCache struct {
    mu          sync.Mutex
    podTimers   map[string]*time.Timer
    informer    EventInformer
    signal      func()
    interval    time.Duration
}


type EventInformer interface {
    Start()
    SetOnEvent(func(name string))
}

func (ec *EventCache) OnEvent(name string) {
    ec.mu.Lock()
    timer := ec.podTimers[name]
    if timer != nil {
        timer.Stop()
    }

    ec.podTimers[name] = time.AfterFunc(ec.interval, func() {
        ec.mu.Lock()
        delete(ec.podTimers, name)
        ec.mu.Unlock()
        ec.signal()
    })
    ec.mu.Unlock()
}

func (ec *EventCache) isWaiting() int {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return len(ec.podTimers)
}

func (ec *EventCache) SetSignal(signal func()) {
    ec.mu.Lock()
    ec.signal = signal
    ec.mu.Unlock()
}

type eventCacheOptions struct {
    interval    time.Duration
}


// Option configures a Scheduler
type Option func(*eventCacheOptions)

func WithInterval(interval time.Duration) Option {
	return func(o *eventCacheOptions) {
		o.interval = interval
	}
}

var defaultEventCacheOptions = eventCacheOptions{
    interval:   100 * time.Millisecond,
}

func NewEventCache(informer EventInformer, opts ...Option) *EventCache {
	options := defaultEventCacheOptions
	for _, opt := range opts {
		opt(&options)
	}

    c := &EventCache{
        informer: informer,
        signal: func() {},
        interval: options.interval,
    }

    c.informer.SetOnEvent(c.OnEvent)
    c.informer.Start()
    return c
}
