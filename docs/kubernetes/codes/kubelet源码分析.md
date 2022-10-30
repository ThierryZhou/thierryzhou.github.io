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
func (m *kubeGenericRuntimeManager) SyncPod(pod *v1.Pod, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, backOff *flowcontrol.Backoff) (result kubecontainer.PodSyncResult) {
	// 1: 计算 sandbox and container 的改变。
	podContainerChanges := m.computePodActions(pod, podStatus)
	klog.V(3).InfoS("computePodActions got for pod", "podActions", podContainerChanges, "pod", klog.KObj(pod))
	if podContainerChanges.CreateSandbox {
		ref, err := ref.GetReference(legacyscheme.Scheme, pod)
		if err != nil {
			klog.ErrorS(err, "Couldn't make a ref to pod", "pod", klog.KObj(pod))
		}
		if podContainerChanges.SandboxID != "" {
			m.recorder.Eventf(ref, v1.EventTypeNormal, events.SandboxChanged, "Pod sandbox changed, it will be killed and re-created.")
		} else {
			klog.V(4).InfoS("SyncPod received new pod, will create a sandbox for it", "pod", klog.KObj(pod))
		}
	}

	// 2: 如果 sandbox 已经改变，则杀死 Pod。
	if podContainerChanges.KillPod {
		if podContainerChanges.CreateSandbox {
			klog.V(4).InfoS("Stopping PodSandbox for pod, will start new one", "pod", klog.KObj(pod))
		} else {
			klog.V(4).InfoS("Stopping PodSandbox for pod, because all other containers are dead", "pod", klog.KObj(pod))
		}

		killResult := m.killPodWithSyncResult(pod, kubecontainer.ConvertPodStatusToRunningPod(m.runtimeName, podStatus), nil)
		result.AddPodSyncResult(killResult)
		if killResult.Error() != nil {
			klog.ErrorS(killResult.Error(), "killPodWithSyncResult failed")
			return
		}

		if podContainerChanges.CreateSandbox {
			m.purgeInitContainers(pod, podStatus)
		}
	} else {
		// 3: 杀死所有不必要保留的容器。
		for containerID, containerInfo := range podContainerChanges.ContainersToKill {
			klog.V(3).InfoS("Killing unwanted container for pod", "containerName", containerInfo.name, "containerID", containerID, "pod", klog.KObj(pod))
			killContainerResult := kubecontainer.NewSyncResult(kubecontainer.KillContainer, containerInfo.name)
			result.AddSyncResult(killContainerResult)
			if err := m.killContainer(pod, containerID, containerInfo.name, containerInfo.message, containerInfo.reason, nil); err != nil {
				killContainerResult.Fail(kubecontainer.ErrKillContainer, err.Error())
				klog.ErrorS(err, "killContainer for pod failed", "containerName", containerInfo.name, "containerID", containerID, "pod", klog.KObj(pod))
				return
			}
		}
	}

	// 终止所有 init containers
	m.pruneInitContainersBeforeStart(pod, podStatus)

	var podIPs []string
	if podStatus != nil {
		podIPs = podStatus.IPs
	}

	// 4: 为 Pod 创建一个sandbox
	podSandboxID := podContainerChanges.SandboxID
	if podContainerChanges.CreateSandbox {
		var msg string
		var err error

		klog.V(4).InfoS("Creating PodSandbox for pod", "pod", klog.KObj(pod))
		metrics.StartedPodsTotal.Inc()
		createSandboxResult := kubecontainer.NewSyncResult(kubecontainer.CreatePodSandbox, format.Pod(pod))
		result.AddSyncResult(createSandboxResult)

		sysctl.ConvertPodSysctlsVariableToDotsSeparator(pod.Spec.SecurityContext)

		podSandboxID, msg, err = m.createPodSandbox(pod, podContainerChanges.Attempt)
		if err != nil {

			if m.podStateProvider.IsPodTerminationRequested(pod.UID) {
				klog.V(4).InfoS("Pod was deleted and sandbox failed to be created", "pod", klog.KObj(pod), "podUID", pod.UID)
				return
			}
			metrics.StartedPodsErrorsTotal.Inc()
			createSandboxResult.Fail(kubecontainer.ErrCreatePodSandbox, msg)
			klog.ErrorS(err, "CreatePodSandbox for pod failed", "pod", klog.KObj(pod))
			ref, referr := ref.GetReference(legacyscheme.Scheme, pod)
			if referr != nil {
				klog.ErrorS(referr, "Couldn't make a ref to pod", "pod", klog.KObj(pod))
			}
			m.recorder.Eventf(ref, v1.EventTypeWarning, events.FailedCreatePodSandBox, "Failed to create pod sandbox: %v", err)
			return
		}
		klog.V(4).InfoS("Created PodSandbox for pod", "podSandboxID", podSandboxID, "pod", klog.KObj(pod))

		resp, err := m.runtimeService.PodSandboxStatus(podSandboxID, false)
		if err != nil {
			ref, referr := ref.GetReference(legacyscheme.Scheme, pod)
			if referr != nil {
				klog.ErrorS(referr, "Couldn't make a ref to pod", "pod", klog.KObj(pod))
			}
			m.recorder.Eventf(ref, v1.EventTypeWarning, events.FailedStatusPodSandBox, "Unable to get pod sandbox status: %v", err)
			klog.ErrorS(err, "Failed to get pod sandbox status; Skipping pod", "pod", klog.KObj(pod))
			result.Fail(err)
			return
		}
		if resp.GetStatus() == nil {
			result.Fail(errors.New("pod sandbox status is nil"))
			return
		}

		if !kubecontainer.IsHostNetworkPod(pod) {
			podIPs = m.determinePodSandboxIPs(pod.Namespace, pod.Name, resp.GetStatus())
			klog.V(4).InfoS("Determined the ip for pod after sandbox changed", "IPs", podIPs, "pod", klog.KObj(pod))
		}
	}

	podIP := ""
	if len(podIPs) != 0 {
		podIP = podIPs[0]
	}

	configPodSandboxResult := kubecontainer.NewSyncResult(kubecontainer.ConfigPodSandbox, podSandboxID)
	result.AddSyncResult(configPodSandboxResult)
	podSandboxConfig, err := m.generatePodSandboxConfig(pod, podContainerChanges.Attempt)
	if err != nil {
		message := fmt.Sprintf("GeneratePodSandboxConfig for pod %q failed: %v", format.Pod(pod), err)
		klog.ErrorS(err, "GeneratePodSandboxConfig for pod failed", "pod", klog.KObj(pod))
		configPodSandboxResult.Fail(kubecontainer.ErrConfigPodSandbox, message)
		return
	}

	start := func(typeName, metricLabel string, spec *startSpec) error {
		startContainerResult := kubecontainer.NewSyncResult(kubecontainer.StartContainer, spec.container.Name)
		result.AddSyncResult(startContainerResult)

		isInBackOff, msg, err := m.doBackOff(pod, spec.container, podStatus, backOff)
		if isInBackOff {
			startContainerResult.Fail(err, msg)
			klog.V(4).InfoS("Backing Off restarting container in pod", "containerType", typeName, "container", spec.container, "pod", klog.KObj(pod))
			return err
		}

		metrics.StartedContainersTotal.WithLabelValues(metricLabel).Inc()
		if sc.HasWindowsHostProcessRequest(pod, spec.container) {
			metrics.StartedHostProcessContainersTotal.WithLabelValues(metricLabel).Inc()
		}
		klog.V(4).InfoS("Creating container in pod", "containerType", typeName, "container", spec.container, "pod", klog.KObj(pod))

		if msg, err := m.startContainer(podSandboxID, podSandboxConfig, spec, pod, podStatus, pullSecrets, podIP, podIPs); err != nil {
			metrics.StartedContainersErrorsTotal.WithLabelValues(metricLabel, err.Error()).Inc()
			if sc.HasWindowsHostProcessRequest(pod, spec.container) {
				metrics.StartedHostProcessContainersErrorsTotal.WithLabelValues(metricLabel, err.Error()).Inc()
			}
			startContainerResult.Fail(err, msg)
			switch {
			case err == images.ErrImagePullBackOff:
				klog.V(3).InfoS("Container start failed in pod", "containerType", typeName, "container", spec.container, "pod", klog.KObj(pod), "containerMessage", msg, "err", err)
			default:
				utilruntime.HandleError(fmt.Errorf("%v %+v start failed in pod %v: %v: %s", typeName, spec.container, format.Pod(pod), err, msg))
			}
			return err
		}

		return nil
	}

	// 5: 启动临时容器
	for _, idx := range podContainerChanges.EphemeralContainersToStart {
		start("ephemeral container", metrics.EphemeralContainer, ephemeralContainerStartSpec(&pod.Spec.EphemeralContainers[idx]))
	}

	// 6: 启动 init container
	if container := podContainerChanges.NextInitContainerToStart; container != nil {
		// 启动下一个 init container
		if err := start("init container", metrics.InitContainer, containerStartSpec(container)); err != nil {
			return
		}

		klog.V(4).InfoS("Completed init container for pod", "containerName", container.Name, "pod", klog.KObj(pod))
	}

	// 7: 启动容器
	for _, idx := range podContainerChanges.ContainersToStart {
		start("container", metrics.Container, containerStartSpec(&pod.Spec.Containers[idx]))
	}

	return
}
```
Pod创建完成后，后续还有CNI CRI 等相关的相关的工作会被传入容器运行时所对应的容器服务中进行，后面的代码暂时不趴了。