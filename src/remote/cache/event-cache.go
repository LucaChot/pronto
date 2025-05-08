package cache

import (
	"log"
	"sync"
	"time"
)

type EventCache struct {
    mu          sync.Mutex
    informer    EventInformer

    timer       *time.Timer
    ends        time.Time

	creating 	        map[string]struct{}
	deleting	        map[string]struct{}
	podContainers       map[string]*ContainerInfo
	changeInPodCount	int
    lastSignal          float64

    signal          func() (float64, error)
    updateCost   func(podCountDiff int, signalDiff float64) float64
    publish         func(signal, podCost float64)


    createInterval      time.Duration
    deleteInterval      time.Duration
}


type Topic int

const (
    Create Topic = iota
    Start
    Exit
    Delete
)

type ContainerInfo struct {
	creating    int
	running 	int
	deleting		int
}

type Event struct {
	containerID string
	podID       string
	topic       Topic
}



type EventInformer interface {
    Start()
    SetOnEvent(func(e Event))
}

func (ec *EventCache) isWaiting() bool {
	if len(ec.creating) > 0 {
		return true
	}
	if len(ec.deleting) > 0 {
		return true
	}
    if time.Now().Before(ec.ends) {
        return true
    }
	return false
}

func (ec *EventCache) GetLastSignal() float64 {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return ec.lastSignal
}

func (ec *EventCache) GetChangeInPodCount() int {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return ec.changeInPodCount
}

func (ec *EventCache) IsWaiting() bool {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return ec.isWaiting()
}

func (ec *EventCache) OnTrigger() {
    ec.mu.Lock()
	if ec.isWaiting() {
        log.Print("(cache) event prevented trigger")
        ec.mu.Unlock()
		return
	}
    changeInPodCount := ec.changeInPodCount
    lastSignal := ec.lastSignal
    ec.changeInPodCount = 0
    ec.mu.Unlock()

    log.Printf("(cache) running signal and cost generation")
	signal, err := ec.signal()
    if err != nil {
        log.Printf("(cache) error generating signal: %s", err)
        return
    }
	cost := ec.updateCost(changeInPodCount, signal - lastSignal)
    log.Printf("(cache) signal = %.4f", signal)
    log.Printf("(cache) per-pod cost = %.4f", cost)
	ec.publish(signal, cost)
}

func (ec *EventCache) OnEvent(e Event) {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    log.Printf("(cache) received an event: %+v", e)
    if !ec.isWaiting() {
        signal, err := ec.signal()
        if err != nil {
            log.Printf("(cache) error generating signal: %s", err)
        } else {
            ec.lastSignal = signal
        }
    }
	switch e.topic {
    case Create:
        if _, ok := ec.podContainers[e.podID]; !ok {
            ec.changeInPodCount += 1
            ec.podContainers[e.podID] = &ContainerInfo{}
        }
        ec.podContainers[e.podID].creating += 1
        ec.creating[e.containerID] = struct{}{}
    case Start:
        delete(ec.creating, e.containerID)
        containerInfo := ec.podContainers[e.podID]
        containerInfo.creating -= 1
        containerInfo.running += 1
        if containerInfo.creating == 0 {
            ends := time.Now().Add(ec.createInterval)
            if ends.After(ec.ends) {
                if ec.timer != nil {
                    ec.timer.Stop()
                }
                ec.timer = time.AfterFunc(ec.createInterval, ec.OnTrigger)
                ec.ends = ends
            }
        }
    case Exit:
        containerInfo := ec.podContainers[e.podID]
        containerInfo.running -= 1
        if containerInfo.running == 0 {
            ec.changeInPodCount -= 1
        }
        containerInfo.deleting += 1
        ec.deleting[e.containerID] = struct{}{}
    case Delete:
        delete(ec.deleting, e.containerID)
        containerInfo := ec.podContainers[e.podID]
        containerInfo.deleting -= 1
        if containerInfo.deleting == 0 {
            ends := time.Now().Add(ec.deleteInterval)
            if ends.After(ec.ends) {
                if ec.timer != nil {
                    ec.timer.Stop()
                }
                ec.timer = time.AfterFunc(ec.deleteInterval, ec.OnTrigger)
                ec.ends = ends
            }
        }
	}
}

func (ec *EventCache) SetSignal(signal func() (float64, error)) {
    ec.mu.Lock()
    ec.signal = signal
    ec.mu.Unlock()
}

func (ec *EventCache) SetUpdateCost(updateCost func(podCountDiff int, signalDiff float64) float64) {
    ec.mu.Lock()
    ec.updateCost = updateCost
    ec.mu.Unlock()
}

func (ec *EventCache) SetPublish(publish func(signal float64, podCost float64)) {
    ec.mu.Lock()
    ec.publish = publish
    ec.mu.Unlock()
}

type eventCacheOptions struct {
    createInterval    time.Duration
    deleteInterval    time.Duration
}


// Option configures a Scheduler
type Option func(*eventCacheOptions)

func WithCreateInterval(interval time.Duration) Option {
	return func(o *eventCacheOptions) {
		o.createInterval = interval
	}
}

func WithExitInterval(interval time.Duration) Option {
	return func(o *eventCacheOptions) {
		o.deleteInterval = interval
	}
}

var defaultEventCacheOptions = eventCacheOptions{
    createInterval:     100 * time.Millisecond,
    deleteInterval:     300 * time.Millisecond,
}

func NewEventCache(informer EventInformer, opts ...Option) *EventCache {
	options := defaultEventCacheOptions
	for _, opt := range opts {
		opt(&options)
	}

    c := &EventCache{
        informer: informer,
        creating: make(map[string]struct{}),
        deleting: make(map[string]struct{}),
        podContainers: make(map[string]*ContainerInfo),

        ends: time.Now(),

        signal: func() (float64, error) {return 0.0, nil},
        updateCost: func(podCountDiff int, signalDiff float64) float64 {return 0.0},
        publish: func(signal float64, podCost float64) {},

        createInterval: options.createInterval,
        deleteInterval: options.deleteInterval,
    }

    c.informer.SetOnEvent(c.OnEvent)
    c.informer.Start()
    return c
}
