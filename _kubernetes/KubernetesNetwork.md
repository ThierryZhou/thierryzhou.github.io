---
title: "Kubernetes 网络详解"
tag: kubernetes
excerpt: 在 Kubernetes 中，调度 (scheduling) 指的是确保 Pod 匹配到合适的节点， 以便 kubelet 能够运行它们。 调度的工作由调度器和控制器协调完成。
---

## Kubernetes网络模型

在Kubernetes网络中存在两种IP（Pod IP和Service Cluster IP），Pod IP 地址是实际存在于某个网卡(可以是虚拟设备)上的，Service Cluster IP它是一个虚拟IP，是由kube-proxy使用Iptables规则重新定向到其本地端口，再均衡到后端Pod的。下面讲讲Kubernetes Pod网络设计模型：

1、基本原则：

每个Pod都拥有一个独立的IP地址（IPper Pod），而且假定所有的pod都在一个可以直接连通的、扁平的网络空间中。

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