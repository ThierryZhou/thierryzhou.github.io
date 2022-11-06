---
title: "Kubernetes 存储详解"
---
## 概述
由于容器中的文件在磁盘上是临时存放的，这样的设计带来了两个问题，其一是如果容器因为某些原因被kubelet重启后，会丢失这些文件；其二是不同的容器之间无法共享文件。为了解决上述问题，Kubernetes 设计了卷（Volume） 这一抽象来管理容器中所有需要使用到的外部文件。

并且由于容器本身存在声明周期以及容器的存储来源有多样性的特点，卷本身带有独立的状态标识实现其生命周期循环，根绝业务场景的不同，卷又细分为持久卷、临时卷、投射卷三大类。

为了让集群管理员可以管理更多不同特性的持久卷，Kubernetes 又设计了存储类（StorageClass) 来管理每一类具有相同特性的持久卷，在后续的Kubernetes 版本迭代过程中陆续加入了一些其他的特性，例如：为了可以将任意第三方存储暴露给容器，增加了容器存储接口 (CSI)；为了增加存储的可靠性，增加类似传统文件系统中快照概念的卷快照 (Snapshot) 资源；为了可以对接任意第三方对象存储，增加了容器对象存储接口(COSI)。

### 持久卷(Persistent Volume)
PersistentVolume 子系统为用户和管理员提供了一组 API，通过引入了两个新的 API 资源：PersistentVolume 存储提供者进行管理； PersistentVolumeClaim 被存储使用者引用。

持久卷（PersistentVolume，PV） 是集群中的一块存储，可以由管理员事先制备 (Provision) ， 或者使用存储类（Storage Class）来动态制备 (Provision) 。 持久卷是集群资源，就像节点也是集群资源一样。PV 持久卷和普通的 Volume 一样， 也是使用卷插件来实现的，只是它们拥有独立于任何使用 PV 的 Pod 的生命周期。

持久卷申领（PersistentVolumeClaim，PVC） 表达的是用户对存储的请求。概念上与 Pod 类似。 Pod 会耗用节点资源，而 PVC 申领会耗用 PV 资源。Pod 可以请求特定数量的资源（CPU 和内存）；同样 PVC 申领也可以请求特定的大小和访问模式 （例如，可以要求 PV 卷能够以 ReadWriteOnce、ReadOnlyMany 或 ReadWriteMany 模式之一来挂载，参见访问模式）。

尽管 PersistentVolumeClaim 允许用户消耗抽象的存储资源， 常见的情况是针对不同的问题用户需要的是具有不同属性（如，性能）的 PersistentVolume 卷。 集群管理员需要能够提供不同性质的 PersistentVolume， 并且这些 PV 卷之间的差别不仅限于卷大小和访问模式，同时又不能将卷是如何实现的这些细节暴露给用户。 为了满足这类需求，就有了存储类（StorageClass） 资源。

#### 生命周期

1. 制备 (Provision)
制备(Provision)一般是准备 Kubernetes 中的 PV 卷资源以及 PV 卷锁对应的存储设备。PV 卷的制备有两种方式：静态制备或动态制备。静态制备，集群管理员创建若干 PV 卷，这些卷对象带有真实存储的细节信息，Pod 中每声明一个PVC 资源，则消费一个与之对应的 PV 资源；动态制备，集群管理员创建一个存储类(StorageClass)，Pod 中每声明一个与存储类对应的 PVC 资源，又存储卷动态的创建一个 PV 资源来给 PVC 进行消费。

为了基于存储类完成动态的存储制备，集群管理员需要在 API 服务器上启用 DefaultStorageClass 准入控制器。 举例而言，可以通过保证 DefaultStorageClass 出现在 API 服务器组件的 --enable-admission-plugins 标志值中实现这点；该标志的值可以是逗号分隔的有序列表。 关于 API 服务器标志的更多信息，可以参考 kube-apiserver 文档。

2. 绑定 (Bind)
用户创建一个带有特定存储容量和特定访问模式需求的 PersistentVolumeClaim 对象； 在动态制备场景下，这个 PVC 对象可能已经创建完毕。 主控节点中的控制回路监测新的 PVC 对象，寻找与之匹配的 PV 卷（如果可能的话）， 并将二者绑定到一起。 如果为了新的 PVC 申领动态制备了 PV 卷，则控制回路总是将该 PV 卷绑定到这一 PVC 申领。 否则，用户总是能够获得他们所请求的资源，只是所获得的 PV 卷可能会超出所请求的配置。 一旦绑定关系建立，则 PersistentVolumeClaim 绑定就是排他性的， 无论该 PVC 申领是如何与 PV 卷建立的绑定关系。 PVC 申领与 PV 卷之间的绑定是一种一对一的映射，实现上使用 ClaimRef 来记述 PV 卷与 PVC 申领间的双向绑定关系。

