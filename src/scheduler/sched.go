package scheduler

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type Scheduler struct {
    // It is expected that changes made via Cache will be observed
	// by NodeLister and Algorithm.
    Cache internalcache.Cache

    // NextPod should be a function that blocks until the next pod
	// is available. We don't use a channel for this, because scheduling
	// a pod may take some amount of time and we don't want pods to get
	// stale while they sit in a channel.
	NextPod func() (*framework.QueuedPodInfo, error)

	// FailureHandler is called upon a scheduling failure.
	FailureHandler FailureHandlerFn

	// SchedulePod tries to schedule the given pod to one of the nodes in the node list.
	// Return a struct of ScheduleResult with the name of suggested host on success,
	// otherwise will return a FitError with reasons.
	SchedulePod func(ctx context.Context, fwk framework.Framework, state *framework.CycleState, pod *v1.Pod) (ScheduleResult, error)

    // Close this to shut down the scheduler.
	StopEverything <-chan struct{}

	// SchedulingQueue holds pods to be scheduled
	SchedulingQueue internalqueue.SchedulingQueue

    client clientset.Interface

	nodeInfoSnapshot *internalcache.Snapshot

	percentageOfNodesToScore int32

	nextStartNodeIndex int
}

func (sched *Scheduler) applyDefaultHandlers() {
	sched.SchedulePod = sched.schedulePod
	sched.FailureHandler = sched.handleSchedulingFailure
}

type schedulerOptions struct {
    name                                string
	kubeConfig                          *restclient.Config
	percentageOfNodesToScore            int32
	podInitialBackoffSeconds            int64
	podMaxBackoffSeconds                int64
	podMaxInUnschedulablePodsDuration   time.Duration
    profiles                            []schedulerapi.KubeSchedulerProfile
	parallelism                         int32
    applyDefaultProfile        bool
}

// Option configures a Scheduler
type Option func(*schedulerOptions)

// WithKubeConfig sets the kube config for Scheduler.
func WithKubeConfig(cfg *restclient.Config) Option {
	return func(o *schedulerOptions) {
		o.kubeConfig = cfg
	}
}

// WithParallelism sets the parallelism for all scheduler algorithms. Default is 16.
func WithParallelism(threads int32) Option {
	return func(o *schedulerOptions) {
		o.parallelism = threads
	}
}

// WithPercentageOfNodesToScore sets percentageOfNodesToScore for Scheduler.
// The default value of 0 will use an adaptive percentage: 50 - (num of nodes)/125.
func WithPercentageOfNodesToScore(percentageOfNodesToScore *int32) Option {
	return func(o *schedulerOptions) {
		if percentageOfNodesToScore != nil {
			o.percentageOfNodesToScore = *percentageOfNodesToScore
		}
	}
}

// WithPodInitialBackoffSeconds sets podInitialBackoffSeconds for Scheduler, the default value is 1
func WithPodInitialBackoffSeconds(podInitialBackoffSeconds int64) Option {
	return func(o *schedulerOptions) {
		o.podInitialBackoffSeconds = podInitialBackoffSeconds
	}
}

// WithPodMaxBackoffSeconds sets podMaxBackoffSeconds for Scheduler, the default value is 10
func WithPodMaxBackoffSeconds(podMaxBackoffSeconds int64) Option {
	return func(o *schedulerOptions) {
		o.podMaxBackoffSeconds = podMaxBackoffSeconds
	}
}

// WithPodMaxInUnschedulablePodsDuration sets podMaxInUnschedulablePodsDuration for PriorityQueue.
func WithPodMaxInUnschedulablePodsDuration(duration time.Duration) Option {
	return func(o *schedulerOptions) {
		o.podMaxInUnschedulablePodsDuration = duration
	}
}

