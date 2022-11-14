---
layout: single
title:  "Kubernetes简介"
date:   2022-11-2 11:00
author_profile: false
---
## 概述
Kubernetes 是一个可移植、可扩展的开源平台，用于管理容器化的工作负载和服务，可促进声明式配置和自动化。 Kubernetes 拥有一个庞大且快速增长的生态，其服务、支持和工具的使用范围相当广泛。

Kubernetes，又称为 k8s（首字母为 k、首字母与尾字母之间有 8 个字符、尾字母为 s，所以简称 k8s）或者简称为 "kube" ，是一种可自动实施 Linux 容器操作的开源平台。它可以帮助用户省去应用容器化过程的许多手动部署和扩展操作。也就是说，您可以将运行 Linux 容器的多组主机聚集在一起，由 Kubernetes 帮助您轻松高效地管理这些集群。而且，这些集群可跨公共云、私有云或混合云部署主机。因此，对于要求快速扩展的云原生应用而言（例如借助 Apache Kafka 进行的实时数据流处理），Kubernetes 是理想的托管平台。

## 起源
随着以 docker 代表的 linux 容器技术的应用场景越来越广，如何在大型的应用服务管理成百上千个容器成为了一个令人头痛的问题，docker 官方给出了docker compose + swarm 的方案来解决大规模容器部署的挑战。

Kubernetes 这个名字源于希腊语，意为“舵手”或“飞行员”。k8s 这个缩写是因为 k 和 s 之间有八个字符的关系。 Google 在 2014 年开源了 Kubernetes 项目。 Kubernetes 建立在 Google 大规模运行生产工作负载十几年经验的基础上， 结合了社区中最优秀的想法和实践。

### 容器编排
容器编排是指自动化容器的部署、管理、扩展和联网。容器编排可以帮助您在不同环境中部署相同的应用，而无需重新设计。利用编排来管理容器的生命周期也为将编排集成到 CI/CD 工作流中的 DevOps 团队提供了支持。与应用编程接口（API）和 DevOps 团队一样，容器化微服务也是云原生应用的重要基础。

### 为什么需要 Kubernetes？
真正的生产型应用会涉及多个容器。这些容器必须跨多个服务器主机进行部署。容器安全性需要多层部署，因此可能会比较复杂。但 Kubernetes 有助于解决这一问题。Kubernetes 可以提供所需的编排和管理功能，以便您针对这些工作负载大规模部署容器。借助 Kubernetes 编排功能，您可以构建跨多个容器的应用服务、跨集群调度、扩展这些容器，并长期持续管理这些容器的健康状况。有了 Kubernetes，您便可切实采取一些措施来提高 IT 安全性。

Kubernetes 为你提供：

#### 服务发现和负载均衡

Kubernetes 可以使用 DNS 名称或自己的 IP 地址来曝露容器。 如果进入容器的流量很大， Kubernetes 可以负载均衡并分配网络流量，从而使部署稳定。

#### 存储编排

Kubernetes 允许你自动挂载你选择的存储系统，例如本地存储、公共云提供商等。

#### 自动部署和回滚

你可以使用 Kubernetes 描述已部署容器的所需状态， 它可以以受控的速率将实际状态更改为期望状态。 例如，你可以自动化 Kubernetes 来为你的部署创建新容器， 删除现有容器并将它们的所有资源用于新容器。

#### 自动完成装箱计算

你为 Kubernetes 提供许多节点组成的集群，在这个集群上运行容器化的任务。 你告诉 Kubernetes 每个容器需要多少 CPU 和内存 (RAM)。 Kubernetes 可以将这些容器按实际情况调度到你的节点上，以最佳方式利用你的资源。

#### 自我修复

Kubernetes 将重新启动失败的容器、替换容器、杀死不响应用户定义的运行状况检查的容器， 并且在准备好服务之前不将其通告给客户端。

#### 密钥与配置管理

Kubernetes 允许你存储和管理敏感信息，例如密码、OAuth 令牌和 ssh 密钥。 你可以在不重建容器镜像的情况下部署和更新密钥和应用程序配置，也无需在堆栈配置中暴露密钥。 

### Kubernetes 不是什么？