如果找不到匹配的 PV 卷，PVC 申领会无限期地处于未绑定状态。 当与之匹配的 PV 卷可用时，PVC 申领会被绑定。 例如，即使某集群上制备了很多 50 Gi 大小的 PV 卷，也无法与请求 100 Gi 大小的存储的 PVC 匹配。当新的 100 Gi PV 卷被加入到集群时， 该 PVC 才有可能被绑定。

3. 使用
Pod 将 PVC 申领当做存储卷来使用。集群会检视 PVC 申领，找到所绑定的卷， 并为 Pod 挂载该卷。对于支持多种访问模式的卷， 用户要在 Pod 中以卷的形式使用申领时指定期望的访问模式。

一旦用户有了申领对象并且该申领已经被绑定， 则所绑定的 PV 卷在用户仍然需要它期间一直属于该用户。 用户通过在 Pod 的 volumes 块中包含 persistentVolumeClaim 节区来调度 Pod，访问所申领的 PV 卷。 相关细节可参阅使用申领作为卷。

#### 回收策略 (Reclaim Policy)
当用户不再使用其存储卷时，他们可以从 API 中将 PVC 对象删除， 从而允许该资源被回收再利用。PersistentVolume 对象的回收策略告诉集群， 当其被从申领中释放时如何处理该数据卷。 目前，数据卷可以被 Retained（保留）、Recycled（回收）或 Deleted（删除）。

1. 保留（Retain）
回收策略 Retain 使得用户可以手动回收资源。当 PersistentVolumeClaim 对象被删除时，PersistentVolume 卷仍然存在，对应的数据卷被视为"已释放（released）"。 由于卷上仍然存在这前一申领人的数据，该卷还不能用于其他申领。 管理员可以通过下面的步骤来手动回收该卷：

删除 PersistentVolume 对象。与之相关的、位于外部基础设施中的存储资产 （例如 AWS EBS、GCE PD、Azure Disk 或 Cinder 卷）在 PV 删除之后仍然存在。
根据情况，手动清除所关联的存储资产上的数据。
手动删除所关联的存储资产。
如果你希望重用该存储资产，可以基于存储资产的定义创建新的 PersistentVolume 卷对象。

2. 删除（Delete）
对于支持 Delete 回收策略的卷插件，删除动作会将 PersistentVolume 对象从 Kubernetes 中移除，同时也会从外部基础设施（如 AWS EBS、GCE PD、Azure Disk 或 Cinder 卷）中移除所关联的存储资产。 动态制备的卷会继承其 StorageClass 中设置的回收策略， 该策略默认为 Delete。管理员需要根据用户的期望来配置 StorageClass； 否则 PV 卷被创建之后必须要被编辑或者修补。 参阅更改 PV 卷的回收策略。

