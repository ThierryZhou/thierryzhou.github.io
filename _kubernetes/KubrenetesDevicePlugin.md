---
title: "Kubrenetes 设备插件详解"
tag: kubernetes
---

## 简介
Kubernetes 提供了一个 设备插件框架， 你可以用它来将系统硬件资源发布到 Kubelet。

供应商可以实现设备插件，由你手动部署或作为 DaemonSet 来部署，而不必定制 Kubernetes 本身的代码。目标设备包括 GPU、高性能 NIC、FPGA、 InfiniBand 适配器以及其他类似的、可能需要特定于供应商的初始化和设置的计算资源。

注册设备插件
kubelet 提供了一个 Registration 的 gRPC 服务：
```shell
service Registration {
	rpc Register(RegisterRequest) returns (Empty) {}
}
```
设备插件可以通过此 gRPC 服务在 kubelet 进行注册。在注册期间，设备插件需要发送下面几样内容：

设备插件的 Unix 套接字。
设备插件的 API 版本。
ResourceName 是需要公布的。这里 ResourceName 需要遵循扩展资源命名方案， 类似于 vendor-domain/resourcetype。（比如 NVIDIA GPU 就被公布为 nvidia.com/gpu。）
成功注册后，设备插件就向 kubelet 发送它所管理的设备列表，然后 kubelet 负责将这些资源发布到 API 服务器，作为 kubelet 节点状态更新的一部分。

比如，设备插件在 kubelet 中注册了 hardware-vendor.example/foo 并报告了节点上的两个运行状况良好的设备后，节点状态将更新以通告该节点已安装 2 个 "Foo" 设备并且是可用的。

然后，用户可以请求设备作为 Pod 规范的一部分， 参见 Container。 请求扩展资源类似于管理请求和限制的方式， 其他资源，有以下区别：

扩展资源仅可作为整数资源使用，并且不能被过量使用
设备不能在容器之间共享
示例
假设 Kubernetes 集群正在运行一个设备插件，该插件在一些节点上公布的资源为 hardware-vendor.example/foo。 下面就是一个 Pod 示例，请求此资源以运行一个工作负载的示例：

---
apiVersion: v1
kind: Pod
metadata:
  name: demo-pod
spec:
  containers:
    - name: demo-container-1
      image: registry.k8s.io/pause:2.0
      resources:
        limits:
          hardware-vendor.example/foo: 2

# 这个 pod 需要两个 hardware-vendor.example/foo 设备
# 而且只能够调度到满足需求的节点上
#
# 如果该节点中有 2 个以上的设备可用，其余的可供其他 Pod 使用
设备插件的实现
设备插件的常规工作流程包括以下几个步骤：

初始化。在这个阶段，设备插件将执行供应商特定的初始化和设置， 以确保设备处于就绪状态。

插件使用主机路径 /var/lib/kubelet/device-plugins/ 下的 Unix 套接字启动一个 gRPC 服务，该服务实现以下接口：
```shell
service DevicePlugin {
    // GetDevicePluginOptions 返回与设备管理器沟通的选项。
    rpc GetDevicePluginOptions(Empty) returns (DevicePluginOptions) {}

    // ListAndWatch 返回 Device 列表构成的数据流。
    // 当 Device 状态发生变化或者 Device 消失时，ListAndWatch
    // 会返回新的列表。
    rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}

    // Allocate 在容器创建期间调用，这样设备插件可以运行一些特定于设备的操作，
    // 并告诉 kubelet 如何令 Device 可在容器中访问的所需执行的具体步骤
    rpc Allocate(AllocateRequest) returns (AllocateResponse) {}

    // GetPreferredAllocation 从一组可用的设备中返回一些优选的设备用来分配，
    // 所返回的优选分配结果不一定会是设备管理器的最终分配方案。
    // 此接口的设计仅是为了让设备管理器能够在可能的情况下做出更有意义的决定。
    rpc GetPreferredAllocation(PreferredAllocationRequest) returns (PreferredAllocationResponse) {}

    // PreStartContainer 在设备插件注册阶段根据需要被调用，调用发生在容器启动之前。
    // 在将设备提供给容器使用之前，设备插件可以运行一些诸如重置设备之类的特定于
    // 具体设备的操作，
    rpc PreStartContainer(PreStartContainerRequest) returns (PreStartContainerResponse) {}
}
```
说明：
插件并非必须为 GetPreferredAllocation() 或 PreStartContainer() 提供有用的实现逻辑， 调用 GetDevicePluginOptions() 时所返回的 DevicePluginOptions 消息中应该设置这些调用是否可用。kubelet 在真正调用这些函数之前，总会调用 GetDevicePluginOptions() 来查看是否存在这些可选的函数。

