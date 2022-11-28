---
title: "Kubernetes 网络模型"
tag: kubernetes
excerpt: 集群网络系统是 Kubernetes 的核心部分，但是想要准确了解它的工作原理可是个不小的挑战。
---

集群网络系统是 Kubernetes 的核心部分，但是想要准确了解它的工作原理可是个不小的挑战。下面列出的是网络系统的的四个主要问题：

1. 高度耦合的容器间通信：这个已经被 Pod 和 localhost 通信解决了。
2. Pod 间通信：这是本文档讲述的重点。
3. Pod 与 Service 间通信：涵盖在 Service 中。
4. 外部与 Service 间通信：也涵盖在 Service 中。

Kubernetes 的宗旨就是在应用之间共享机器。 通常来说，共享机器需要两个应用之间不能使用相同的端口，但是在多个应用开发者之间去大规模地协调端口是件很困难的事情，尤其是还要让用户暴露在他们控制范围之外的集群级别的问题上。

为每个应用都需要设置一个端口的参数，而 API 服务器还需要知道如何将动态端口数值插入到配置模块中，服务也需要知道如何找到对方等等。这无疑会为系统带来很多复杂度，Kubernetes 选择了增加抽象层的方式去分解系统的复杂度。

## Kubernetes网络模型

集群中每一个 Pod 都会获得一个独一无二的 IP 地址， 这就意味着你不需要显式地在 Pod 之间创建链接，不需要处理容器端口到主机端口之间的映射。kubernetes的网络模型里，Pod 可以被视作虚拟机或者物理主机。

Kubernetes 强制要求所有网络设施都满足以下基本要求（从而排除了有意隔离网络的策略）：  
1. Pod 能够与所有其他节点上的 Pod 通信， 且不需要网络地址转译(NAT)
2. 节点上的守护进程(kubelet)可以和节点上的所有 Pod 通信

如果你的任务开始是在虚拟机中运行的，你的虚拟机有一个 IP， 可以和项目中其他虚拟机通信。这里的模型是基本相同的。这与 kubernetes 的网络模型基本相同，它可以帮助你实现从虚拟机向容器平滑迁移。

![kubernetes-network](/assets/images/kubernetes/kubernetes-network.png)

Kubernetes 的 IP 地址存在于 Pod 范围内，容器共享它们的网络命名空间，包括它们的 IP 地址和 MAC 地址。这就意味着 Pod 内的容器都可以通过 localhost 到达对方端口。 这和虚拟机中的进程似乎没有什么不同， 这也被称为“一个 Pod 一个 IP”模型。

实现以上需求，Kubernetes 网络还需要解决四方面的问题：

1. 一个 Pod 中的容器之间通过本地回路（loopback）通信。
2. 集群网络在不同 Pod 之间提供通信。
3. 向外暴露 Pod 中运行的应用， 以支持来自于集群外部的访问。
4. 提供专门用于暴露 HTTP 应用程序、网站和 API 的额外功能

第一个问题，同一个 Pod 的容器共享同一个网络命名空间，它们之间的访问可以用 localhost 地址 + 容器端口就可以访问。

第二个问题，提供一种容器网络机制，将 Pod 的 IP 和所在的 Node 的 IP 关联起来，通过这个关联让 Pod 可以互相访问。

第三个问题， kubernetes 提供了 Service 资源允许你 向外暴露 Pod 中运行的应用， 以支持来自于集群外部的访问，Service是一组Pod的服务抽象，相当于一组Pod的LB，负责将请求分发给对应的 Pod，Service会为这个LB提供一个IP，一般称为ClusterIP。

第四个问题， kubernetes 提供了 Ingress 资源于暴露 HTTP 应用程序、网站和 API 等。

