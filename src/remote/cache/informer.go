package cache

import (
	"log"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Cache struct {
    mu          sync.Mutex
    podsRunning int
    perPodCost  float64
}

func (c *Cache) AddPod(obj interface{}) {
    pod := obj.(*corev1.Pod)
    if pod.Status.Phase == corev1.PodRunning {
        log.Printf("%s running on this node", pod.Name)
        c.mu.Lock()
        c.podsRunning++
        c.mu.Unlock()
    }
}
func (c *Cache) UpdatePod(oldObj, newObj interface{}) {
    oldPod := oldObj.(*corev1.Pod)
    newPod := newObj.(*corev1.Pod)

    // If phase changes, update counter
    if oldPod.Status.Phase != corev1.PodRunning && newPod.Status.Phase == corev1.PodRunning {
        log.Printf("%s running on this node", oldPod.Name)
        c.mu.Lock()
        c.podsRunning++
        c.mu.Unlock()
    } else if oldPod.Status.Phase == corev1.PodRunning && newPod.Status.Phase != corev1.PodRunning {
        log.Printf("%s terminated", oldPod.Name)
        c.mu.Lock()
        c.podsRunning--
        c.mu.Unlock()
    }
}

func (c *Cache) DeletePod(obj interface{}) {
    pod, ok := obj.(*corev1.Pod)
    if !ok {
        tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
        if !ok {
            return
        }
        pod, ok = tombstone.Obj.(*corev1.Pod)
        if !ok {
            return
        }
    }
    if pod.Status.Phase == corev1.PodRunning {
        log.Printf("%s terminated", pod.Name)
        c.mu.Lock()
        c.podsRunning--
        c.mu.Unlock()
    }
}

type cacheOptions struct {
    clientSet       clientset.Interface
    nodeName        string
    namespace       string
    stopEverything  <-chan struct{}
}

// Option configures a Scheduler
type Option func(*cacheOptions)

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithClientSet(clientSet clientset.Interface) Option {
	return func(o *cacheOptions) {
		o.clientSet = clientSet
	}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithNodeName(nodeName string) Option {
	return func(o *cacheOptions) {
		o.nodeName = nodeName
	}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithNamespace(namespace string) Option {
	return func(o *cacheOptions) {
		o.namespace = namespace
	}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithStopEverything(stopEverything <-chan struct{}) Option {
	return func(o *cacheOptions) {
		o.stopEverything = stopEverything
	}
}

func New(opts ...Option) *Cache {
    c := &Cache{}

    options := cacheOptions{}
	for _, opt := range opts {
		opt(&options)
	}

    lw := cache.NewListWatchFromClient(
        options.clientSet.CoreV1().RESTClient(),
        "pods",
        options.namespace,
        fields.OneTermEqualSelector("spec.nodeName", options.nodeName),
    )

    _, controller := cache.NewInformerWithOptions(cache.InformerOptions{
        ListerWatcher: lw,
        ObjectType: &corev1.Pod{},
        Handler: cache.ResourceEventHandlerFuncs{
            AddFunc: c.AddPod,
            UpdateFunc: c.UpdatePod,
            DeleteFunc: c.DeletePod,
        },
    })
    go controller.Run(options.stopEverything)
    return c
}

func (c *Cache) GetPodCount() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.podsRunning
}