插件通过 Unix socket 在主机路径 /var/lib/kubelet/device-plugins/kubelet.sock 处向 kubelet 注册自身。
成功注册自身后，设备插件将以服务模式运行，在此期间，它将持续监控设备运行状况， 并在设备状态发生任何变化时向 kubelet 报告。它还负责响应 Allocate gRPC 请求。 在 Allocate 期间，设备插件可能还会做一些设备特定的准备；例如 GPU 清理或 QRNG 初始化。 如果操作成功，则设备插件将返回 AllocateResponse，其中包含用于访问被分配的设备容器运行时的配置。 kubelet 将此信息传递到容器运行时。
处理 kubelet 重启
设备插件应能监测到 kubelet 重启，并且向新的 kubelet 实例来重新注册自己。 在当前实现中，当 kubelet 重启的时候，新的 kubelet 实例会删除 /var/lib/kubelet/device-plugins 下所有已经存在的 Unix 套接字。 设备插件需要能够监控到它的 Unix 套接字被删除，并且当发生此类事件时重新注册自己。

设备插件部署
你可以将你的设备插件作为节点操作系统的软件包来部署、作为 DaemonSet 来部署或者手动部署。

规范目录 /var/lib/kubelet/device-plugins 是需要特权访问的， 所以设备插件必须要在被授权的安全的上下文中运行。 如果你将设备插件部署为 DaemonSet，/var/lib/kubelet/device-plugins 目录必须要在插件的 PodSpec 中声明作为 卷（Volume） 被挂载到插件中。

如果你选择 DaemonSet 方法，你可以通过 Kubernetes 进行以下操作： 将设备插件的 Pod 放置在节点上，在出现故障后重新启动守护进程 Pod，来进行自动升级。

API 兼容性
Kubernetes 设备插件支持还处于 beta 版本。所以在稳定版本出来之前 API 会以不兼容的方式进行更改。 作为一个项目，Kubernetes 建议设备插件开发者：

注意未来版本的更改
支持多个版本的设备插件 API，以实现向后/向前兼容性。
如果你启用 DevicePlugins 功能，并在需要升级到 Kubernetes 版本来获得较新的设备插件 API 版本的节点上运行设备插件，请在升级这些节点之前先升级设备插件以支持这两个版本。 采用该方法将确保升级期间设备分配的连续运行。

监控设备插件资源
特性状态： Kubernetes v1.15 [beta]
为了监控设备插件提供的资源，监控代理程序需要能够发现节点上正在使用的设备， 并获取元数据来描述哪个指标与容器相关联。 设备监控代理暴露给 Prometheus 的指标应该遵循 Kubernetes Instrumentation Guidelines， 使用 pod、namespace 和 container 标签来标识容器。

kubelet 提供了 gRPC 服务来使得正在使用中的设备被发现，并且还为这些设备提供了元数据：
```shell
// PodResourcesLister 是一个由 kubelet 提供的服务，用来提供供节点上 
// Pod 和容器使用的节点资源的信息
service PodResourcesLister {
    rpc List(ListPodResourcesRequest) returns (ListPodResourcesResponse) {}
    rpc GetAllocatableResources(AllocatableResourcesRequest) returns (AllocatableResourcesResponse) {}
}
List gRPC 端点
这一 List 端点提供运行中 Pod 的资源信息，包括类似独占式分配的 CPU ID、设备插件所报告的设备 ID 以及这些设备分配所处的 NUMA 节点 ID。 此外，对于基于 NUMA 的机器，它还会包含为容器保留的内存和大页的信息。

// ListPodResourcesResponse 是 List 函数的响应
message ListPodResourcesResponse {
    repeated PodResources pod_resources = 1;
}

// PodResources 包含关于分配给 Pod 的节点资源的信息
message PodResources {
    string name = 1;
    string namespace = 2;
    repeated ContainerResources containers = 3;
}

// ContainerResources 包含分配给容器的资源的信息
message ContainerResources {
    string name = 1;
    repeated ContainerDevices devices = 2;
    repeated int64 cpu_ids = 3;
    repeated ContainerMemory memory = 4;
}

// ContainerMemory 包含分配给容器的内存和大页信息
message ContainerMemory {
    string memory_type = 1;
    uint64 size = 2;
    TopologyInfo topology = 3;
}

// Topology 描述资源的硬件拓扑结构
message TopologyInfo {
        repeated NUMANode nodes = 1;
}

// NUMA 代表的是 NUMA 节点
message NUMANode {
        int64 ID = 1;
}

// ContainerDevices 包含分配给容器的设备信息
message ContainerDevices {
    string resource_name = 1;
    repeated string device_ids = 2;
    TopologyInfo topology = 3;
}
```
说明：
List 端点中的 ContainerResources 中的 cpu_ids 对应于分配给某个容器的专属 CPU。 如果要统计共享池中的 CPU，List 端点需要与 GetAllocatableResources 端点一起使用，如下所述:

