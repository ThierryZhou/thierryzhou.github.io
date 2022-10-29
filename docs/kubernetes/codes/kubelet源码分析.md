## kubelet入口函数
kubelet 在 Node 节点上负责 Pod 的创建、销毁、监控上报等核心流程，通过 Cobra 命令行解析参数启动二进制可执行文件，
Cobra启动入口在kubernetes/cmd/kubelet/kubelet.go文件中，经过封装后，实际的入口如下：
```go
...
// kubernetes/cmd/kubelet/app/server.go  
func run(ctx context.Context, s *options.KubeletServer, kubeDeps *kubelet.Dependencies, featureGate featuregate.FeatureGate) (err error) {
	...
	// 初始化runtime service
	err = kubelet.PreInitRuntimeService(&s.KubeletConfiguration, kubeDeps, s.RemoteRuntimeEndpoint, s.RemoteImageEndpoint)
	if err != nil {
		return err
	}

	// 启动Kubelet主流程，启动Pod事件监听
	if err := RunKubelet(s, kubeDeps, s.RunOnce); err != nil {
		return err
	}
	...
}
...
```
### PreInitRuntimeService
负责初始容器运行时、镜像、资源收集三个GRPC服务
```go
func PreInitRuntimeService(kubeCfg *kubeletconfiginternal.KubeletConfiguration,
	kubeDeps *Dependencies,
	remoteRuntimeEndpoint string,
	remoteImageEndpoint string) error {
	// remoteImageEndpoint如果为空，则使用remoteRuntimeEndpoint
	if remoteRuntimeEndpoint != "" && remoteImageEndpoint == "" {
		remoteImageEndpoint = remoteRuntimeEndpoint
	}

    // 从配置文件中读取endpoint信息然后初始化容器运行时服务、镜像服务、资源metrics收集服务
	var err error
	if kubeDeps.RemoteRuntimeService, err = remote.NewRemoteRuntimeService(remoteRuntimeEndpoint, kubeCfg.RuntimeRequestTimeout.Duration, kubeDeps.TracerProvider); err != nil {
		return err
	}
	if kubeDeps.RemoteImageService, err = remote.NewRemoteImageService(remoteImageEndpoint, kubeCfg.RuntimeRequestTimeout.Duration); err != nil {
		return err
	}

	kubeDeps.useLegacyCadvisorStats = cadvisor.UsingLegacyCadvisorStats(remoteRuntimeEndpoint)

	return nil
}

```
### RunKubelet
调用了[createAndInitKubelet](#createAndInitKubelet)，[startKubelet](#startKubelet)分析
```go
// kubernets/cmd/kubelet/app/server.go

func RunKubelet(kubeServer *options.KubeletServer, kubeDeps *kubelet.Dependencies, runOnce bool) error {
	...
    // 初始化、创建各种对象
	k, err := createAndInitKubelet(kubeServer,
		kubeDeps,
		hostname,
		hostnameOverridden,
		nodeName,
		nodeIPs)
	if err != nil {
		return fmt.Errorf("failed to create kubelet: %w", err)
	}

    ...

	// process pods and exit.
	if runOnce {
		if _, err := k.RunOnce(podCfg.Updates()); err != nil {
			return fmt.Errorf("runonce failed: %w", err)
		}
		klog.InfoS("Started kubelet as runonce")
	} else {
        // 启动kubelet
		startKubelet(k, podCfg, &kubeServer.KubeletConfiguration, kubeDeps, kubeServer.EnableServer)
		klog.InfoS("Started kubelet")
	}
	return nil
}
```

### createAndInitKubelet
调用NewMainKubeletkubelet创建Kubelet对象，并调用对象的成员接口StartGarbageCollection启动垃圾回收后台事件。
```go
// kubernets/cmd/kubelet/app/server.go
func createAndInitKubelet(kubeServer *options.KubeletServer,
	kubeDeps *kubelet.Dependencies,
	hostname string,
	hostnameOverridden bool,
	nodeName types.NodeName,
	nodeIPs []net.IP) (k kubelet.Bootstrap, err error) {

    // 对象初始化的驻留在在NewMainKubelet中，由于次函数主要是各种对象创建和成员变量复制，这里就不展开此函数了
	k, err = kubelet.NewMainKubelet(
		// 省略所有参数
		...
    )
	if err != nil {
		return nil, err
	}

    // 上报Starting kubelet的事件
	k.BirthCry()

    // 启动GC线程
	k.StartGarbageCollection()

	return k, nil
}

```

### startKubelet
负责启动kuelet，并监听服务端口处理网络请求。
```go
// kubernets/cmd/kubelet/app/server.go
func startKubelet(k kubelet.Bootstrap, podCfg *config.PodConfig, kubeCfg *kubeletconfiginternal.KubeletConfiguration, kubeDeps *kubelet.Dependencies, enableServer bool) {
	// 启动kubelet
	go k.Run(podCfg.Updates())

	// 监听端口，启动kubelet服务
	if enableServer {
		go k.ListenAndServe(kubeCfg, kubeDeps.TLSOptions, kubeDeps.Auth, kubeDeps.TracerProvider)
	}
	if kubeCfg.ReadOnlyPort > 0 {
		go k.ListenAndServeReadOnly(netutils.ParseIPSloppy(kubeCfg.Address), uint(kubeCfg.ReadOnlyPort))
	}
	if utilfeature.DefaultFeatureGate.Enabled(features.KubeletPodResources) {
		go k.ListenAndServePodResources()
	}
}

```
# Kubelet.Run
Pod 创建/删除等事件的处理流程采用 channel 生产者-消费者模型实现，生产者的流程封装在[PLEG](#PLEG)(Pod Lifecycle Event Generator) 进行 Pod 生命周期管理。消费者流程封装在[syncLoop](#Kubelet.syncLoop)函数中。
```go
// kubernets/pkg/kubelet/kubelet.go
func (kl *Kubelet) Run(updates <-chan kubetypes.PodUpdate) {
	if kl.logServer == nil {
		kl.logServer = http.StripPrefix("/logs/", http.FileServer(http.Dir("/var/log/")))
	}
	if kl.kubeClient == nil {
		klog.InfoS("No API server defined - no node status update will be sent")
	}

	// 与厂商相关资源同步，略过
	if kl.cloudResourceSyncManager != nil {
		go kl.cloudResourceSyncManager.Run(wait.NeverStop)
	}

	// 初始化子模块(如：PrometheusMetric模块、镜像管理器、oom监听器、资源管理器等)。
	if err := kl.initializeModules(); err != nil {
		kl.recorder.Eventf(kl.nodeRef, v1.EventTypeWarning, events.KubeletSetupFailed, err.Error())
		klog.ErrorS(err, "Failed to initialize internal modules")
		os.Exit(1)
	}

	// 启动卷管理器，响应节点csi插件和in-tree文卷插件的请求，管理好节点上的持久化存储。
	go kl.volumeManager.Run(kl.sourcesReady, wait.NeverStop)

	if kl.kubeClient != nil {
		// 引入一些小的抖动，以确保由于优先级和空气性效应，随着时间的推移，请求不会从节点集几乎同时开始积累。
		go wait.JitterUntil(kl.syncNodeStatus, kl.nodeStatusUpdateFrequency, 0.04, true, wait.NeverStop)
		go kl.fastStatusUpdateOnce()

		// 开始同步节点node租约
		go kl.nodeLeaseController.Run(wait.NeverStop)
	}
	go wait.Until(kl.updateRuntimeUp, 5*time.Second, wait.NeverStop)

	// 设置iptables规则
	if kl.makeIPTablesUtilChains {
		kl.initNetworkUtil()
	}

	// 启动状态管理器
	kl.statusManager.Start()

	// 启动运行时管理器
	if kl.runtimeClassManager != nil {
		kl.runtimeClassManager.Start(wait.NeverStop)
	}

	// 通过 PLEG 进行 Pod 生命周期事件管理  
	kl.pleg.Start()
	kl.syncLoop(updates, kl)
}
```
### volumeManager
Volume 的创建和管理在 Kubernetes 中主要由卷管理器 VolumeManager 和 AttachDetachController 和 PVController 三个组件负责。其中卷管理器会负责卷的创建和管理的大部分工作，而 AttachDetachController 主要负责对集群中的卷进行 Attach 和 Detach，PVController 负责处理持久卷的变更。
```go
// kubernetes/pkg/kubelet/volumemanager/volume_manager.go
type volumeManager struct {
	// 用于与API Server通信以获取PV和PVC对象的API Client
	kubeClient clientset.Interface

	// 卷插件管理器，用于访问卷插件
	volumePluginMgr *volume.VolumePluginMgr

	// 正在被Pod引用的文卷在集群中被期望的状态。
	desiredStateOfWorld cache.DesiredStateOfWorld

	// 哪些卷被附加到这个节点上的文卷对应的实际状态。
	actualStateOfWorld cache.ActualStateOfWorld

	// 启动异步加载、卸载、挂载和卸载操作
	operationExecutor operationexecutor.OperationExecutor

    // 通过使用operationExecutor触发附加、分离、挂载和卸载操作来协调desiredStateOfWorld和actualStateOfWorld
	reconciler reconciler.Reconciler

	// 运行异步周期循环，使用kubelet PodManager保证desiredStateOfWorld中的文卷状态。
	desiredStateOfWorldPopulator populator.DesiredStateOfWorldPopulator

	//  跟踪插件的CSI迁移状态
	csiMigratedPluginManager csimigration.PluginManager

	// 将in-tree插件翻译为CSI插件
	intreeToCSITranslator csimigration.InTreeToCSITranslator
}
```
volumeManager的工作流程如下，后面准备也做一下Kubernetes Volume管理的博客，暂时只贴一下Run函数的代码不深入分析了。、
DesiredStateOfWorldPopulator 生产者讲将当前节点的期望状态同步到产品队列 DesiredStateOfWorld 中，等待消费者(reconciler)的处理启动。
```go
func (vm *volumeManager) Run(sourcesReady config.SourcesReady, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	if vm.kubeClient != nil {
		// 启动csi informer
		go vm.volumePluginMgr.Run(stopCh)
	}


	go vm.desiredStateOfWorldPopulator.Run(sourcesReady, stopCh)
	klog.V(2).InfoS("The desired_state_of_world populator starts")

	klog.InfoS("Starting Kubelet Volume Manager")
	go vm.reconciler.Run(stopCh)

    // 卷管理器metrics注册
	metrics.Register(vm.actualStateOfWorld, vm.desiredStateOfWorld, vm.volumePluginMgr)

	<-stopCh
	klog.InfoS("Shutting down Kubelet Volume Manager")
}

```

### PLEG(Pod Lifecycle Event Generator)
Pod生命周期管理相关接口如下：
```go
// kubernetes/pkg/kubelet/pleg/pleg.go  
type PodLifecycleEventGenerator interface {  
	Start()                         // 通过 relist 获取所有 Pods 并计算事件类型  
	Watch() chan *PodLifecycleEvent // 监听 eventChannel，传递给下游消费者  
	Healthy() (bool, error)  
}
```
PLEG的入口函数为GenericPLEG的成员函数Start
```go
// kubernetes/pkg/kubelet/pleg/generic.go  
// 启动一个 goroutine 周期性的重新生成 Pods 列表。
func (g *GenericPLEG) Start() {
	go wait.Until(g.relist, g.relistPeriod, wait.NeverStop)
}
```
relist函数在重新生成新的 Pods 列表的过程中，不管生产新的 Pod 事件：
```go
// kubernetes/pkg/kubelet/pleg/generic.go  
// 生产者：获取所有 Pods 列表，计算出对应的事件类型，进行 Sync  
func (g *GenericPLEG) relist() {  
    klog.V(5).InfoS("GenericPLEG: Relisting")  
    ...  
    // 获取当前所有 Pods 列表  
    podList, err := g.runtime.GetPods(true)  
    if err != nil {  
        klog.ErrorS(err, "GenericPLEG: Unable to retrieve pods")  
        return  
    }  

    // 遍历所有Pod和所有容器
    for pid := range g.podRecords {  
        allContainers := getContainersFromPods(oldPod, pod)
        for _, container := range allContainers {  
            // 计算事件类型：running/exited/unknown/non-existent  
            events := computeEvents(oldPod, pod, &container.ID)  
            for _, e := range events {  
                updateEvents(eventsByPodID, e)  
            }
        }
    }

    // 遍历所有事件  
    for pid, events := range eventsByPodID {  
        for i := range events {  
            // Filter out events that are not reliable and no other components use yet.  
            if events[i].Type == ContainerChanged {  
                continue  
            }  
            select {  
                case g.eventChannel <- events[i]: // 生产者：发送到事件 channel，对应监听的 goroutine 会消费  
                default:  
                metrics.PLEGDiscardEvents.Inc()  
                klog.ErrorS(nil, "Event channel is full, discard this relist() cycle event")  
            }
        }
    }
    ...
} 
```

#### Kubelet.syncLoop
syncLoop 是Pods生命周期管理的主循环。当监听到 Pod 事件时，进行对应 Pod 的创建或删除，流程如下：

syncLoop -> syncLoopIteration -> SyncPodCreate/Kill -> UpdatePod -> syncPod/syncTerminatingPod -> (containerRuntime service)syncPod -> (grpc)Pod running/teminated

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) syncLoop(updates <-chan kubetypes.PodUpdate, handler SyncHandler) {
	// 对于看到的任何新更改，将对期望状态和运行状态运行同步。如果没有看到配置的更改，将每隔一个同步频率秒同步最后一个已知的期望状态。

	klog.InfoS("Starting kubelet main sync loop")

	// syncTicker唤醒kubelet检查是否需要同步，同步间隔默认为10s。
	syncTicker := time.NewTicker(time.Second)
	defer syncTicker.Stop()
	housekeepingTicker := time.NewTicker(housekeepingPeriod)
	defer housekeepingTicker.Stop()
	plegCh := kl.pleg.Watch()
	const (
		base   = 100 * time.Millisecond
		max    = 5 * time.Second
		factor = 2
	)
	duration := base

	if kl.dnsConfigurer != nil && kl.dnsConfigurer.ResolverConfig != "" {
		kl.dnsConfigurer.CheckLimitsForResolvConf()
	}

	for {
		if err := kl.runtimeState.runtimeErrors(); err != nil {
			klog.ErrorS(err, "Skipping pod synchronization")
			// exponential backoff
			time.Sleep(duration)
			duration = time.Duration(math.Min(float64(max), factor*float64(duration)))
			continue
		}
		// reset backoff if we have a success
		duration = base

		kl.syncLoopMonitor.Store(kl.clock.Now())
		if !kl.syncLoopIteration(updates, handler, syncTicker.C, housekeepingTicker.C, plegCh) {
			break
		}
		kl.syncLoopMonitor.Store(kl.clock.Now())
	}
}
```
syncLoopIteration 每次从 channel 中取出一个事件，进行 Pod 同步 
```go
// kubernetes/pkg/kubelet/kubelet.go  
// The workflow is:
//   - If the pod is being created, record pod worker start latency
//   - Call generateAPIPodStatus to prepare an v1.PodStatus for the pod
//   - If the pod is being seen as running for the first time, record pod
//     start latency
//   - Update the status of the pod in the status manager
//   - Stop the pod's containers if it should not be running due to soft
//     admission
//   - Ensure any background tracking for a runnable pod is started
//   - Create a mirror pod if the pod is a static pod, and does not
//     already have a mirror pod
//   - Create the data directories for the pod if they do not exist
//   - Wait for volumes to attach/mount
//   - Fetch the pull secrets for the pod
//   - Call the container runtime's SyncPod callback
//   - Update the traffic shaping for the pod's ingress and egress limits
//
// If any step of this workflow errors, the error is returned, and is repeated
// on the next syncPod call.
//
// This operation writes all events that are dispatched in order to provide
// the most accurate information possible about an error situation to aid debugging.
// Callers should not write an event if this operation returns an error.
func (kl *Kubelet) syncPod(ctx context.Context, updateType kubetypes.SyncPodType, pod, mirrorPod *v1.Pod, podStatus *kubecontainer.PodStatus) (isTerminal bool, err error) {
	klog.V(4).InfoS("syncPod enter", "pod", klog.KObj(pod), "podUID", pod.UID)
	defer func() {
		klog.V(4).InfoS("syncPod exit", "pod", klog.KObj(pod), "podUID", pod.UID, "isTerminal", isTerminal)
	}()

	// Latency measurements for the main workflow are relative to the
	// first time the pod was seen by kubelet.
	var firstSeenTime time.Time
	if firstSeenTimeStr, ok := pod.Annotations[kubetypes.ConfigFirstSeenAnnotationKey]; ok {
		firstSeenTime = kubetypes.ConvertToTimestamp(firstSeenTimeStr).Get()
	}

	// Record pod worker start latency if being created
	// TODO: make pod workers record their own latencies
	if updateType == kubetypes.SyncPodCreate {
		if !firstSeenTime.IsZero() {
			// This is the first time we are syncing the pod. Record the latency
			// since kubelet first saw the pod if firstSeenTime is set.
			metrics.PodWorkerStartDuration.Observe(metrics.SinceInSeconds(firstSeenTime))
		} else {
			klog.V(3).InfoS("First seen time not recorded for pod",
				"podUID", pod.UID,
				"pod", klog.KObj(pod))
		}
	}

	// Generate final API pod status with pod and status manager status
	apiPodStatus := kl.generateAPIPodStatus(pod, podStatus)
	// The pod IP may be changed in generateAPIPodStatus if the pod is using host network. (See #24576)
	// TODO(random-liu): After writing pod spec into container labels, check whether pod is using host network, and
	// set pod IP to hostIP directly in runtime.GetPodStatus
	podStatus.IPs = make([]string, 0, len(apiPodStatus.PodIPs))
	for _, ipInfo := range apiPodStatus.PodIPs {
		podStatus.IPs = append(podStatus.IPs, ipInfo.IP)
	}
	if len(podStatus.IPs) == 0 && len(apiPodStatus.PodIP) > 0 {
		podStatus.IPs = []string{apiPodStatus.PodIP}
	}

	// If the pod is terminal, we don't need to continue to setup the pod
	if apiPodStatus.Phase == v1.PodSucceeded || apiPodStatus.Phase == v1.PodFailed {
		kl.statusManager.SetPodStatus(pod, apiPodStatus)
		isTerminal = true
		return isTerminal, nil
	}

	// If the pod should not be running, we request the pod's containers be stopped. This is not the same
	// as termination (we want to stop the pod, but potentially restart it later if soft admission allows
	// it later). Set the status and phase appropriately
	runnable := kl.canRunPod(pod)
	if !runnable.Admit {
		// Pod is not runnable; and update the Pod and Container statuses to why.
		if apiPodStatus.Phase != v1.PodFailed && apiPodStatus.Phase != v1.PodSucceeded {
			apiPodStatus.Phase = v1.PodPending
		}
		apiPodStatus.Reason = runnable.Reason
		apiPodStatus.Message = runnable.Message
		// Waiting containers are not creating.
		const waitingReason = "Blocked"
		for _, cs := range apiPodStatus.InitContainerStatuses {
			if cs.State.Waiting != nil {
				cs.State.Waiting.Reason = waitingReason
			}
		}
		for _, cs := range apiPodStatus.ContainerStatuses {
			if cs.State.Waiting != nil {
				cs.State.Waiting.Reason = waitingReason
			}
		}
	}

	// Record the time it takes for the pod to become running
	// since kubelet first saw the pod if firstSeenTime is set.
	existingStatus, ok := kl.statusManager.GetPodStatus(pod.UID)
	if !ok || existingStatus.Phase == v1.PodPending && apiPodStatus.Phase == v1.PodRunning &&
		!firstSeenTime.IsZero() {
		metrics.PodStartDuration.Observe(metrics.SinceInSeconds(firstSeenTime))
	}

	kl.statusManager.SetPodStatus(pod, apiPodStatus)

	// Pods that are not runnable must be stopped - return a typed error to the pod worker
	if !runnable.Admit {
		klog.V(2).InfoS("Pod is not runnable and must have running containers stopped", "pod", klog.KObj(pod), "podUID", pod.UID, "message", runnable.Message)
		var syncErr error
		p := kubecontainer.ConvertPodStatusToRunningPod(kl.getRuntime().Type(), podStatus)
		if err := kl.killPod(pod, p, nil); err != nil {
			kl.recorder.Eventf(pod, v1.EventTypeWarning, events.FailedToKillPod, "error killing pod: %v", err)
			syncErr = fmt.Errorf("error killing pod: %v", err)
			utilruntime.HandleError(syncErr)
		} else {
			// There was no error killing the pod, but the pod cannot be run.
			// Return an error to signal that the sync loop should back off.
			syncErr = fmt.Errorf("pod cannot be run: %s", runnable.Message)
		}
		return false, syncErr
	}

	// If the network plugin is not ready, only start the pod if it uses the host network
	if err := kl.runtimeState.networkErrors(); err != nil && !kubecontainer.IsHostNetworkPod(pod) {
		kl.recorder.Eventf(pod, v1.EventTypeWarning, events.NetworkNotReady, "%s: %v", NetworkNotReadyErrorMsg, err)
		return false, fmt.Errorf("%s: %v", NetworkNotReadyErrorMsg, err)
	}

	// ensure the kubelet knows about referenced secrets or configmaps used by the pod
	if !kl.podWorkers.IsPodTerminationRequested(pod.UID) {
		if kl.secretManager != nil {
			kl.secretManager.RegisterPod(pod)
		}
		if kl.configMapManager != nil {
			kl.configMapManager.RegisterPod(pod)
		}
	}

	// Create Cgroups for the pod and apply resource parameters
	// to them if cgroups-per-qos flag is enabled.
	pcm := kl.containerManager.NewPodContainerManager()
	// If pod has already been terminated then we need not create
	// or update the pod's cgroup
	// TODO: once context cancellation is added this check can be removed
	if !kl.podWorkers.IsPodTerminationRequested(pod.UID) {
		// When the kubelet is restarted with the cgroups-per-qos
		// flag enabled, all the pod's running containers
		// should be killed intermittently and brought back up
		// under the qos cgroup hierarchy.
		// Check if this is the pod's first sync
		firstSync := true
		for _, containerStatus := range apiPodStatus.ContainerStatuses {
			if containerStatus.State.Running != nil {
				firstSync = false
				break
			}
		}
		// Don't kill containers in pod if pod's cgroups already
		// exists or the pod is running for the first time
		podKilled := false
		if !pcm.Exists(pod) && !firstSync {
			p := kubecontainer.ConvertPodStatusToRunningPod(kl.getRuntime().Type(), podStatus)
			if err := kl.killPod(pod, p, nil); err == nil {
				podKilled = true
			} else {
				klog.ErrorS(err, "KillPod failed", "pod", klog.KObj(pod), "podStatus", podStatus)
			}
		}
		// Create and Update pod's Cgroups
		// Don't create cgroups for run once pod if it was killed above
		// The current policy is not to restart the run once pods when
		// the kubelet is restarted with the new flag as run once pods are
		// expected to run only once and if the kubelet is restarted then
		// they are not expected to run again.
		// We don't create and apply updates to cgroup if its a run once pod and was killed above
		if !(podKilled && pod.Spec.RestartPolicy == v1.RestartPolicyNever) {
			if !pcm.Exists(pod) {
				if err := kl.containerManager.UpdateQOSCgroups(); err != nil {
					klog.V(2).InfoS("Failed to update QoS cgroups while syncing pod", "pod", klog.KObj(pod), "err", err)
				}
				if err := pcm.EnsureExists(pod); err != nil {
					kl.recorder.Eventf(pod, v1.EventTypeWarning, events.FailedToCreatePodContainer, "unable to ensure pod container exists: %v", err)
					return false, fmt.Errorf("failed to ensure that the pod: %v cgroups exist and are correctly applied: %v", pod.UID, err)
				}
			}
		}
	}

	// Create Mirror Pod for Static Pod if it doesn't already exist
	if kubetypes.IsStaticPod(pod) {
		deleted := false
		if mirrorPod != nil {
			if mirrorPod.DeletionTimestamp != nil || !kl.podManager.IsMirrorPodOf(mirrorPod, pod) {
				// The mirror pod is semantically different from the static pod. Remove
				// it. The mirror pod will get recreated later.
				klog.InfoS("Trying to delete pod", "pod", klog.KObj(pod), "podUID", mirrorPod.ObjectMeta.UID)
				podFullName := kubecontainer.GetPodFullName(pod)
				var err error
				deleted, err = kl.podManager.DeleteMirrorPod(podFullName, &mirrorPod.ObjectMeta.UID)
				if deleted {
					klog.InfoS("Deleted mirror pod because it is outdated", "pod", klog.KObj(mirrorPod))
				} else if err != nil {
					klog.ErrorS(err, "Failed deleting mirror pod", "pod", klog.KObj(mirrorPod))
				}
			}
		}
		if mirrorPod == nil || deleted {
			node, err := kl.GetNode()
			if err != nil || node.DeletionTimestamp != nil {
				klog.V(4).InfoS("No need to create a mirror pod, since node has been removed from the cluster", "node", klog.KRef("", string(kl.nodeName)))
			} else {
				klog.V(4).InfoS("Creating a mirror pod for static pod", "pod", klog.KObj(pod))
				if err := kl.podManager.CreateMirrorPod(pod); err != nil {
					klog.ErrorS(err, "Failed creating a mirror pod for", "pod", klog.KObj(pod))
				}
			}
		}
	}

	// Make data directories for the pod
	if err := kl.makePodDataDirs(pod); err != nil {
		kl.recorder.Eventf(pod, v1.EventTypeWarning, events.FailedToMakePodDataDirectories, "error making pod data directories: %v", err)
		klog.ErrorS(err, "Unable to make pod data directories for pod", "pod", klog.KObj(pod))
		return false, err
	}

	// Volume manager will not mount volumes for terminating pods
	// TODO: once context cancellation is added this check can be removed
	if !kl.podWorkers.IsPodTerminationRequested(pod.UID) {
		// Wait for volumes to attach/mount
		if err := kl.volumeManager.WaitForAttachAndMount(pod); err != nil {
			kl.recorder.Eventf(pod, v1.EventTypeWarning, events.FailedMountVolume, "Unable to attach or mount volumes: %v", err)
			klog.ErrorS(err, "Unable to attach or mount volumes for pod; skipping pod", "pod", klog.KObj(pod))
			return false, err
		}
	}

	// Fetch the pull secrets for the pod
	pullSecrets := kl.getPullSecretsForPod(pod)

	// Ensure the pod is being probed
	kl.probeManager.AddPod(pod)

	// Call the container runtime's SyncPod callback
	result := kl.containerRuntime.SyncPod(pod, podStatus, pullSecrets, kl.backOff)
	kl.reasonCache.Update(pod.UID, result)
	if err := result.Error(); err != nil {
		// Do not return error if the only failures were pods in backoff
		for _, r := range result.SyncResults {
			if r.Error != kubecontainer.ErrCrashLoopBackOff && r.Error != images.ErrImagePullBackOff {
				// Do not record an event here, as we keep all event logging for sync pod failures
				// local to container runtime, so we get better errors.
				return false, err
			}
		}

		return false, nil
	}

	return false, nil
}
```
经过上一步 Pod 事件的生产与消费传递，PodWorkers 会将事件转化为 gRPC client 请求，然后调用 dockershim gRPC server，进行 PodSandbox、infra-container（也叫 pause 容器）的创建。

接着，会调用 CNI 接口 SetUpPod 进行相关网络配置与启动，此时建立起来的容器网络，就可以直接用于之后创建的业务容器如 initContainers、containers 进行共享网络。