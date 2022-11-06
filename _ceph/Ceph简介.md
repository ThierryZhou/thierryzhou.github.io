---
title: Ceph简介
---
### 什么是分布式存储
与集中式存储相反，分布式存储通常采用存储单元集群的形式，并具有在集群节点之间进行数据同步和协调的机制。分布式存储最初是由Google提出的，其目的是通过廉价服务器解决大规模，高并发情况下的Web访问问题。

分布式存储具有几个优点：
1. 可扩展性-支持通过在系统中添加或删除存储单元来水平扩展存储系统。
2. 冗余-在多台服务器之间存储相同数据的复制，以实现高可用性， 备份和灾难恢复目的。
3. 节省成本 -可以使用更便宜的商品服务器以低成本存储大量 数据。
4. 性能-在某些情况下，性能比单个服务器更好，例如，它可以将数据存储在离其使用者更近的位置，或者允许大规模并行访问大文件。

### 起源
Ceph项目最早起源于Sage就读博士期间发表的，并随后贡献给开源社区。在经过了数年的发展之后，目前已得到众多云计算厂商的支持并被广泛应用。RedHat及OpenStack都可与Ceph整合以支持虚拟机镜像的后端存储。但是在2014年OpenStack火爆的时候、Ceph并不被很多人所接受。当时Ceph并不稳定（Ceph发布的第四个版本 Dumpling v0.67），而且架构新颖，复杂，当时人们对Ceph在生产落地如何保障数据的安全，数据的一致性存在怀疑。

随着OpenStack的快速发展，越来越多的人使用Ceph作为OpenStack的底层共享存储，Ceph在中国的社区也蓬勃发展起来。近两年OpenStack火爆度不及当年，借助于云原生尤其是Kubernetes技术的发展，作为底层存储的基石，Ceph再次发力，为Kubernets有状态化业务提供了存储机制的实现。
## 优点
1. 高性能  
a. 摒弃了传统的集中式存储元数据寻址的方案，采用CRUSH算法，数据分布均衡，并行度高。  
b. 考虑了容灾域的隔离，能够实现各类负载的副本放置规则，例如跨机房、机架感知等。  
c. 能够支持上千个存储节点的规模，支持TB到PB级的数据。  
2. 高可用性  
a. 副本数可以灵活控制。  
b. 支持故障域分隔，数据强一致性。  
c. 多种故障场景自动进行修复自愈。  
d. 没有单点故障，自动管理。  
3. 高可扩展性  
a. 去中心化。  
b. 扩展灵活。  
c. 随着节点增加而线性增长。  
4. 特性丰富  
a. 支持三种存储接口：块存储、文件存储、对象存储。  
b. 支持自定义接口，支持多种语言驱动。

## 服务架构
一个 Ceph 存储集群至少需要一个 Ceph Monitor（监视器）、Ceph Manager（管理器） 和 Ceph OSD（对象存储守护进程）。

![服务架构](/assets/ceph/ceph-arch.png)
### Monitors
Ceph Monitor (ceph-mon) 通过维护包括监视器表(MonMap)、管理表(MGRMap)、OSD表(OSDMap)等组件状态表的保障集群正常运行。ceph-osd 相互之间协调工作时，需要从 ceph-mon 中获取这些表的信息。ceph-mon 还负责管理 ceph-osd 和客户端之间的身份验证。一个Ceph集群为了保证冗余和高可用性通常需要至少三个监视器，它们之间通过Paxos同步数据。

### Managers
Ceph Manager (ceph-mgr) 负责跟踪运行时指标和 Ceph 集群的当前状态，包括存储利用率、当前性能指标、集群报警和系统负载等。ceph-mon 和 ceph-mgr 协调配合共同维持集群稳定。高可用性通常需要至少两个管理器。
### OSDS
Ceph OSD（ceph-osd）全称是Object Storage Device，负责包括处理数据复制、恢复、重新平衡在内的实际数据存储工作，并且一个 OSD 检查可以通过检查其他 OSD 的心跳的方式将其他 OSD 的异常状态上报给 MON。一个Ceph集群一般都有很多个OSD。

## 逻辑架构

Ceph 将数据作为对象存储在逻辑存储池中。使用 CRUSH算法，Ceph 计算出哪个归置组应该包含该对象，并进一步计算出哪个 Ceph OSD Daemon 应该存储该归置组。CRUSH 算法使 Ceph 存储集群能够动态扩展、重新平衡和恢复。

RADOS： 由自我修复、自我管理、智能存储节点组成的可靠、自主、分布式对象存储。

LIBRADOS： Ceph 供的外部访问的对象存储 API，允许客户端通过此 API 访问Ceph集群完成文件的读写工作，支持 C、C++、Java、Python、Ruby 和 PHP 等多种语言。