Kubernetes 不是传统的、包罗万象的 PaaS（平台即服务）系统。 由于 Kubernetes 是在容器级别运行，而非在硬件级别，它提供了 PaaS 产品共有的一些普遍适用的功能， 例如部署、扩展、负载均衡，允许用户集成他们的日志记录、监控和警报方案。 但是，Kubernetes 不是单体式（monolithic）系统，那些默认解决方案都是可选、可插拔的。 Kubernetes 为构建开发人员平台提供了基础，但是在重要的地方保留了用户选择权，能有更高的灵活性。

Kubernetes：

1. 不限制支持的应用程序类型。 Kubernetes 旨在支持极其多种多样的工作负载，包括无状态、有状态和数据处理工作负载。 如果应用程序可以在容器中运行，那么它应该可以在 Kubernetes 上很好地运行。
不部署源代码，也不构建你的应用程序。 持续集成（CI）、交付和部署（CI/CD）工作流取决于组织的文化和偏好以及技术要求。
不提供应用程序级别的服务作为内置服务，例如中间件（例如消息中间件）、 数据处理框架（例如 Spark）、数据库（例如 MySQL）、缓存、集群存储系统 （例如 Ceph）。这样的组件可以在 Kubernetes 上运行，并且/或者可以由运行在 Kubernetes 上的应用程序通过可移植机制 （例如开放服务代理）来访问。
2. 不是日志记录、监视或警报的解决方案。 它集成了一些功能作为概念证明，并提供了收集和导出指标的机制。
3. 不提供也不要求配置用的语言、系统（例如 jsonnet），它提供了声明性 API， 该声明性 API 可以由任意形式的声明性规范所构成。
4. 不提供也不采用任何全面的机器配置、维护、管理或自我修复系统。

## Kubernetes 架构

![Kubernetes 架构图](  /assets/images/kubernetes/kubernetes.png)

### 控制平面组件（Control Plane Components）

控制平面组件会为集群做出全局决策，比如资源的调度。以及检测和响应集群事件，例如当不满足部署的 replicas 字段时， 要启动新的 pod）。

控制平面组件可以在集群中的任何节点上运行。 然而，为了简单起见，设置脚本通常会在同一个计算机上启动所有控制平面组件， 并且不会在此计算机上运行用户容器。 请参阅使用 kubeadm 构建高可用性集群 中关于跨多机器控制平面设置的示例。

控制平面包括组件: kube-apiserver, etcd, kube-scheduler, kube-controller-manager, kubelet。

### 工作平面组件（Work Plane Components）

工作平面组件运行在工作节点中, 提供 Kubernetes 中的 Pod 所需的运行环境。

工作平面组包括组件: kubelet, kube-proxy, 容器运行时。

## Kubernetes 组件

![Kubernetes 组件](  /assets/images/kubernetes/components-of-kubernetes.png)

#### kube-apiserver

API 服务器是 Kubernetes 控制平面的组件， 该组件负责公开了 Kubernetes API，负责处理接受请求的工作。 API 服务器是 Kubernetes 控制平面的前端。

Kubernetes API 服务器的主要实现是 kube-apiserver。 kube-apiserver 设计上考虑了水平扩缩，也就是说，它可通过部署多个实例来进行扩缩。 你可以运行 kube-apiserver 的多个实例，并在这些实例之间平衡流量。

#### etcd

一致且高度可用的键值存储，用作 Kubernetes 的所有集群数据的后台数据库。

如果你的 Kubernetes 集群使用 etcd 作为其后台数据库， 请确保你针对这些数据有一份 备份计划。

你可以在官方文档中找到有关 etcd 的深入知识。

#### kube-scheduler

kube-scheduler 是控制平面的组件， 负责监视新创建的、未指定运行节点（node）的 Pods， 并选择节点来让 Pod 在上面运行。

调度决策考虑的因素包括单个 Pod 及 Pods 集合的资源需求、软硬件及策略约束、 亲和性及反亲和性规范、数据位置、工作负载间的干扰及最后时限。

#### kube-controller-manager

kube-controller-manager 是控制平面的组件， 负责运行控制器进程。

