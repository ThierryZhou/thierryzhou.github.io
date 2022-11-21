## Kubernetes 架构

![Kubernetes 架构图](  /assets/images/kubernetes/kubernetes.png)

## 控制平面组件（Control Plane Components）

控制平面组件会为集群做出全局决策，比如资源的调度。以及检测和响应集群事件，例如当不满足部署的 replicas 字段时， 要启动新的 pod）。

控制平面组件可以在集群中的任何节点上运行。 然而，为了简单起见，设置脚本通常会在同一个计算机上启动所有控制平面组件， 并且不会在此计算机上运行用户容器。 请参阅使用 kubeadm 构建高可用性集群 中关于跨多机器控制平面设置的示例。

控制平面包括组件: kube-apiserver, etcd, kube-scheduler, kube-controller-manager, kubelet。

### Etcd简介
etcd 是一个分布式键值对存储系统，由coreos 开发，内部采用 raft 协议作为一致性算法，用于可靠、快速地保存关键数据，并提供访问。通过分布式锁、leader选举和写屏障(write barriers)，来实现可靠的分布式协作。etcd集群是为高可用、持久化数据存储和检索而准备。

etcd 以一致和容错的方式存储元数据。分布式系统使用 etcd 作为一致性键值存储系统，用于配置管理、服务发现和协调分布式工作。使用 etcd 的通用分布式模式包括领导选举、分布式锁和监控机器活动。

虽然 etcd 也支持单点部署，但是在生产环境中推荐集群方式部署。由于etcd内部使用投票机制，一般 etcd 节点数会选择 3、5、7等奇数。etcd 会保证所有的节点都会保存数据，并保证数据的一致性和正确性。

etcd 目前默认使用 2379 端口提供 HTTP API 服务，2380 端口和 peer 通信（这两个端口已经被 IANA 官方预留给 etcd）。在之前的版本中，可能会分别使用 4001 和 7001，在使用的过程中需要注意这个区别。

#### Etcd概念词汇
Raft：etcd所采用的保证分布式系统强一致性的算法。

Node：一个Raft状态机实例。

Member： 一个etcd实例。它管理着一个Node，并且可以为客户端请求提供服务。

Cluster：由多个Member构成、可以协同工作的etcd集群。

Peer：对同一个etcd集群中另外一个Member的称呼。

Client： 向etcd集群发送HTTP请求的客户端。

WAL：预写式日志，etcd用于持久化存储的日志格式。

snapshot：etcd防止WAL文件过多而设置的快照，存储etcd数据状态。

Proxy：etcd的一种模式，为etcd集群提供反向代理服务。

Leader：Raft算法中，通过竞选而产生的、处理所有数据提交的节点。

Follower：竞选失败的节点作为Raft中的从属节点，为算法提供强一致性保证。

Candidate：当Follower超过一定时间接收不到Leader的心跳时转变为Candidate开始竞选。

Term：某个节点成为Leader到下一次竞选时间，称为一个Term。

Index：数据项编号。Raft中通过Term和Index来定位数据。

#### etcd优点
etcd作为一个受到ZooKeeper与doozer启发而催生的项目，除了拥有与之类似的功能外，更专注于以下四点：

简单：安装配置简单，而且提供了 HTTP API 进行交互，使用也很简单
安全：支持 SSL 证书验证
快速：根据官方提供的 benchmark 数据，单实例支持每秒 2k+ 读操作
可靠：采用 raft 算法，实现分布式系统数据的可用性和一致性

#### 工作原理
etcd的工作原理图如下所示：

![etcd-stack](/assets/images/kubernetes/etcd-stack.png)

从etcd的架构图中我们可以看到，etcd主要分为四个部分：

第1部分是HTTP Server： 用于处理用户发送的API请求，以及其它etcd节点的同步与心跳信息请求。
第2部分是Store：用于处理etcd支持的各类功能的事务，包括数据索引、节点状态变更、监控与反馈、事件处理与执行等等，是etcd对用户提供的大多数API功能的具体实现。
第3部分是Raft：Raft强一致性算法的具体实现，是etcd的核心。
第4部分是WAL：Write Ahead Log（预写式日志），是etcd的数据存储方式。除了在内存中存有所有数据的状态以及节点的索引以外，etcd就通过WAL进行持久化存储。在WAL中，所有的数据提交前都会事先记录日志。Snapshot是为了防止数据过多而进行的状态快照；Entry表示存储的具体日志内容。

通常，一个用户的请求发送过来，会经由HTTP Server转发给Store，以进行具体的事务处理。如果涉及到节点修改，则交给Raft模块进行状态变更、日志记录；然后，再同步给别的etcd节点，以确认数据提交；最后，进行数据提交，再次同步。

### ApiServer 简介


### Kube-Controller-Manager 简介


### Kube-Scheduler 简介