你也可以使用 Service 来发布仅供集群内部使用的服务。集群网络解释了如何为集群设置网络， 还概述了所涉及的技术。
在Kubernetes网络中存在两种IP（Pod IP 和 Service Cluster IP），Pod IP 地址是实际存在于某个网卡(可以是虚拟设备)上的，Service Cluster IP它是一个虚拟IP，是由kube-proxy使用Iptables规则重新定向到其本地端口，再均衡到后端Pod的。下面讲讲Kubernetes Pod网络设计模型：
1. 基本原则：每个Pod都拥有一个独立的IP地址（IP per Pod），而且假定所有的pod都在一个可以直接连通的、扁平的网络空间中。
2. 设计原因：用户不需要额外考虑如何建立Pod之间的连接，也不需要考虑将容器端口映射到主机端口等问题。
3. 网络要求：所有的容器都可以在不用NAT的方式下同别的容器通讯；所有节点都可在不用NAT的方式下同所有容器通讯；容器的地址和别人看到的地址是同一个地址。

## Kubernetes Service

service 关于 Cluster IP 的定义有点类似 linux 服务器上常用的 VIP。 kubernetes 在每个节点上都会启动一个名为 Kube-proxy 的守护进程，它会维护一张 Cluster IP 与 Pod IP 映射表，用户请求到 Cluster IP 后，这条请求根据  Cluster IP 与 Pod IP 映射表，被转发到对应的 Pod 中。

除了通过 IP 访问 Service，kubernetes 还提供了一种机制允许用户可以通过 service name 与 Pod进行通信。kubernetes 一般会部署 一个 名为 coredns 的 DNS 服务，它用来为 service分配子域名，在集群中可以通过名称访问service；通常 coredns 会为 service 赋予一个名为“service-name.namespace.svc.cluster.local”的A记录，用来解析 service 的 Cluster IP。

## Kubernetes 网络插件

如何实现 Pod IP，如何实现 Pod IP之间的通信，Kubernetes 并没有给出具体的实现方案。但是社会根据上面提过的设计思路，提供了一套名为容器网络接口 (CNI)的，Kubernetes 网络插件协议。

CNI（Container Network Interface）是 CNCF 旗下的一个项目，由一组用于配置 Linux 容器的网络接口的规范和库组成，同时还包含了一些插件。CNI 仅关心容器创建时的网络分配，和当容器被删除时释放网络资源。通过此链接浏览该项目：https://github.com/containernetworking/cni。

### CNI 工作原理

kubernetes 控制平面完成了 deployment、ststateful、pod 集群负载的调度后，kubelet 开始 pod 的创建工作，当创建 pod 时 kubelet 会通过预定义的 cni 回调调用网络插件（比如 flannel、calico、ovs）实现网络管理（例如：创建、回收 IP等工作）。在早起的 kubernetes 版本(使用运行时 docker 及 docker-shim )中 kubelet 通过命令行参数 cni-bin-dir 和 cni-conf-dir 获知 cni 插件的二进制文件及配置文件的路径，启用 cni 插件 。cni-bin-dir 默认路径为 /opt/cni/bin ，是各个计算节点的放着这些插件的路径，cni-conf-dir 默认指向 /etc/cni/net.d， 是 cni 配置文件存放在的目录。
```shell
$ ls -l /etc/cni/net.d/
total 4
-rw-r--r-- 1 root root 311 Nov 18 17:13 10-flannel.conflist
$ cat /etc/cni/net.d/10-flannel.conflist
{
  "name":"cbr0",
  "cniVersion":"1.0.0",
  "plugins":[
    {
      "type":"flannel",
      "delegate":{
        "hairpinMode":true,
        "forceAddress":true,
        "isDefaultGateway":true
      }
    },
    {
      "type":"portmap",
      "capabilities":{
        "portMappings":true
      }
    }
  ]
}
```
改用containerd作为容器运行时后，搜索和加载 cni 网络插件的任务由 containerd 接管。其工作流程如下。
![containerd-cni](/assets/images/posts/containered-cni.png)

这里 containerd 中所描述的 pod 与 kubernetes 中的 pod 是相同概念，而 Sandbox Contaienr 也一般我们也成为Infrastructure Container，在 kubernetes 中，它还有一个大家很熟悉的名字 pause 容器。

