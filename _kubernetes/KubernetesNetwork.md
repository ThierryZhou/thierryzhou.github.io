---
title: "Kubernetes 网络详解"
tag: kubernetes
excerpt: 在 Kubernetes 中，调度 (scheduling) 指的是确保 Pod 匹配到合适的节点， 以便 kubelet 能够运行它们。 调度的工作由调度器和控制器协调完成。
---

集群网络系统是 Kubernetes 的核心部分，但是想要准确了解它的工作原理可是个不小的挑战。 下面列出的是网络系统的的四个主要问题：

1. 高度耦合的容器间通信：这个已经被 Pod 和 localhost 通信解决了。
2. Pod 间通信：这是本文档讲述的重点。
3. Pod 与 Service 间通信：涵盖在 Service 中。
4. 外部与 Service 间通信：也涵盖在 Service 中。

Kubernetes 的宗旨就是在应用之间共享机器。 通常来说，共享机器需要两个应用之间不能使用相同的端口，但是在多个应用开发者之间去大规模地协调端口是件很困难的事情，尤其是还要让用户暴露在他们控制范围之外的集群级别的问题上。

为每个应用都需要设置一个端口的参数，而 API 服务器还需要知道如何将动态端口数值插入到配置模块中，服务也需要知道如何找到对方等等。这无疑会为系统带来很多复杂度，Kubernetes 选择了增加抽象层的方式去分解系统的复杂度。

## Kubernetes网络模型

集群中每一个 Pod 都会获得自己的、 独一无二的 IP 地址， 这就意味着你不需要显式地在 Pod 之间创建链接，你几乎不需要处理容器端口到主机端口之间的映射。这将形成一个干净的、向后兼容的模型；在这个模型里，从端口分配、命名、服务发现、 负载均衡、 应用配置和迁移的角度来看，Pod 可以被视作虚拟机或者物理主机。

Kubernetes 强制要求所有网络设施都满足以下基本要求（从而排除了有意隔离网络的策略）：

1. Pod 能够与所有其他节点上的 Pod 通信， 且不需要网络地址转译(NAT)
2. 节点上的守护进程(kubelet)可以和节点上的所有 Pod 通信

说明：对于支持在主机网络中运行 Pod 的平台， 当 Pod 挂接到节点的宿主网络上时，它们仍可以不通过 NAT 和所有节点上的 Pod 通信。

这个模型不仅不复杂，而且还和 Kubernetes 的实现从虚拟机向容器平滑迁移的初衷相符， 如果你的任务开始是在虚拟机中运行的，你的虚拟机有一个 IP， 可以和项目中其他虚拟机通信。这里的模型是基本相同的。

Kubernetes 的 IP 地址存在于 Pod 范围内 —— 容器共享它们的网络命名空间 —— 包括它们的 IP 地址和 MAC 地址。这就意味着 Pod 内的容器都可以通过 localhost 到达对方端口。 这也意味着 Pod 内的容器需要相互协调端口的使用，但是这和虚拟机中的进程似乎没有什么不同， 这也被称为“一个 Pod 一个 IP”模型。

如何实现以上需求是所使用的特定容器运行时的细节。

也可以在 Node 本身请求端口，并用这类端口转发到你的 Pod（称之为主机端口）， 但这是一个很特殊的操作。转发方式如何实现也是容器运行时的细节。 Pod 自己并不知道这些主机端口的存在。

Kubernetes 网络解决四方面的问题：

1. 一个 Pod 中的容器之间通过本地回路（loopback）通信。
2. 集群网络在不同 Pod 之间提供通信。
3. Service 资源允许你 向外暴露 Pod 中运行的应用， 以支持来自于集群外部的访问。
4. Ingress 提供专门用于暴露 HTTP 应用程序、网站和 API 的额外功能。
你也可以使用 Service 来发布仅供集群内部使用的服务。
集群网络解释了如何为集群设置网络， 还概述了所涉及的技术。

在Kubernetes网络中存在两种IP（Pod IP和Service Cluster IP），Pod IP 地址是实际存在于某个网卡(可以是虚拟设备)上的，Service Cluster IP它是一个虚拟IP，是由kube-proxy使用Iptables规则重新定向到其本地端口，再均衡到后端Pod的。下面讲讲Kubernetes Pod网络设计模型：

1、基本原则：