从逻辑上讲， 每个控制器都是一个单独的进程， 但是为了降低复杂性，它们都被编译到同一个可执行文件，并在同一个进程中运行。

这些控制器包括：

节点控制器（Node Controller）：负责在节点出现故障时进行通知和响应
任务控制器（Job Controller）：监测代表一次性任务的 Job 对象，然后创建 Pods 来运行这些任务直至完成
端点控制器（Endpoints Controller）：填充端点（Endpoints）对象（即加入 Service 与 Pod）
服务帐户和令牌控制器（Service Account & Token Controllers）：为新的命名空间创建默认帐户和 API 访问令牌
cloud-controller-manager
一个 Kubernetes 控制平面组件， 嵌入了特定于云平台的控制逻辑。 云控制器管理器（Cloud Controller Manager）允许你将你的集群连接到云提供商的 API 之上， 并将与该云平台交互的组件同与你的集群交互的组件分离开来。
cloud-controller-manager 仅运行特定于云平台的控制器。 因此如果你在自己的环境中运行 Kubernetes，或者在本地计算机中运行学习环境， 所部署的集群不需要有云控制器管理器。

与 kube-controller-manager 类似，cloud-controller-manager 将若干逻辑上独立的控制回路组合到同一个可执行文件中， 供你以同一进程的方式运行。 你可以对其执行水平扩容（运行不止一个副本）以提升性能或者增强容错能力。

下面的控制器都包含对云平台驱动的依赖：

节点控制器（Node Controller）：用于在节点终止响应后检查云提供商以确定节点是否已被删除
路由控制器（Route Controller）：用于在底层云基础架构中设置路由
服务控制器（Service Controller）：用于创建、更新和删除云提供商负载均衡器

#### kubelet

kubelet 会在集群中每个节点（node）上运行。 它保证容器（containers）都运行在 Pod 中。

kubelet 接收一组通过各类机制提供给它的 PodSpecs， 确保这些 PodSpecs 中描述的容器处于运行状态且健康。 kubelet 不会管理不是由 Kubernetes 创建的容器。

#### kube-proxy

kube-proxy 是集群中每个节点（node）所上运行的网络代理， 实现 Kubernetes 服务（Service） 概念的一部分。

kube-proxy 维护节点上的一些网络规则， 这些网络规则会允许从集群内部或外部的网络会话与 Pod 进行网络通信。

如果操作系统提供了可用的数据包过滤层，则 kube-proxy 会通过它来实现网络规则。 否则，kube-proxy 仅做流量转发。

#### 容器运行时（Container Runtime）

容器运行环境是负责运行容器的软件。

Kubernetes 支持许多容器运行环境，例如 containerd、 CRI-O 以及 Kubernetes CRI (容器运行环境接口) 的其他任何实现。

## Kubernetes 相关术语

和其它技术一样，Kubernetes 也会采用一些专用的词汇，这可能会对初学者理解和掌握这项技术造成一定的障碍。为了帮助您了解 Kubernetes，我们在下面来解释一些常用术语。

#### 管理节点（Master）： 用于控制 Kubernetes 节点的计算机。所有任务分配都来自于此。

#### 工作节点（WorkNode）：负责执行请求和所分配任务的计算机。由 Kubernetes 主机负责对节点进行控制。

##### 容器集（Pods）：被部署在单个节点上的，且包含一个或多个容器的容器组。同一容器集中的所有容器共享同一个 IP 地址、IPC、主机名称及其它资源。容器集会将网络和存储从底层容器中抽象出来。这样，您就能更加轻松地在集群中移动容器。

##### 复制控制器（Replication controller）：用于控制应在集群某处运行的完全相同的容器集副本数量。

##### 服务（Service）：将工作内容与容器集分离。Kubernetes 服务代理会自动将服务请求分发到正确的容器集——无论这个容器集会移到集群中的哪个位置，甚至可以被替换掉。

##### Kubelet：运行在节点上的服务，可读取容器清单（container manifest），确保指定的容器启动并运行。

##### kubectl： Kubernetes 的命令行配置工具。

##### kubeadm:  快速构建 Kubernetes 集群的官方工具。

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io

## 参考
- [1] [kubenretes官方文档](https://kubernetes.io/zh-cn/docs/concepts/overview/)