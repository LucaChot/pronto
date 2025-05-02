package cache

import (
	"log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type ApiInformer struct {
    onChange        func(count int)
    stopEverything  <-chan struct{}
    controller      cache.Controller
}

func (ai *ApiInformer) SetOnChange(onChange func(count int)) {
    ai.onChange = onChange
}

func (ai *ApiInformer) AddPod(obj interface{}) {
    pod := obj.(*corev1.Pod)
    if pod.Status.Phase == corev1.PodRunning {
        log.Printf("%s running on this node", pod.Name)
        ai.onChange(1)
    }
}
func (ai *ApiInformer) UpdatePod(oldObj, newObj interface{}) {
    oldPod := oldObj.(*corev1.Pod)
    newPod := newObj.(*corev1.Pod)

    // If phase changes, update counter
    if oldPod.Status.Phase != corev1.PodRunning && newPod.Status.Phase == corev1.PodRunning {
        log.Printf("%s running on this node", oldPod.Name)
        ai.onChange(1)
    } else if oldPod.Status.Phase == corev1.PodRunning && newPod.Status.Phase != corev1.PodRunning {
        log.Printf("%s terminated", oldPod.Name)
        ai.onChange(-1)
    }
}

func (ai *ApiInformer) DeletePod(obj interface{}) {
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
        ai.onChange(-1)
    }
}

type ApiInformerOptions struct {
    clientSet       clientset.Interface
    nodeName        string
    namespace       string
    stopEverything  <-chan struct{}
}

// ApiInformerOption configures a Scheduler
type ApiInformerOption func(*ApiInformerOptions)

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithClientSet(clientSet clientset.Interface) ApiInformerOption {
	return func(o *ApiInformerOptions) {
		o.clientSet = clientSet
	}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithNodeName(nodeName string) ApiInformerOption {
	return func(o *ApiInformerOptions) {
		o.nodeName = nodeName
	}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithNamespace(namespace string) ApiInformerOption {
	return func(o *ApiInformerOptions) {
		o.namespace = namespace
	}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithStopEverything(stopEverything <-chan struct{}) ApiInformerOption {
	return func(o *ApiInformerOptions) {
		o.stopEverything = stopEverything
	}
}

func NewApiInformer(opts ...ApiInformerOption) *ApiInformer {
    options := ApiInformerOptions{}
	for _, opt := range opts {
		opt(&options)
	}

    lw := cache.NewListWatchFromClient(
        options.clientSet.CoreV1().RESTClient(),
        "pods",
        options.namespace,
        fields.OneTermEqualSelector("spec.nodeName", options.nodeName),
    )

    ai := &ApiInformer{
        stopEverything: options.stopEverything,
    }

    _, controller := cache.NewInformerWithOptions(cache.InformerOptions{
        ListerWatcher: lw,
        ObjectType: &corev1.Pod{},
        Handler: cache.ResourceEventHandlerFuncs{
            AddFunc: ai.AddPod,
            UpdateFunc: ai.UpdatePod,
            DeleteFunc: ai.DeletePod,
        },
    })

    ai.controller = controller

    return ai
}

func (ai *ApiInformer) Start() {
    go ai.controller.Run(ai.stopEverything)
}
