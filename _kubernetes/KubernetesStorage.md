---
title: "Kubernetes 存储详解"
tag: kubernetes
---
## 概述
由于容器默认情况下rootfs系统树没有一个与之对应的存储设备，因此可以认为容器中任何文件操作都是临时性的，这样的设计带来了两个问题，其一是如果容器因为某些原因被kubelet重启后，会丢失这些文件；其二是不同的容器之间无法共享文件。为了解决上述问题，Kubernetes 设计了卷（Volume） 这一抽象来管理容器中所有需要使用到的外部文件。

并且由于容器本身存在声明周期以及容器的存储来源有多样性的特点，卷本身带有独立的状态标识实现其生命周期循环，根绝业务场景的不同，卷又细分为持久卷、临时卷、投射卷三大类。

为了让集群管理员可以管理更多不同特性的持久卷，Kubernetes 又设计了存储类（StorageClass) 来管理每一类具有相同特性的持久卷，在后续的 Kubernetes 版本迭代过程中陆续加入了一些其他的特性，例如：为了可以将任意第三方存储暴露给容器，增加了容器存储接口 (CSI)；为了增加存储的可靠性，增加类似传统文件系统中快照概念的卷快照 (Snapshot) 资源；为了可以对接任意第三方对象存储，增加了容器对象存储接口(COSI)。

### 持久卷(Persistent Volume)
PersistentVolume 子系统为用户和管理员提供了一组 API，通过引入了两个新的 API 资源：PersistentVolume 存储提供者进行管理； PersistentVolumeClaim 被存储使用者引用。

持久卷（PersistentVolume，PV） 是集群中的一块存储，可以由管理员事先制备 (Provision) ， 或者使用存储类（Storage Class）来动态制备 (Provision) 。 持久卷是集群资源，就像节点也是集群资源一样。PV 持久卷和普通的 Volume 一样， 也是使用卷插件来实现的，只是它们拥有独立于任何使用 PV 的 Pod 的生命周期。
```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: foo-pv
spec:
  storageClassName: ""
  claimRef:
    name: foo-pvc
    namespace: foo
```

持久卷申领（PersistentVolumeClaim，PVC） 表达的是用户对存储的请求。概念上与 Pod 类似。 Pod 会耗用节点资源，而 PVC 申领会耗用 PV 资源。Pod 可以请求特定数量的资源（CPU 和内存）；同样 PVC 申领也可以请求特定的大小和访问模式 （例如，可以要求 PV 卷能够以 ReadWriteOnce、ReadOnlyMany 或 ReadWriteMany 模式之一来挂载，参见访问模式）。
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: foo-pvc
  namespace: foo
spec:
  storageClassName: ""
  volumeName: foo-pv
```

尽管 PersistentVolumeClaim 允许用户消耗抽象的存储资源， 常见的情况是针对不同的问题用户需要的是具有不同属性（如，性能）的 PersistentVolume 卷。 集群管理员需要能够提供不同性质的 PersistentVolume， 并且这些 PV 卷之间的差别不仅限于卷大小和访问模式，同时又不能将卷是如何实现的这些细节暴露给用户。 为了满足这类需求，就有了存储类（StorageClass） 资源。

#### 存储卷生命周期管理

##### 1. 制备 (Provision)

制备(Provision)一般是只在 Kubernetes API Server中的 PV 资源以及在存储系统中申请存储空间。PV 卷的制备有两种方式：静态制备或动态制备。

静态制备时，集群管理员创建若干 PV 卷，这些卷对象带有真实存储的细节信息。

动态制备时，集群管理员在Kubernetes上创建一个存储类，当集群发现没有PV可以与PVC匹配时，由存储类来管理 PV 资源的创建和存储空间的创建。

为了基于存储类完成动态的存储制备，集群管理员需要在 API 服务器上启用 DefaultStorageClass 准入控制器。

##### 2. 绑定 (Bind)

前面我们讲过 Pod 中每声明一个PVC 资源，则消费一个与之对应的 PV 资源，这一过程也被我们成为绑定 (Bind) 。集群控制平面监测到新的 PVC 对象，并寻找一个与之匹配的 PV 卷， 并将二者绑定到一起（将 PV 信息写入 PVC 对象的 ClaimRef 信息中，并将 PV 状态修改 Bound ）。

如果找不到匹配的 PV 卷，PVC 会无限期地处于未绑定状态。例如，即使某集群上制备了很多 50 Gi 大小的 PV 卷，也无法与请求 100 Gi 大小的存储的 PVC 匹配。当新的 100 Gi PV 卷被加入到集群时， 该 PVC 才有可能被绑定。

##### 3. 使用 (Using)

Pod 将 PVC 申领当做存储卷来使用。集群会检视 PVC 申领，找到所绑定的卷， 并为 Pod 挂载该卷。对于支持多种访问模式的卷， 用户要在 Pod 中以卷的形式使用申领时指定期望的访问模式。

一旦用户有了申领对象并且该申领已经被绑定， 则所绑定的 PV 卷在用户仍然需要它期间一直属于该用户。 用户通过在 Pod 的 volumes 块中包含 persistentVolumeClaim 节区来调度 Pod，访问所申领的 PV 卷。 相关细节可参阅使用申领作为卷。
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: volume-test
spec:
  containers:
  - name: container-test
    image: busybox:1.28
    volumeMounts:
    - name: foo
      mountPath: "/mnt/foo"
  volumes:
  - name: foo
    persistentVolumeClaim:
        claimName: foo-pvc
```