下面以 containerd 以及 cni 提供的 example 为例来说明 cni 网络插件的工作机制。
```go
package main

import (
	"context"
	"fmt"
	"log"

	gocni "github.com/containerd/go-cni"
)

func main() {
	id := "example"
	netns := "/var/run/netns/example-ns-1"

	// CNI allows multiple CNI configurations and the network interface
	// will be named by eth0, eth1, ..., ethN.
	ifPrefixName := "eth"
	defaultIfName := "eth0"

	// Initializes library
	l, err := gocni.New(
		// one for loopback network interface
		gocni.WithMinNetworkCount(2),
		gocni.WithPluginConfDir("/etc/cni/net.d"),
		gocni.WithPluginDir([]string{"/opt/cni/bin"}),
		// Sets the prefix for network interfaces, eth by default
		gocni.WithInterfacePrefix(ifPrefixName))
	if err != nil {
		log.Fatalf("failed to initialize cni library: %v", err)
	}

	// Load the cni configuration
	if err := l.Load(gocni.WithLoNetwork, gocni.WithDefaultConf); err != nil {
		log.Fatalf("failed to load cni configuration: %v", err)
	}

	// Setup network for namespace.
	labels := map[string]string{
		"K8S_POD_NAMESPACE":          "namespace1",
		"K8S_POD_NAME":               "pod1",
		"K8S_POD_INFRA_CONTAINER_ID": id,
		// Plugin tolerates all Args embedded by unknown labels, like
		// K8S_POD_NAMESPACE/NAME/INFRA_CONTAINER_ID...
		"IgnoreUnknown": "1",
	}

	ctx := context.Background()

	// Teardown network
	defer func() {
		if err := l.Remove(ctx, id, netns, gocni.WithLabels(labels)); err != nil {
			log.Fatalf("failed to teardown network: %v", err)
		}
	}()

	// Setup network
	result, err := l.Setup(ctx, id, netns, gocni.WithLabels(labels))
	if err != nil {
		log.Fatalf("failed to setup network for namespace: %v", err)
	}

	// Get IP of the default interface
	IP := result.Interfaces[defaultIfName].IPConfigs[0].IP.String()
	fmt.Printf("IP of the default interface %s:%s", defaultIfName, IP)
}
```

首先看一下 containerd 启动是如何加载 cni 插件。

```go
// 代码路径：pkg/cri/server/service.go
func NewCRIService(config criconfig.Config, client *containerd.Client) (CRIService, error) {
	var err error
	labels := label.NewStore()
	c := &criService{
		config:                      config,
		client:                      client,
		os:                          osinterface.RealOS{},
		sandboxStore:                sandboxstore.NewStore(labels),
		containerStore:              containerstore.NewStore(labels),
		imageStore:                  imagestore.NewStore(client),
		snapshotStore:               snapshotstore.NewStore(),
		sandboxNameIndex:            registrar.NewRegistrar(),
		containerNameIndex:          registrar.NewRegistrar(),
		initialized:                 atomic.NewBool(false),
		netPlugin:                   make(map[string]cni.CNI),
		unpackDuplicationSuppressor: kmutex.New(),
	}

	if client.SnapshotService(c.config.ContainerdConfig.Snapshotter) == nil {
		return nil, fmt.Errorf("failed to find snapshotter %q", c.config.ContainerdConfig.Snapshotter)
	}

	c.imageFSPath = imageFSPath(config.ContainerdRootDir, config.ContainerdConfig.Snapshotter)
	logrus.Infof("Get image filesystem path %q", c.imageFSPath)

	if err := c.initPlatform(); err != nil {
		return nil, fmt.Errorf("initialize platform: %w", err)
	}

	c.streamServer, err = newStreamServer(c, config.StreamServerAddress, config.StreamServerPort, config.StreamIdleTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream server: %w", err)
	}

	c.eventMonitor = newEventMonitor(c)

	// cni 网络插件初始化
	c.cniNetConfMonitor = make(map[string]*cniNetConfSyncer)
	for name, i := range c.netPlugin {
		path := c.config.NetworkPluginConfDir
		if name != defaultNetworkPlugin {
			if rc, ok := c.config.Runtimes[name]; ok {
				path = rc.NetworkPluginConfDir
			}
		}
		if path != "" {
			m, err := newCNINetConfSyncer(path, i, c.cniLoadOptions())
			if err != nil {
				return nil, fmt.Errorf("failed to create cni conf monitor for %s: %w", name, err)
			}
			c.cniNetConfMonitor[name] = m
		}
	}

	c.baseOCISpecs, err = loadBaseOCISpecs(&config)
	if err != nil {
		return nil, err
	}

	return c, nil
}
```