调用 GetAllocatableResources 获取所有可用的 CPU。
在系统中所有的 ContainerResources 上调用 GetCpuIds。
用 GetAllocatableResources 获取的 CPU 数减去 GetCpuIds 获取的 CPU 数。
GetAllocatableResources gRPC 端点
特性状态： Kubernetes v1.23 [beta]
端点 GetAllocatableResources 提供工作节点上原始可用的资源信息。 此端点所提供的信息比导出给 API 服务器的信息更丰富。

说明：
GetAllocatableResources 应该仅被用于评估一个节点上的可分配的资源。 如果目标是评估空闲/未分配的资源，此调用应该与 List() 端点一起使用。 除非暴露给 kubelet 的底层资源发生变化，否则 GetAllocatableResources 得到的结果将保持不变。 这种情况很少发生，但当发生时（例如：热插拔，设备健康状况改变），客户端应该调用 GetAlloctableResources 端点。 然而，调用 GetAllocatableResources 端点在 cpu、内存被更新的情况下是不够的， Kubelet 需要重新启动以获取正确的资源容量和可分配的资源。
```shell
// AllocatableResourcesResponses 包含 kubelet 所了解到的所有设备的信息
message AllocatableResourcesResponse {
    repeated ContainerDevices devices = 1;
    repeated int64 cpu_ids = 2;
    repeated ContainerMemory memory = 3;
}
```
从 Kubernetes v1.23 开始，GetAllocatableResources 被默认启用。 你可以通过关闭 KubeletPodResourcesGetAllocatable 特性门控来禁用。

在 Kubernetes v1.23 之前，要启用这一功能，kubelet 必须用以下标志启动：
```shell
--feature-gates=KubeletPodResourcesGetAllocatable=true
```
ContainerDevices 会向外提供各个设备所隶属的 NUMA 单元这类拓扑信息。 NUMA 单元通过一个整数 ID 来标识，其取值与设备插件所报告的一致。 设备插件注册到 kubelet 时 会报告这类信息。

gRPC 服务通过 /var/lib/kubelet/pod-resources/kubelet.sock 的 UNIX 套接字来提供服务。 设备插件资源的监控代理程序可以部署为守护进程或者 DaemonSet。 规范的路径 /var/lib/kubelet/pod-resources 需要特权来进入， 所以监控代理程序必须要在获得授权的安全的上下文中运行。 如果设备监控代理以 DaemonSet 形式运行，必须要在插件的 PodSpec 中声明将 /var/lib/kubelet/pod-resources 目录以卷的形式被挂载到设备监控代理中。

对 “PodResourcesLister 服务”的支持要求启用 KubeletPodResources 特性门控。 从 Kubernetes 1.15 开始默认启用，自从 Kubernetes 1.20 开始为 v1。

设备插件与拓扑管理器的集成
特性状态： Kubernetes v1.18 [beta]
拓扑管理器是 Kubelet 的一个组件，它允许以拓扑对齐方式来调度资源。 为了做到这一点，设备插件 API 进行了扩展来包括一个 TopologyInfo 结构体。
```shell
message TopologyInfo {
    repeated NUMANode nodes = 1;
}

message NUMANode {
    int64 ID = 1;
}
```
设备插件希望拓扑管理器可以将填充的 TopologyInfo 结构体作为设备注册的一部分以及设备 ID 和设备的运行状况发送回去。然后设备管理器将使用此信息来咨询拓扑管理器并做出资源分配决策。

TopologyInfo 支持将 nodes 字段设置为 nil 或一个 NUMA 节点的列表。 这样就可以使设备插件通告跨越多个 NUMA 节点的设备。

将 TopologyInfo 设置为 nil 或为给定设备提供一个空的 NUMA 节点列表表示设备插件没有该设备的 NUMA 亲和偏好。

下面是一个由设备插件为设备填充 TopologyInfo 结构体的示例：