#### 回收策略 (Reclaim Policy)

当用户不再使用其存储卷时，他们可以从 API 中将 PVC 对象删除， 从而允许该资源被回收再利用。PersistentVolume 对象的回收策略告诉集群， 当其被从申领中释放时如何处理该数据卷。 目前，数据卷可以被 Retained（保留）、Recycled（回收）或 Deleted（删除）。

##### 1. 保留（Retain）
回收策略 Retain 使得用户可以手动回收资源。当 PersistentVolumeClaim 对象被删除时，PersistentVolume 卷仍然存在，对应的数据卷被视为"已释放（released）"。 由于卷上仍然存在这前一申领人的数据，该卷还不能用于其他申领。 管理员可以通过下面的步骤来手动回收该卷：

删除 PersistentVolume 对象。与之相关的、位于外部基础设施中的存储资产 （例如 AWS EBS、GCE PD、Azure Disk 或 Cinder 卷）在 PV 删除之后仍然存在。
根据情况，手动清除所关联的存储资产上的数据。
手动删除所关联的存储资产。
如果你希望重用该存储资产，可以基于存储资产的定义创建新的 PersistentVolume 卷对象。

##### 2. 删除（Delete）
对于支持 Delete 回收策略的卷插件，删除动作会将 PersistentVolume 对象从 Kubernetes 中移除，同时也会从外部基础设施（如 AWS EBS、GCE PD、Azure Disk 或 Cinder 卷）中移除所关联的存储资产。 动态制备的卷会继承其 StorageClass 中设置的回收策略， 该策略默认为 Delete。管理员需要根据用户的期望来配置 StorageClass； 否则 PV 卷被创建之后必须要被编辑或者修补。 参阅更改 PV 卷的回收策略。