var defaultSchedulerOptions = schedulerOptions{
	percentageOfNodesToScore:          schedulerapi.DefaultPercentageOfNodesToScore,
	podInitialBackoffSeconds:          int64(internalqueue.DefaultPodInitialBackoffDuration.Seconds()),
	podMaxBackoffSeconds:              int64(internalqueue.DefaultPodMaxBackoffDuration.Seconds()),
	podMaxInUnschedulablePodsDuration: internalqueue.DefaultPodMaxInUnschedulablePodsDuration,
	parallelism:                       int32(parallelize.DefaultParallelism),
    // Ideally we would statically set the default profile here, but we can't because
	// creating the default profile may require testing feature gates, which may get
	// set dynamically in tests. Therefore, we delay creating it until New is actually
	// invoked.
	applyDefaultProfile: true,
}

// New returns a Scheduler
func New(ctx context.Context,
	client clientset.Interface,
	informerFactory informers.SharedInformerFactory,
	dynInformerFactory dynamicinformer.DynamicSharedInformerFactory,
	recorderFactory profile.RecorderFactory,
	opts ...Option) (*Scheduler, error) {

	stopEverything := ctx.Done()

	options := defaultSchedulerOptions
	for _, opt := range opts {
		opt(&options)
	}

    if options.applyDefaultProfile {
		var versionedCfg configv1.KubeSchedulerConfiguration
		scheme.Scheme.Default(&versionedCfg)
		cfg := schedulerapi.KubeSchedulerConfiguration{}
		if err := scheme.Scheme.Convert(&versionedCfg, &cfg, nil); err != nil {
			return nil, err
		}
		options.profiles = cfg.Profiles
	}

    registry := frameworkplugins.NewInTreeRegistry()
	if err := registry.Merge(options.frameworkOutOfTreeRegistry); err != nil {
		return nil, err
	}



	podLister := informerFactory.Core().V1().Pods().Lister()
	nodeLister := informerFactory.Core().V1().Nodes().Lister()

	snapshot := internalcache.NewEmptySnapshot()
	waitingPods := frameworkruntime.NewWaitingPodsMap()

	profiles, err := profile.NewMap(ctx, options.profiles, registry, recorderFactory,
		frameworkruntime.WithClientSet(client),
		frameworkruntime.WithKubeConfig(options.kubeConfig),
		frameworkruntime.WithInformerFactory(informerFactory),
		frameworkruntime.WithSnapshotSharedLister(snapshot),
		frameworkruntime.WithParallelism(int(options.parallelism)),
		frameworkruntime.WithWaitingPods(waitingPods),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing profiles: %v", err)
	}

	if len(profiles) == 0 {
		return nil, errors.New("at least one profile is required")
	}

	preEnqueuePluginMap := make(map[string][]framework.PreEnqueuePlugin)
	queueingHintsPerProfile := make(internalqueue.QueueingHintMapPerProfile)
	var returnErr error
	for profileName, profile := range profiles {
		preEnqueuePluginMap[profileName] = profile.PreEnqueuePlugins()
		queueingHintsPerProfile[profileName], err = buildQueueingHintMap(ctx, profile.EnqueueExtensions())
		if err != nil {
			returnErr = errors.Join(returnErr, err)
		}
	}

	if returnErr != nil {
		return nil, returnErr
	}

	podQueue := internalqueue.NewSchedulingQueue(
		profiles[options.profiles[0].SchedulerName].QueueSortFunc(),
		informerFactory,
		internalqueue.WithClock(options.clock),
		internalqueue.WithPodInitialBackoffDuration(time.Duration(options.podInitialBackoffSeconds)*time.Second),
		internalqueue.WithPodMaxBackoffDuration(time.Duration(options.podMaxBackoffSeconds)*time.Second),
		internalqueue.WithPodLister(podLister),
		internalqueue.WithPodMaxInUnschedulablePodsDuration(options.podMaxInUnschedulablePodsDuration),
		internalqueue.WithPreEnqueuePluginMap(preEnqueuePluginMap),
		internalqueue.WithQueueingHintMapPerProfile(queueingHintsPerProfile),
		internalqueue.WithPluginMetricsSamplePercent(pluginMetricsSamplePercent),
		internalqueue.WithMetricsRecorder(*metricsRecorder),
	)

	for _, fwk := range profiles {
		fwk.SetPodNominator(podQueue)
		fwk.SetPodActivator(podQueue)
	}

	schedulerCache := internalcache.New(ctx, durationToExpireAssumedPod)

	// Setup cache debugger.
	debugger := cachedebugger.New(nodeLister, podLister, schedulerCache, podQueue)
	debugger.ListenForSignal(ctx)

	sched := &Scheduler{
		Cache:                    schedulerCache,
		client:                   client,
		nodeInfoSnapshot:         snapshot,
		percentageOfNodesToScore: options.percentageOfNodesToScore,
		Extenders:                extenders,
		StopEverything:           stopEverything,
		SchedulingQueue:          podQueue,
		Profiles:                 profiles,
		logger:                   logger,
	}
	sched.NextPod = podQueue.Pop
	sched.applyDefaultHandlers()

	if err = addAllEventHandlers(sched, informerFactory, dynInformerFactory, resourceClaimCache, resourceSliceTracker, unionedGVKs(queueingHintsPerProfile)); err != nil {
		return nil, fmt.Errorf("adding event handlers: %w", err)
	}

	return sched, nil
}