每个Pod都拥有一个独立的IP地址（IP per Pod），而且假定所有的pod都在一个可以直接连通的、扁平的网络空间中。

2、设计原因：

用户不需要额外考虑如何建立Pod之间的连接，也不需要考虑将容器端口映射到主机端口等问题。

3、网络要求：

所有的容器都可以在不用NAT的方式下同别的容器通讯；所有节点都可在不用NAT的方式下同所有容器通讯；容器的地址和别人看到的地址是同一个地址。

## Kubernetes网络基础组件

Linux网络基础组件解释：

1. 网络的命名空间：Linux在网络栈中引入网络命名空间，将独立的网络协议栈隔离到不同的命令空间中，彼此间无法通信；docker利用这一特性，实现不容器间的网络隔离。

2. Veth设备对：Veth设备对的引入是为了实现在不同网络命名空间的通信。

3. Iptables/Netfilter：Netfilter负责在内核中执行各种挂接的规则(过滤、修改、丢弃等)，运行在内核 模式中；Iptables模式是在用户模式下运行的进程，负责协助维护内核中Netfilter的各种规则表；通过二者的配合来实现整个Linux网络协议栈中灵活的数据包处理机制。

4. 网桥：网桥是一个二层网络设备,通过网桥可以将linux支持的不同的端口连接起来,并实现类似交换机那样的多对多的通信。

5. 路由：Linux系统包含一个完整的路由功能，当IP层在处理数据发送或转发的时候，会使用路由表来决定发往哪里。

## Kubernetes 网络上层组件

1、容器间通信：

同一个Pod的容器共享同一个网络命名空间，它们之间的访问可以用localhost地址 + 容器端口就可以访问。

2、同一Node中Pod间通信：

同一Node中Pod的默认路由都是docker0的地址，由于它们关联在同一个docker0网桥上，地址网段相同，所有它们之间应当是能直接通信的。

3、不同Node中Pod间通信：

不同Node中Pod间通信要满足2个条件： Pod的IP不能冲突； 将Pod的IP和所在的Node的IP关联起来，通过这个关联让Pod可以互相访问。


4、Service介绍：

Service是一组Pod的服务抽象，相当于一组Pod的LB，负责将请求分发给对应的

Pod；Service会为这个LB提供一个IP，一般称为ClusterIP。

5、Kube-proxy介绍：

Kube-proxy是一个简单的网络代理和负载均衡器，它的作用主要是负责Service的实现，具体来说，就是实现了内部从Pod到Service和外部的从NodePort向Service的访问。

6、Kube-dns介绍

Kube-dns用来为kubernetes service分配子域名，在集群中可以通过名称访问service；通常kube-dns会为service赋予一个名为“service名称.namespace.svc.cluster.local”的A记录，用来解析service的clusterip。

## Kubernetes 网络插件

Kubernetes 1.25 支持用于集群联网的容器网络接口 (CNI) 插件。 你必须使用和你的集群相兼容并且满足你的需求的 CNI 插件。 在更广泛的 Kubernetes 生态系统中你可以使用不同的插件（开源和闭源）。

要实现 Kubernetes 网络模型，你必须在集群中部署一套 CNI 插件系统。下面分析一下两个常见的 CNI 插件：Fannel 和 Calico

### Flannel 简介

Flannel是一种基于overlay网络的跨主机容器网络解决方案，也就是将TCP数据包封装在另一种网络包里面进行路由转发和通信。

Flannel配合etcd可以实现不同宿主机上的docker容器内网IP的互通。

Flannel是CoreOS开发,专门用于docker多机互联的一个工具,让集群中的不同节点主机创建的容器都具有全集群唯一的虚拟ip地址。

Flannel使用go语言编写。

#### 实现原理

Flannel为每个host分配一个subnet，容器从这个subnet中分配IP，这些IP可以在host间路由，容器间无需使用nat和端口映射即可实现跨主机通信

每个subnet都是从一个更大的IP池中划分的，flannel会在每个主机上运行一个叫flanneld的agent，其职责就是从池子中分配subnet

Flannel使用etcd存放网络配置、已分配 的subnet、host的IP等信息

Flannel数据包在主机间转发是由backend实现的，目前已经支持UDP、VxLAN、host-gw、AWS VPC和GCE路由等多种backend




### Caclico 简介