##### 3. 回收（Recycle）
如果下层的卷插件支持，回收策略 Recycle 会在卷上执行一些基本的擦除 （rm -rf /thevolume/*）操作，之后允许该卷用于新的 PVC 。

### 临时卷

有些应用程序需要额外的存储，但并不关心数据在重启后是否仍然可用。 例如，缓存服务经常受限于内存大小，而且可以将不常用的数据转移到比内存慢的存储中，对总体性能的影响并不大。

另有些应用程序需要以文件形式注入的只读数据，比如配置数据或密钥。

临时卷 就是为此类用例设计的。因为卷会遵从 Pod 的生命周期，与 Pod 一起创建和删除， 所以停止和重新启动 Pod 时，不会受持久卷在何处可用的限制。

临时卷在 Pod 规约中以 内联 方式定义，这简化了应用程序的部署和管理。

Kubernetes 为了不同的用途，支持几种不同类型的临时卷：

emptyDir： Pod 启动时为空，存储空间来自本地的 kubelet 根目录（通常是根磁盘）或内存

configMap、 downwardAPI、 secret： 将不同类型的 Kubernetes 数据注入到 Pod 中

CSI 临时卷： 类似于前面的卷类型，但由专门支持此特性 的指定 CSI 驱动程序提供

通用临时卷： 它可以由所有支持持久卷的存储驱动程序提供

emptyDir、configMap、downwardAPI、secret 是作为 本地临时存储 提供的。它们由各个节点上的 kubelet 管理。

CSI 临时卷 必须 由第三方 CSI 存储驱动程序提供。

通用临时卷 可以 由第三方 CSI 存储驱动程序提供，也可以由支持动态制备的任何其他存储驱动程序提供。 一些专门为 CSI 临时卷编写的 CSI 驱动程序，不支持动态制备：因此这些驱动程序不能用于通用临时卷。

使用第三方驱动程序的优势在于，它们可以提供 Kubernetes 本身不支持的功能， 例如，与 kubelet 管理的磁盘具有不同性能特征的存储，或者用来注入不同的数据。

### 投射卷

一个 projected 卷可以将若干现有的卷源映射到同一个目录之上。

目前，源可以被投射的卷包括：secret, downwardAPI, configMap, serviceAccountToken。

所有的卷源都要求处于 Pod 所在的同一个名字空间内。进一步的详细信息，可参考 一体化卷设计文档。
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: volume-test
spec:
  containers:
  - name: container-test
    image: busybox:1.28
    volumeMounts:
    - name: all-in-one
      mountPath: "/projected-volume"
      readOnly: true
  volumes:
  - name: all-in-one
    projected:
      sources:
      - secret:
          name: mysecret
          items:
            - key: username
              path: my-group/my-username
      - downwardAPI:
          items:
            - path: "labels"
              fieldRef:
                fieldPath: metadata.labels
            - path: "cpu_limit"
              resourceFieldRef:
                containerName: container-test
                resource: limits.cpu
      - configMap:
          name: myconfigmap
          items:
            - key: config
              path: my-group/my-config
```
## 存储架构

![存储架构](/assets/images/kubernetes/storage-arch.png)

为了存储卷管理的功能，Kubernetes 设计了4大组件，从顶层向一下依次为：

Volume Plugins — 存储提供的扩展接口, 包含了各类存储提供者的plugin实现。其中Volume Plugins是一个基础部件，功能上来说Volume Manager、PV Controller、AD Controller三部分。

Volume Manager — 运行在kubelet 里让存储Ready的部件，负责管理数据卷的 Mount/Umount 操作（也负责数据卷的 Attach/Detach 操作，需配置 kubelet 相关参数开启该特性）、卷设备的格式化等等。

PV Controller — 运行在控制平面的 kube-controller-manager 上的组件，主要负责 PV/PVC 绑定及周期管理，根据需求进行对 PVC 的 Provision/Delete 操作；所谓将一个 PV 与 PVC 进行“绑定”，其实就是将这个 PV 对象的名字，填在了 PVC 对象的 spec.volumeName 字段上。

AD(Attach/Detach) Controller — 运行在控制平面的 kube-controller-manager 上的组件，负责数据卷的 Attach/Detach 操作，将设备挂接到目标节点。

### 存储卷挂载

PV Controller和K8S其它组件一样监听API Server中的资源更新，对于卷管理主要是监听PV，PVC， SC三类资源，当监听到这些资源的创建、删除、修改时，PV Controller经过判断是需要做创建、删除、绑定、回收等动作（后续会展开介绍内部逻辑），然后根据需要调用Volume Plugins进行业务处理。


![存储管理](/assets/images/kubernetes/kubernetes-storage.png)


存储卷管理核心做三个动作：

Provision/Delete （创建/删除存储卷，处理pv和pvc之间的关系）

Attach/Detach （挂接和摘除存储卷，处理的是volumes和node上目录之间的关系）

Mount/Unmount （挂载和摘除目录，处理的是volumes和pod之间的关系）

### CSI存储插件流程分析

以 nfs plugin 插件为例，分析 Kubernetes 集群如何为用户提供存储资源。

#### CSI 简介


csi插件的实现，官方已经封装好的lib，我们只要实现对应的接口就可以了，目前我们可以实现的插件有两种类型，如下：

Controller Plugin，负责存储对象（Volume）的生命周期管理，在集群中仅需要有一个即可；

Node Plugin，在必要时与使用 Volume 的容器所在的节点交互，提供诸如节点上的 Volume 挂载/卸载等动作支持，如有需要则在每个服务节点上均部署。

官方提供了rpc接口实现上面两个插件，如下：

##### CSI Identity Service(身份服务)
Node Plugin和Controller Plugin都必须实现这些RPC集，接口如下：

```go
service Identity {
    // GetPluginInfo， 获取 Plugin 基本信息
  rpc GetPluginInfo(GetPluginInfoRequest)
    returns (GetPluginInfoResponse) {}

    // GetPluginCapabilities，获取 Plugin 支持的能力
  rpc GetPluginCapabilities(GetPluginCapabilitiesRequest)
    returns (GetPluginCapabilitiesResponse) {}

    //Probe，探测 Plugin 的健康状态
  rpc Probe (ProbeRequest)
    returns (ProbeResponse) {}
}
```

##### CSI Controller Service(控制器服务)
控制器服务，Controller Plugin 必须实现这些RPC集。CSI Controller 服务里定义的这些操作有个共同特点，那就是它们都无需在宿主机上进行，而是属于 Kubernetes 里 Volume Controller 的逻辑，也就是属于 Master 节点的一部分。需要注意的是，正如我在前面提到的那样，CSI Controller 服务的实际调用者，并不是 Kubernetes（即：通过 pkg/volume/csi 发起 CSI 请求），而是 External Provisioner 和 External Attacher。这两个 External Components，分别通过监听 PVC 和 VolumeAttachement 对象，来跟 Kubernetes 进行协作。

```go
service Controller {
    // Volume CRUD，包括了扩容和容量探测等 Volume 状态检查与操作接口
  rpc CreateVolume (CreateVolumeRequest)
    returns (CreateVolumeResponse) {}

  rpc DeleteVolume (DeleteVolumeRequest)
    returns (DeleteVolumeResponse) {}

    // Publish/Unpublish ，也就是对 CSI Volume 进行 Attach/Dettach，还包括Node 对 Volume 的访问权限管理
  rpc ControllerPublishVolume (ControllerPublishVolumeRequest)
    returns (ControllerPublishVolumeResponse) {}

  rpc ControllerUnpublishVolume (ControllerUnpublishVolumeRequest)
    returns (ControllerUnpublishVolumeResponse) {}

  rpc ValidateVolumeCapabilities (ValidateVolumeCapabilitiesRequest)
    returns (ValidateVolumeCapabilitiesResponse) {}

  rpc ListVolumes (ListVolumesRequest)
    returns (ListVolumesResponse) {}

  rpc GetCapacity (GetCapacityRequest)
    returns (GetCapacityResponse) {}

  rpc ControllerGetCapabilities (ControllerGetCapabilitiesRequest)
    returns (ControllerGetCapabilitiesResponse) {}

    // Snapshot CRD，快照的创建和删除操作，目前 CSI 定义的 Snapshot 仅用于创建 Volume，未提供回滚的语义
  rpc CreateSnapshot (CreateSnapshotRequest)
    returns (CreateSnapshotResponse) {}

  rpc DeleteSnapshot (DeleteSnapshotRequest)
    returns (DeleteSnapshotResponse) {}

  rpc ListSnapshots (ListSnapshotsRequest)
    returns (ListSnapshotsResponse) {}

  rpc ControllerExpandVolume (ControllerExpandVolumeRequest)
    returns (ControllerExpandVolumeResponse) {}

  rpc ControllerGetVolume (ControllerGetVolumeRequest)
    returns (ControllerGetVolumeResponse) {
        option (alpha_method) = true;
    }
}
```
##### CSI Controller Service(控制器服务)
节点服务：Node Plugin必须实现这些RPC集。CSI Volume 需要在宿主机上执行的操作，都定义在了 CSI Node 服务里面
```go
service Node {
    // Node Stage/Unstage/Publish/Unpublish/GetStats Volume，节点上 Volume 的连接状态管理，也就是mount是由 NodeStageVolume 和 NodePublishVolume 两个接口共同实现的。
rpc NodeStageVolume (NodeStageVolumeRequest)
    returns (NodeStageVolumeResponse) {}

  rpc NodeUnstageVolume (NodeUnstageVolumeRequest)
    returns (NodeUnstageVolumeResponse) {}

  rpc NodePublishVolume (NodePublishVolumeRequest)
    returns (NodePublishVolumeResponse) {}

  rpc NodeUnpublishVolume (NodeUnpublishVolumeRequest)
    returns (NodeUnpublishVolumeResponse) {}

  rpc NodeGetVolumeStats (NodeGetVolumeStatsRequest)
    returns (NodeGetVolumeStatsResponse) {}

    // Node Expand Volume, 节点上的 Volume 扩容操作，在 volume 逻辑大小扩容之后，可能还需要同步的扩容 Volume 之上的文件系统并让使用 Volume 的 Container 感知到，所以在 Node Plugin 上需要有对应的接口
  rpc NodeExpandVolume(NodeExpandVolumeRequest)
    returns (NodeExpandVolumeResponse) {}

    // Node Get Capabilities/Info， Plugin 的基础属性与 Node 的属性查询

  rpc NodeGetCapabilities (NodeGetCapabilitiesRequest)
    returns (NodeGetCapabilitiesResponse) {}

  rpc NodeGetInfo (NodeGetInfoRequest)
    returns (NodeGetInfoResponse) {}
}
```

![csi-call](/assets/images/kubernetes/csi-caller.png)

Driver Registrar 组件，负责将插件注册到 kubelet 里面（这可以类比为，将可执行文件放在插件目录下）。而在具体实现上，Driver Registrar 需要请求 CSI 插件的 Identity 服务来获取插件信息。

External Provisioner 组件，负责的正是 Provision 阶段。在具体实现上，External Provisioner 监听（Watch）了 APIServer 里的 PVC 对象。当一个 PVC 被创建时，它就会调用 CSI Controller 的 CreateVolume 方法，为你创建对应 PV。

External Attacher 组件，负责的正是“Attach 阶段”。在具体实现上，它监听了 APIServer 里 VolumeAttachment 对象的变化。VolumeAttachment 对象是 Kubernetes 确认一个 Volume 可以进入“Attach 阶段”的重要标志。一旦出现了 VolumeAttachment 对象，External Attacher 就会调用 CSI Controller 服务的 ControllerPublish 方法，完成它所对应的 Volume 的 Attach 阶段。

Volume 的“Mount 阶段”，并不属于 External Components 的职责。当 kubelet 的 
VolumeManagerReconciler 控制循环检查到它需要执行 Mount 操作的时候，会通过 pkg/volume/csi 包，直接调用 CSI Node 服务完成 Volume 的“Mount 阶段”。

如果你要实现一个自己的 CSI Driver 你需要至少提供两个 gRPC 服务：CSI Identity service，负责 CSI 插件的识别工作；CSI Node Driver 负责


nfs plugin 项目路径： https://github.com/kubernetes-sigs/csi-driver-nfs
nfs controller 项目路径：https://github.com/kubernetes-sigs/external-provisioner

#### 部署

部署方式
```shell
# 安装存储插件
curl -skSL https://raw.githubusercontent.com/kubernetes-csi/csi-driver-nfs/master/deploy/install-driver.sh | bash -s master --

# 查看 Pod 工作状态
kubectl -n kube-system get pod -o wide -l app=csi-nfs-controller
kubectl -n kube-system get pod -o wide -l app=csi-nfs-node

# 结果
NAME                                       READY   STATUS    RESTARTS   AGE     IP             NODE
csi-nfs-controller-56bfddd689-dh5tk       4/4     Running   0          35s     10.240.0.19    k8s-agentpool-22533604-0
csi-nfs-node-cvgbs                        3/3     Running   0          35s     10.240.0.35    k8s-agentpool-22533604-1
csi-nfs-node-dr4s4                        3/3     Running   0          35s     10.240.0.4     k8s-agentpool-22533604-0
```

csi nfs driver 包含 controller 和 nodeplugin 两部分，controller 负责存储资源管理，nodeplugin 负责将存储卷加载到容器中。

controller pod 中有两个工作组件，一个是 provisioner 容器，负责的正是 Provision 阶段；一个 nfsplugin 容器，负责存储卷声明周期管理。

node pod 中有两个工作组件，一个 node-driver-registrar 容器，负责向 Kubelet 注册一个CSI 插件；一个 nfs 容器，负责 PV 挂载。

#### nfscontroller 源码解析
##### 1. 服务入口
controller 服务的主流程直接写在了main函数中，直接看代码：
```go
func main() {
	// ...

	// 快照客户端
	snapClient, err := snapclientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create snapshot client: %v", err)
	}

	// 资源采集器
	metricsManager := metrics.NewCSIMetricsManagerWithOptions("", /* driverName */
		// Will be provided via default gatherer.
		metrics.WithProcessStartTime(false),
		metrics.WithSubsystem(metrics.SubsystemSidecar),
	)

	grpcClient, err := ctrl.Connect(*csiEndpoint, metricsManager)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	err = ctrl.Probe(grpcClient, *operationTimeout)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// 查询集群中 driver 对应的 provisionerName
	provisionerName, err := ctrl.GetDriverName(grpcClient, *operationTimeout)
	if err != nil {
		klog.Fatalf("Error getting CSI driver name: %s", err)
	}
	klog.V(2).Infof("Detected CSI driver %s", provisionerName)
	metricsManager.SetDriverName(provisionerName)

	translator := csitrans.New()
	supportsMigrationFromInTreePluginName := ""
	if translator.IsMigratedCSIDriverByName(provisionerName) {
		supportsMigrationFromInTreePluginName, err = translator.GetInTreeNameFromCSIName(provisionerName)
		if err != nil {
			klog.Fatalf("Failed to get InTree plugin name for migrated CSI plugin %s: %v", provisionerName, err)
		}
		klog.V(2).Infof("Supports migration from in-tree plugin: %s", supportsMigrationFromInTreePluginName)

		// Create a new connection with the metrics manager with migrated label
		metricsManager = metrics.NewCSIMetricsManagerWithOptions(provisionerName,
			// Will be provided via default gatherer.
			metrics.WithProcessStartTime(false),
			metrics.WithMigration())
		migratedGrpcClient, err := ctrl.Connect(*csiEndpoint, metricsManager)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}
		grpcClient.Close()
		grpcClient = migratedGrpcClient

		err = ctrl.Probe(grpcClient, *operationTimeout)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}
	}

	// 准备资源采集、选主、监控检查
	mux := http.NewServeMux()
	gatherers := prometheus.Gatherers{
		legacyregistry.DefaultGatherer,
		metricsManager.GetRegistry(),
	}

	pluginCapabilities, controllerCapabilities, err := ctrl.GetDriverCapabilities(grpcClient, *operationTimeout)
	if err != nil {
		klog.Fatalf("Error getting CSI driver capabilities: %s", err)
	}

	// 为 provisioner 生成唯一的ID
	timeStamp := time.Now().UnixNano() / int64(time.Millisecond)
	identity := strconv.FormatInt(timeStamp, 10) + "-" + strconv.Itoa(rand.Intn(10000)) + "-" + provisionerName
	if *enableNodeDeployment {
		identity = identity + "-" + node
	}

	factory := informers.NewSharedInformerFactory(clientset, ctrl.ResyncPeriodOfCsiNodeInformer)
	var factoryForNamespace informers.SharedInformerFactory 

	// 创建 informer 监听所有资源请求

	// 监听 StorageClass 和 PVC
	scLister := factory.Storage().V1().StorageClasses().Lister()
	claimLister := factory.Core().V1().PersistentVolumeClaims().Lister()

	var vaLister storagelistersv1.VolumeAttachmentLister
	if controllerCapabilities[csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME] {
		klog.Info("CSI driver supports PUBLISH_UNPUBLISH_VOLUME, watching VolumeAttachments")
		vaLister = factory.Storage().V1().VolumeAttachments().Lister()
	} else {
		klog.Info("CSI driver does not support PUBLISH_UNPUBLISH_VOLUME, not watching VolumeAttachments")
	}

    // 默认不开启 enableNodeDeployment
	var nodeDeployment *ctrl.NodeDeployment
	if *enableNodeDeployment {
		nodeDeployment = &ctrl.NodeDeployment{
			NodeName:         node,
			ClaimInformer:    factory.Core().V1().PersistentVolumeClaims(),
			ImmediateBinding: *nodeDeploymentImmediateBinding,
			BaseDelay:        *nodeDeploymentBaseDelay,
			MaxDelay:         *nodeDeploymentMaxDelay,
		}
		nodeInfo, err := ctrl.GetNodeInfo(grpcClient, *operationTimeout)
		if err != nil {
			klog.Fatalf("Failed to get node info from CSI driver: %v", err)
		}
		nodeDeployment.NodeInfo = *nodeInfo
	}

	// 监听 所有节点 和 CSI 对应的存储节点
	var nodeLister listersv1.NodeLister
	var csiNodeLister storagelistersv1.CSINodeLister
	if ctrl.SupportsTopology(pluginCapabilities) {
		if nodeDeployment != nil {
			// Avoid watching in favor of fake, static objects. This is particularly relevant for
			// Node objects, which can generate significant traffic.
			csiNode := &storagev1.CSINode{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeDeployment.NodeName,
				},
				Spec: storagev1.CSINodeSpec{
					Drivers: []storagev1.CSINodeDriver{
						{
							Name:   provisionerName,
							NodeID: nodeDeployment.NodeInfo.NodeId,
						},
					},
				},
			}
			node := &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeDeployment.NodeName,
				},
			}
			if nodeDeployment.NodeInfo.AccessibleTopology != nil {
				for key := range nodeDeployment.NodeInfo.AccessibleTopology.Segments {
					csiNode.Spec.Drivers[0].TopologyKeys = append(csiNode.Spec.Drivers[0].TopologyKeys, key)
				}
				node.Labels = nodeDeployment.NodeInfo.AccessibleTopology.Segments
			}
			klog.Infof("using local topology with Node = %+v and CSINode = %+v", node, csiNode)

			// We make those fake objects available to the topology code via informers which
			// never change.
			stoppedFactory := informers.NewSharedInformerFactory(clientset, 1000*time.Hour)
			csiNodes := stoppedFactory.Storage().V1().CSINodes()
			nodes := stoppedFactory.Core().V1().Nodes()
			csiNodes.Informer().GetStore().Add(csiNode)
			nodes.Informer().GetStore().Add(node)
			csiNodeLister = csiNodes.Lister()
			nodeLister = nodes.Lister()

		} else {
			csiNodeLister = factory.Storage().V1().CSINodes().Lister()
			nodeLister = factory.Core().V1().Nodes().Lister()
		}
	}

	// PVC Informer
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(*retryIntervalStart, *retryIntervalMax)
	claimQueue := workqueue.NewNamedRateLimitingQueue(rateLimiter, "claims")
	claimInformer := factory.Core().V1().PersistentVolumeClaims().Informer()

	// Setup options
	provisionerOptions := []func(*controller.ProvisionController) error{
		controller.LeaderElection(false), // Always disable leader election in provisioner lib. Leader election should be done here in the CSI provisioner level instead.
		controller.FailedProvisionThreshold(0),
		controller.FailedDeleteThreshold(0),
		controller.RateLimiter(rateLimiter),
		controller.Threadiness(int(*workerThreads)),
		controller.CreateProvisionedPVLimiter(workqueue.DefaultControllerRateLimiter()),
		controller.ClaimsInformer(claimInformer),
		controller.NodesLister(nodeLister),
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.HonorPVReclaimPolicy) {
		provisionerOptions = append(provisionerOptions, controller.AddFinalizer(true))
	}

	if supportsMigrationFromInTreePluginName != "" {
		provisionerOptions = append(provisionerOptions, controller.AdditionalProvisionerNames([]string{supportsMigrationFromInTreePluginName}))
	}

	// 创建一个 provisioner 和两个 controller 
	csiProvisioner := ctrl.NewCSIProvisioner(
		clientset,
		*operationTimeout,
		identity,
		*volumeNamePrefix,
		*volumeNameUUIDLength,
		grpcClient,
		snapClient,
		provisionerName,
		pluginCapabilities,
		controllerCapabilities,
		supportsMigrationFromInTreePluginName,
		*strictTopology,
		*immediateTopology,
		translator,
		scLister,
		csiNodeLister,
		nodeLister,
		claimLister,
		vaLister,
		*extraCreateMetadata,
		*defaultFSType,
		nodeDeployment,
		*controllerPublishReadOnly,
		*preventVolumeModeConversion,
	)

	// ...

	provisionController = controller.NewProvisionController(
		clientset,
		provisionerName,
		csiProvisioner,
		provisionerOptions...,
	)

	csiClaimController := ctrl.NewCloningProtectionController(
		clientset,
		claimLister,
		claimInformer,
		claimQueue,
		controllerCapabilities,
	)

	// 启动 http 服务
	if addr != "" {
		// ...

		go func() {
			klog.Infof("ServeMux listening at %q", addr)
			err := http.ListenAndServe(addr, mux)
			if err != nil {
				klog.Fatalf("Failed to start HTTP server at specified address (%q) and metrics path (%q): %s", addr, *metricsPath, err)
			}
		}()
	}

	run := func(ctx context.Context) {
		factory.Start(ctx.Done())
		if factoryForNamespace != nil {
			// Starting is enough, the capacity controller will
			// wait for sync.
			factoryForNamespace.Start(ctx.Done())
		}

		// informer 缓存同步
		cacheSyncResult := factory.WaitForCacheSync(ctx.Done())
		for _, v := range cacheSyncResult {
			if !v {
				klog.Fatalf("Failed to sync Informers!")
			}
		}

        // 启动控制器
		if capacityController != nil {
			go capacityController.Run(ctx, int(*capacityThreads))
		}
		if csiClaimController != nil {
			go csiClaimController.Run(ctx, int(*finalizerThreads))
		}
		provisionController.Run(ctx)
	}

	// ...
}
```

external-provisioner中所实现的控制器均继承自官方提供的 lib 仓库，下面看一下源码：

```go
func (ctrl *ProvisionController) Run(ctx context.Context) {
	run := func(ctx context.Context) {
		klog.Infof("Starting provisioner controller %s!", ctrl.component)
		defer utilruntime.HandleCrash()
		defer ctrl.claimQueue.ShutDown()
		defer ctrl.volumeQueue.ShutDown()

		ctrl.hasRunLock.Lock()
		ctrl.hasRun = true
		ctrl.hasRunLock.Unlock()
		if ctrl.metricsPort > 0 {
			// ...
		}

		// 将 controller 初始化传入的 Informer 启动
		if !ctrl.customClaimInformer {
			go ctrl.claimInformer.Run(ctx.Done())
		}
		if !ctrl.customVolumeInformer {
			go ctrl.volumeInformer.Run(ctx.Done())
		}
		if !ctrl.customClassInformer {
			go ctrl.classInformer.Run(ctx.Done())
		}

		// informer 缓存同步
		if !cache.WaitForCacheSync(ctx.Done(), ctrl.claimInformer.HasSynced, ctrl.volumeInformer.HasSynced, ctrl.classInformer.HasSynced) {
			return
		}

		// 启动 goroutine 处理由于被监听到发生了增删改查事件的 PV 和 PVC 队列
		for i := 0; i < ctrl.threadiness; i++ {
			go wait.Until(func() { ctrl.runClaimWorker(ctx) }, time.Second, ctx.Done())
			go wait.Until(func() { ctrl.runVolumeWorker(ctx) }, time.Second, ctx.Done())
		}

		klog.Infof("Started provisioner controller %s!", ctrl.component)

		select {}
	}

	go ctrl.volumeStore.Run(ctx, DefaultThreadiness)

	if ctrl.leaderElection {
		// ...
	} else {
		run(ctx)
	}
}
```

事件处理的 runClaimWorker 和 runVolumeWorker 会去调用我们声明在 external-provsioner 中的Controller Service 接口，具体的调用过程这里不展开，如果有兴趣可以到官方仓库查看源码：  https://github.com/kubernetes-sigs/sig-storage-lib-external-provisioner


#### 2. 控制器服务结构
```go
func (p *csiProvisioner) Provision(ctx context.Context, options controller.ProvisionOptions) (*v1.PersistentVolume, controller.ProvisioningState, error) {
	claim := options.PVC
	provisioner, ok := claim.Annotations[annStorageProvisioner]
	if !ok {
		provisioner = claim.Annotations[annBetaStorageProvisioner]
	}

	// ...

	// 检测工作节点
	owned, err := p.checkNode(ctx, claim, options.StorageClass, "provision")
	if err != nil {
		return nil, controller.ProvisioningNoChange,
			fmt.Errorf("node check failed: %v", err)
	}
	if !owned {
		return nil, controller.ProvisioningNoChange, &controller.IgnoredError{
			Reason: fmt.Sprintf("not responsible for provisioning of PVC %s/%s because it is not assigned to node %q", claim.Namespace, claim.Name, p.nodeDeployment.NodeName),
		}
	}

	// provision 预处理
	result, state, err := p.prepareProvision(ctx, claim, options.StorageClass, options.SelectedNode)
	if result == nil {
		return nil, state, err
	}

	// ...

	// 创建 PV
	rep, err := p.csiClient.CreateVolume(createCtx, req)
	if err != nil {
		mayReschedule := p.supportsTopology() &&
			options.SelectedNode != nil
		state := checkError(err, mayReschedule)
		klog.V(5).Infof("CreateVolume failed, supports topology = %v, node selected %v => may reschedule = %v => state = %v: %v",
			p.supportsTopology(),
			options.SelectedNode != nil,
			mayReschedule,
			state,
			err)
		return nil, state, err
	}

	// ...


	// 将 PV 创建得到的返回值写回到预处理结果中
	if len(volCaps) == 1 && volCaps[0].GetAccessMode().GetMode() == csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY && p.controllerPublishReadOnly {
		pvReadOnly = true
	}

	result.csiPVSource.VolumeHandle = p.volumeIdToHandle(rep.Volume.VolumeId)
	result.csiPVSource.VolumeAttributes = volumeAttributes
	result.csiPVSource.ReadOnly = pvReadOnly
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: v1.PersistentVolumeSpec{
			AccessModes:  options.PVC.Spec.AccessModes,
			MountOptions: options.StorageClass.MountOptions,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): bytesToQuantity(respCap),
			},
			// TODO wait for CSI VolumeSource API
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: result.csiPVSource,
			},
		},
	}

	// 修改 PV 资源的 annDeletionSecretRefName 和 namespace
	if result.provDeletionSecrets != nil {
		klog.V(5).Infof("createVolumeOperation: set annotation [%s/%s] on pv [%s].", annDeletionProvisionerSecretRefNamespace, annDeletionProvisionerSecretRefName, pv.Name)
		metav1.SetMetaDataAnnotation(&pv.ObjectMeta, annDeletionProvisionerSecretRefName, result.provDeletionSecrets.name)
		metav1.SetMetaDataAnnotation(&pv.ObjectMeta, annDeletionProvisionerSecretRefNamespace, result.provDeletionSecrets.namespace)
	} else {
		metav1.SetMetaDataAnnotation(&pv.ObjectMeta, annDeletionProvisionerSecretRefName, "")
		metav1.SetMetaDataAnnotation(&pv.ObjectMeta, annDeletionProvisionerSecretRefNamespace, "")
	}

	if options.StorageClass.ReclaimPolicy != nil {
		pv.Spec.PersistentVolumeReclaimPolicy = *options.StorageClass.ReclaimPolicy
	}

	// ...

	return pv, controller.ProvisioningFinished, nil
}