![逻辑架构](/assets/ceph/ceph-stack.png)

用户可以基于自己的业务需要直接 LIBRADOS API 的基础上，开发自己需要的存储业务。社区在 LIBRADOS API 开发了三种成熟的存储产品：块存储、分布式文件存储系统、对象存储。

### RADOSGW(Rados Gateway)
基于LIBRADOS API构建的兼容了 s3 和 Swift 协议的 RESTful 网关。
![rgw](/assets/ceph/rgw-stack.png)

### RBD(Rados Block Device)
基于LIBRADOS API构建的，使用了Linux内核client和QEMU/KVM驱动程序的分布式块存储设备。
![rbd](/assets/ceph/rbd-stack.png)

### CEPHFS(Ceph FileSystem)
基于LIBRADOS API构建的，符合POSIX标准的分布式文件系统。Ceph 文件系统 需要至少指定一个metadata存储池和一个data存储池，并且Ceph 文件系统 需要集群至少有一个Metadata服务。
![Cephfs](/assets/ceph/cephfs-stack.png)

### Ceph逻辑组件
#### Object
Ceph 最底层的存储单元是 Object 对象，每个 Object 包含元数据和原始数据。

#### PG
PG 全称 Placement Grouops，是一个逻辑的概念，一个 PG 包含多个 OSD。引入 PG 这一层其实是为了更好的分配数据和定位数据。

##### CRUSH
CRUSH 是 Ceph 使用的数据分布算法，类似一致性哈希，让数据分配到预期的地方。

## 文件存储
不管是来自 Ceph 块设备、 Ceph 对象存储、 Ceph 文件系统、还是基于 librados 的自定义存储，将数据存储到 Ceph 集群上的步骤大致相同，大概可以归纳为Ceph客户端将数据存放在存储对象中，存储对象经过Ceph集群处理被发送给了 OSD ，由 OSD 将对象持久化到磁盘上。
![File](/assets/ceph/ceph-file.png)

Ceph OSD 在扁平的命名空间内把所有数据存储为对象（也就是没有目录层次）。对象包含一个标识符、二进制数据、和由名字/值对组成的元数据，元数据语义完全取决于 Ceph 客户端。例如， CephFS 用元数据存储文件属性，如文件所有者、创建日期、最后修改日期等等。

![Binary](/assets/ceph/ceph-binary.png)

## IO流程
![IO](/assets/ceph/ceph-osd-io.png)
1. client 创建 cluster handler。  
2. client 读取配置文件。  
3. client 连接上 monitor，获取集群 map 信息。  
4. client 读写 io 根据 crshmap 算法请求对应的主 osd 数据节点。  
5. 主 osd 数据节点同时写入另外两个副本节点数据。  
6. 等待主节点以及另外两个副本节点写完数据状态。  
7. 主节点及副本节点写入状态都成功后，返回给 client，io 写入完成。

![Logical](/assets/ceph/ceph-logical-io.png)
1. File 用户需要读写的文件。File->Object 映射：  
a. ino (File 的元数据，File 的唯一 id)。  
b. ono(File 切分产生的某个 object 的序号，默认以 4M 切分一个块大小)。  
c. oid(object id: ino + ono)。  
2. Object 是 RADOS 需要的对象。Ceph 指定一个静态 hash 函数计算 oid 的值，将 oid 映射成一个近似均匀分布的伪随机值，然后和 mask 按位相与，得到 pgid。Object->PG 映射：  
a)  hash(oid) & mask-> pgid 。  
b)  mask = PG 总数 m(m 为 2 的整数幂)-1 。  
3. PG(Placement Group),用途是对 object 的存储进行组织和位置映射, (类似于 redis cluster 里面的 slot 的概念) 一个 PG 里面会有很多 object。采用 CRUSH 算法，将 pgid 代入其中，然后得到一组 OSD。PG->OSD 映射：  
a)  CRUSH(pgid)->(osd1,osd2,osd3) 。

### Ceph编排工具
Ceph社区开发了多种编排工具，方便你快速构建一个Ceph集群。
如果你想在物理机上以传统后台服务的方式运行你的集群，可以使用基于ansible框架开发的ceph-ansible。  
https://docs.ceph.com/projects/ceph-ansible/en/latest/index.html

如果你希望你的集群运行在物理机上的docker容器中，可以使用cephadm工具。  
https://docs.ceph.com/en/quincy/cephadm/#cephadm

如果你希望你的集群运行在Kubernetes中，运行在云服务器上，可以使用rook-ceph。  
https://rook.io/docs/rook/v1.10/Getting-Started/intro/

## 参考
- [1] [Ceph官方文档](https://docs.ceph.com/)  
- [2] [分布式存储 Ceph 介绍及原理架构分享](https://www.infoq.cn/article/brjtisyrudhgec4odexh)