/* Creates a new CentralScheduler */
func New(name string, opts ...Option) *Scheduler {
    /* Initialise scheduler values */
	ctl := &Scheduler{
		name: name,
        rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
    }

    for _, opt := range opts {
		opt(ctl)
	}
	return ctl
}

func (ctl *Scheduler) Init() error {
    err := ctl.findNodes()
    if err != nil {
        return fmt.Errorf("failed to find nodes: %w", err)
	}

    ctl.nodeSignals = make([]atomic.Uint64, len(ctl.nodeMap))
    for node := range(len(ctl.nodeMap)) {
        ctl.nodeSignals[node].Store(math.Float64bits(0.0))
    }

    return nil
}

func (ctl *Scheduler) Start(ctx context.Context) {
    ctl.startPlacementServer(ctx)
    ctl.startBindPodWorkers(ctx, 4)
    ctl.startWatchHandler(ctx)
}

/* Core Scheduling loop */
func (ctl *Scheduler) RunScheduler(ctx context.Context) error {
    log.Print("(sched) starting main scheduler loop")
    for {
        for {
            select {
            case <-ctx.Done():
                log.Print("(sched) context cancelled, exiting event loop")
                return nil
            case pod := <- ctl.retryQueue:
                log.Printf("(sched) %s <-retry", pod.Name)
                /* Find a node to place the pod */
                node, err := ctl.SchedulePod()
                if err != nil {
                    log.Printf("(sched) %s\n(sched) retry <-%s : total %d", err.Error(), pod.Name, len(ctl.retryQueue)+1)
                    ctl.retryQueue <- pod
                    time.Sleep(10 * time.Millisecond)
                    break
                }

                log.Printf("(sched) bind <-%s", pod.Name)
                ctl.bindQueue <- &podBind{
                    pod:    pod,
                    node:   node.ID,
                }
            case pod := <- ctl.watchQueue:
                log.Printf("(sched) %s <-watch", pod.Name)

                /* Find a node to place the pod */
                node, err := ctl.SchedulePod()
                if err != nil {
                    log.Printf("(sched) %s\n(sched) retry <-%s : total %d", err.Error(), pod.Name, len(ctl.retryQueue)+1)
                    ctl.retryQueue <- pod
                    time.Sleep(10 * time.Millisecond)
                    break
                }

                log.Printf("(sched) bind <-%s", pod.Name)
                ctl.bindQueue <- &podBind{
                    pod:    pod,
                    node:   node.ID,
                }
            }
        }
    }
}