在 containerd 中接收到从 kubelet 发送过来的容器运行指令后，containerd 将会创建并运行一个sanbox容器， cni 的创建容器网络的工作被放在了 sanbox 运行逻辑中，代码如下所示。
``` go
// pkg/cri/sbserver/sandbox_run.go
func (c *criService) RunPodSandbox(ctx context.Context, r *runtime.RunPodSandboxRequest) (_ *runtime.RunPodSandboxResponse, retErr error) {
	// ...

	if podNetwork {
		netStart := time.Now()
		// If it is not in host network namespace then create a namespace and set the sandbox
		// handle. NetNSPath in sandbox metadata and NetNS is non empty only for non host network
		// namespaces. If the pod is in host network namespace then both are empty and should not
		// be used.
		var netnsMountDir = "/var/run/netns"
		if c.config.NetNSMountsUnderStateDir {
			netnsMountDir = filepath.Join(c.config.StateDir, "netns")
		}
		sandbox.NetNS, err = netns.NewNetNS(netnsMountDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create network namespace for sandbox %q: %w", id, err)
		}
		sandbox.NetNSPath = sandbox.NetNS.GetPath()
		defer func() {
			if retErr != nil {
				deferCtx, deferCancel := ctrdutil.DeferContext()
				defer deferCancel()
				// Teardown network if an error is returned.
				if err := c.teardownPodNetwork(deferCtx, sandbox); err != nil {
					log.G(ctx).WithError(err).Errorf("Failed to destroy network for sandbox %q", id)
				}

				if err := sandbox.NetNS.Remove(); err != nil {
					log.G(ctx).WithError(err).Errorf("Failed to remove network namespace %s for sandbox %q", sandbox.NetNSPath, id)
				}
				sandbox.NetNSPath = ""
			}
		}()

		// 设置容器网络
		if err := c.setupPodNetwork(ctx, &sandbox); err != nil {
			return nil, fmt.Errorf("failed to setup network for sandbox %q: %w", id, err)
		}
		sandboxCreateNetworkTimer.UpdateSince(netStart)
	}

	// ...
}

// 设置容器网络
func (c *criService) setupPodNetwork(ctx context.Context, sandbox *sandboxstore.Sandbox) error {
	var (
		id        = sandbox.ID
		config    = sandbox.Config
		path      = sandbox.NetNSPath
		netPlugin = c.getNetworkPlugin(sandbox.RuntimeHandler)
	)
	if netPlugin == nil {
		return errors.New("cni config not initialized")
	}

	opts, err := cniNamespaceOpts(id, config)
	if err != nil {
		return fmt.Errorf("get cni namespace options: %w", err)
	}
	log.G(ctx).WithField("podsandboxid", id).Debugf("begin cni setup")

    // 调用插件
	result, err := netPlugin.Setup(ctx, id, path, opts...)
	if err != nil {
		return err
	}
	logDebugCNIResult(ctx, id, result)
	// Check if the default interface has IP config
	if configs, ok := result.Interfaces[defaultIfName]; ok && len(configs.IPConfigs) > 0 {
		sandbox.IP, sandbox.AdditionalIPs = selectPodIPs(ctx, configs.IPConfigs, c.config.IPPreference)
		sandbox.CNIResult = result
		return nil
	}
	return fmt.Errorf("failed to find network info for sandbox %q", id)
}
```