func (p *csiProvisioner) Delete(ctx context.Context, volume *v1.PersistentVolume) error {
	// ...

    // 从 PV 中获取 volumeId
	volumeId := p.volumeHandleToId(volume.Spec.CSI.VolumeHandle)

    // ...

    // 删除 PV 资源，调用 CSI 客户端删除存储卷
	if err := p.canDeleteVolume(volume); err != nil {
		return err
	}

	_, err = p.csiClient.DeleteVolume(deleteCtx, &req)

	return err
}
```
external-provisioner 职责到这里就结束了，后需要处理将由 kube-controller-manager中的AD controller 来接管。

#### nfsplugin 源码解析
##### 1. 程序入口
从程序入口出发，实际的服务启动逻辑在driver.Run函数中。

我们可以看到服务启动时，三个CSI Service(CSI Identity , CSI Controller , CSI Node)全都被初始化了，一般我们开发插件时会把三者放在一个二进制程序中。

```go
func (n *Driver) Run(testMode bool) {
	versionMeta, err := GetVersionYAML(n.name)
	if err != nil {
		klog.Fatalf("%v", err)
	}
	klog.V(2).Infof("\nDRIVER INFORMATION:\n-------------------\n%s\n\nStreaming logs below:", versionMeta)

	n.ns = NewNodeServer(n, mount.New(""))
	s := NewNonBlockingGRPCServer()
	s.Start(n.endpoint,
		NewDefaultIdentityServer(n),
		// NFS plugin has not implemented ControllerServer
		// using default controllerserver.
		NewControllerServer(n),
		n.ns,
		testMode)
	s.Wait()
}
```

### 2. 服务启动

```go
func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer, testMode bool) {

	proto, addr, err := ParseEndpoint(endpoint)
	if err != nil {
		klog.Fatal(err.Error())
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			klog.Fatalf("Failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	s.server = server

    // 注册 gRPC 服务
	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}

	// Used to stop the server while running tests
	if testMode {
		s.wg.Done()
		go func() {
			// make sure Serve() is called
			s.wg.Wait()
			time.Sleep(time.Millisecond * 1000)
			s.server.GracefulStop()
		}()
	}

	klog.Infof("Listening for connections on address: %#v", listener.Addr())

	err = server.Serve(listener)
	if err != nil {
		klog.Fatalf("Failed to serve grpc server: %v", err)
	}
}
```



#### 1. Provision


### Kubelet存储管理源码分析


### 调度器存储插件源码分析

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io