3. 回收（Recycle）
如果下层的卷插件支持，回收策略 Recycle 会在卷上执行一些基本的擦除 （rm -rf /thevolume/*）操作，之后允许该卷用于新的 PVC 申领。

不过，管理员可以按 参考资料 中所述，使用 Kubernetes 控制器管理器命令行参数来配置一个定制的回收器（Recycler） Pod 模板。此定制的回收器 Pod 模板必须包含一个 volumes 规约，如下例所示：
```yaml

```

### 临时卷

本文档描述 Kubernetes 中的 临时卷（Ephemeral Volume）。 建议先了解卷，特别是 PersistentVolumeClaim 和 PersistentVolume。

有些应用程序需要额外的存储，但并不关心数据在重启后是否仍然可用。 例如，缓存服务经常受限于内存大小，而且可以将不常用的数据转移到比内存慢的存储中，对总体性能的影响并不大。

另有些应用程序需要以文件形式注入的只读数据，比如配置数据或密钥。

临时卷 就是为此类用例设计的。因为卷会遵从 Pod 的生命周期，与 Pod 一起创建和删除， 所以停止和重新启动 Pod 时，不会受持久卷在何处可用的限制。

临时卷在 Pod 规约中以 内联 方式定义，这简化了应用程序的部署和管理。

临时卷的类型
Kubernetes 为了不同的用途，支持几种不同类型的临时卷：

emptyDir： Pod 启动时为空，存储空间来自本地的 kubelet 根目录（通常是根磁盘）或内存
configMap、 downwardAPI、 secret： 将不同类型的 Kubernetes 数据注入到 Pod 中
CSI 临时卷： 类似于前面的卷类型，但由专门支持此特性 的指定 CSI 驱动程序提供
通用临时卷： 它可以由所有支持持久卷的存储驱动程序提供
emptyDir、configMap、downwardAPI、secret 是作为 本地临时存储 提供的。它们由各个节点上的 kubelet 管理。

CSI 临时卷 必须 由第三方 CSI 存储驱动程序提供。

通用临时卷 可以 由第三方 CSI 存储驱动程序提供，也可以由支持动态制备的任何其他存储驱动程序提供。 一些专门为 CSI 临时卷编写的 CSI 驱动程序，不支持动态制备：因此这些驱动程序不能用于通用临时卷。

使用第三方驱动程序的优势在于，它们可以提供 Kubernetes 本身不支持的功能， 例如，与 kubelet 管理的磁盘具有不同性能特征的存储，或者用来注入不同的数据。

### 映射卷

投射卷
本文档描述 Kubernetes 中的投射卷（Projected Volumes）。 建议先熟悉卷概念。

介绍
一个 projected 卷可以将若干现有的卷源映射到同一个目录之上。

目前，以下类型的卷源可以被投射：

secret
downwardAPI
configMap
serviceAccountToken
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

K8S里存储相关的组件，从顶层来讲，主要包含4大组件：

Volume Plugins — 存储提供的扩展接口, 包含了各类存储提供者的plugin实现。
Volume Manager — 运行在kubelet 里让存储Ready的部件，负责管理数据卷的 Mount/Umount 操作（也负责数据卷的 Attach/Detach 操作，需配置 kubelet 相关参数开启该特性）、卷设备的格式化等等。
PV/PVC Controller — 运行在Master上的部件，主要提供卷生命周期管理，负责 PV/PVC 绑定及周期管理，根据需求进行数据卷的 Provision/Delete 操作；所谓将一个 PV 与 PVC 进行“绑定”，其实就是将这个 PV 对象的名字，填在了 PVC 对象的 spec.volumeName 字段上。
Attach/Detach — 运行在Master上，负责数据卷的 Attach/Detach 操作，将设备挂接到目标节点。
其中Volume Plugins是一个基础部件，后三个是逻辑部件，依赖于Volume Plugins。

上诉其实就是K8S内部的基本逻辑架构，扩展出去再加上外部与这些部件有交互关系的部件(调用者和实现者)和内部可靠性保证的部件，就可以得出K8S的存储的架构全景。

![存储架构](/assets/kubernetes/storage-arch-1.png)

Docker 也有卷（Volume） 的概念，但对它只有少量且松散的管理。 Docker 卷是磁盘上或者另外一个容器内的一个目录。 Docker 提供卷驱动程序，但是其功能非常有限。-

Kubernetes 支持很多类型的卷。 Pod 可以同时使用任意数目的卷类型。 临时卷类型的生命周期与 Pod 相同，但持久卷可以比 Pod 的存活期长。 当 Pod 不再存在时，Kubernetes 也会销毁临时卷；不过 Kubernetes 不会销毁持久卷。 对于给定 Pod 中任何类型的卷，在容器重启期间数据都不会丢失。

卷的核心是一个目录，其中可能存有数据，Pod 中的容器可以访问该目录中的数据。 所采用的特定的卷类型将决定该目录如何形成的、使用何种介质保存数据以及目录中存放的内容。

使用卷时, 在 .spec.volumes 字段中设置为 Pod 提供的卷，并在 .spec.containers[*].volumeMounts 字段中声明卷在容器中的挂载位置。 容器中的进程看到的文件系统视图是由它们的 容器镜像 的初始内容以及挂载在容器中的卷（如果定义了的话）所组成的。 其中根文件系统同容器镜像的内容相吻合。 任何在该文件系统下的写入操作，如果被允许的话，都会影响接下来容器中进程访问文件系统时所看到的内容。

卷挂载在镜像中的指定路径下。 Pod 配置中的每个容器必须独立指定各个卷的挂载位置。

卷不能挂载到其他卷之上（不过存在一种使用 subPath 的相关机制），也不能与其他卷有硬链接。

![存储架构](/assets/kubernetes/storage-arch-2.png)

### 存储控制器源码分析


### Kubelet存储管理源码分析


### 调度器存储插件源码分析