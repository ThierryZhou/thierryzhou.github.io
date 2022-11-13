---
title: Cep PG 和 OSD 集群状态分析
tag: ceph
---

## 前言

作为一个成熟、可靠的分布式存储框架，Ceph集群中的各个组件都具备很强的自运维能力，这样的能力都是依托于 Ceph 优秀的故障检测机制。这篇文章主要分析一下集群状态的变迁。

## Ceph OSD 状态分析

up、down代表OSD临时故障或下电，不会发起数据恢复；in、out代表被踢出集群，集群发起数据恢复。OSD 的状态通过心跳检测的方式被集群确认，即拥有心跳的 OSD 被标记为 up，心跳停止的 OSD 则被标记为 down。当 OSD 被标记为 down的时间超过的阈值（默认为600秒）时，OSD 被标记为 out。

|---------------------+----------------------------------|
|          状态        |                描述              |
|:-------------------:|:---------------------------------|
|          in         |           OSD在Ceph集群内         |
|          out        |           OSD在Ceph集群外         |
|          up         |         OSD(心跳)活着且在运行      |
|          down       |          OSD 挂了且不再运行        |
|---------------------+----------------------------------|

## Ceph OSD 心跳检测

![心跳检测](/assets/images/ceph/ceph-heartbeat.webp)

那么 OSD 是如何检测心跳的呢？
OSD 上电的过程中有一个非常关键的步骤是创建四个用于心跳通信的async messenger。
```cpp

```

所有类型的故障会记录再osdmap后报告到Monitor，然后扩散至集群，其他OSD收到消息后采取对应的措施。

Monitor通过一下三种方式检测OSD故障（下电）：
1. OSD自主上报状态，优雅下电，Monitor 将 OSD 标记为 down。
2. OSD检测到伙伴OSD返回ECONNREFUSED错误，则设置Force/ Immediate标识，向Monitor上报错误。当
该 OSD 积累的有效错误投票的票数达到阈值（默认2），Monitor 将 OSD 标记为 down，投票采用少数服从多数的方式，并且来自某个最小故障域的主机所有OSD针对候选OSD的投票算1票。
3. Monitor检测到与OSD之间心跳超时，失联的 OSD 被标记为 down。
每个OSD周期性（默认300秒）的向Monitor发送Beacon消息证明自己存活，如果Monitor一段时间（默认900秒）没收到OSD的Beacon，就标记OSD为down。OSDdown后超过600S，会被标记为out（Monitor通过 一个名为 mon_ osd_ down_ out_ subtree_ limit的配置项来限制自动数据迁移的粒度，例如设置为主机，则 当某个主机上的OSD全部宕掉时，这些OSD不再会被自动标记为Out，也就无法自动进行数据迁移，从而避免 数据迁移风暴）



#### OSD 如何选择伙伴 OSD？

1. 选择与当前 OSD 所在处的 PG 的 OSD 表中，其他处于 Up + Activing 的 OSD

2. 选择在编号上与当前 OSD 临近(前一个以及后一个)处于 Up + Activing 的 OSD

3. 如果 OSD 的心跳伙伴 OSD 个数小于预期（一般为10个），则依次原则在编号上与当前 OSD 临近且处于 Up + Activing 的 OSD，直到编号的最小(最大)值。

#### 客户端如何感知 OSD 状态变化？

客户端读写前会先从Monitor中获取到 PGMap 以及 OSDMap，PGMap 根据 version 信息判断 Map 是否过期如果过期则重启更新 Map，OSDMap 则是根据 epoch 信息来判断。

## Ceph 心跳机制

心跳是用于节点间检测对方是否故障的，以便及时发现故障节点进入相应的故障处理流程。

问题：



a. 故障检测时间和心跳报文带来的负载之间做权衡。



b. 心跳频率太高则过多的心跳报文会影响系统性能。



c. 心跳频率过低则会延长发现故障节点的时间，从而影响系统的可用性。



故障检测策略应该能够做到：



及时：节点发生异常如宕机或网络中断时，集群可以在可接受的时间范围内感知。



适当的压力：包括对节点的压力，和对网络的压力。



容忍网络抖动：网络偶尔延迟。



扩散机制：节点存活状态改变导致的元信息变化需要通过某种机制扩散到整个集群。


OSD 节点会监听 public、cluster、front 和 back 四个端口



public 端口：监听来自 Monitor 和 Client 的连接。

cluster 端口：监听来自 OSD Peer 的连接。

front 端口：供客户端连接集群使用的网卡, 这里临时给集群内部之间进行心跳。

back 端口：供客集群内部使用的网卡。集群内部之间进行心跳。

hbclient：发送 ping 心跳的 messenger。



▍3.3 Ceph OSD 之间相互心跳检测


步骤：



a. 同一个 PG 内 OSD 互相心跳，他们互相发送 PING/PONG 信息。



b. 每隔 6s 检测一次(实际会在这个基础上加一个随机时间来避免峰值)。



c. 20s 没有检测到心跳回复，加入 failure 队列。


OSD 报告给 Monitor：



a. OSD 有事件发生时（比如故障、PG 变更）。

b. 自身启动 5 秒内。

c. OSD 周期性的上报给 Monito

d. OSD 检查 failure_queue 中的伙伴 OSD 失败信息。

e. 向 Monitor 发送失效报告，并将失败信息加入 failure_pending 队列，然后将其从 failure_queue 移除。

f. 收到来自 failure_queue 或者 failure_pending 中的 OSD 的心跳时，将其从两个队列中移除，并告知 Monitor 取消之前的失效报告。

g. 当发生与 Monitor 网络重连时，会将 failure_pending 中的错误报告加回到 failure_queue 中，并再次发送给 Monitor。

h. Monitor 统计下线 OSD

i. Monitor 收集来自 OSD 的伙伴失效报告。

j. 当错误报告指向的 OSD 失效超过一定阈值，且有足够多的 OSD 报告其失效时，将该 OSD 下线。



Ceph 心跳检测总结
Ceph 通过伙伴 OSD 汇报失效节点和 Monitor 统计来自 OSD 的心跳两种方式判定 OSD 节点失效。



及时：



伙伴 OSD 可以在秒级发现节点失效并汇报 Monitor，并在几分钟内由 Monitor 将失效 OSD 下线。



适当的压力：



由于有伙伴 OSD 汇报机制，Monitor 与 OSD 之间的心跳统计更像是一种保险措施，因此 OSD 向 Monitor 发送心跳的间隔可以长达 600 秒，Monitor 的检测阈值也可以长达 900 秒。Ceph 实际上是将故障检测过程中中心节点的压力分散到所有的 OSD 上，以此提高中心节点 Monitor 的可靠性，进而提高整个集群的可扩展性。



容忍网络抖动：



Monitor 收到 OSD 对其伙伴 OSD 的汇报后，并没有马上将目标 OSD 下线，而是周期性的等待几个条件：



目标 OSD 的失效时间大于通过固定量 osd_heartbeat_grace 和历史网络条件动态确定的阈值。

来自不同主机的汇报达到 mon_osd_min_down_reporters。

满足前两个条件前失效汇报没有被源 OSD 取消。



扩散：



作为中心节点的 Monitor 并没有在更新 OSDMap 后尝试广播通知所有的 OSD 和 Client，而是惰性的等待 OSD 和 Client 来获取。以此来减少 Monitor 压力并简化交互逻辑。

### Ceph PG 状态分析
正常的PG状态是 100%的active + clean， 这表示所有的PG是可访问的，所有副本都对全部PG都可用。
如果Ceph也报告PG的其他的警告或者错误状态。
active + clean是PG的健康状态，然而PG也会生病,有的是小感冒，有的则可能是一级伤残，下面就是集群进入恢复状态时的一个截图，这里面聚集了各种老弱病残，现在就来分析下每种病症的原因:

Degraded

降级就是在发生了一些故障比如OSD挂掉之后，ceph将这个OSD上的所有PG标记为degraded，但是此时的集群还是可以正常读写数据的，降级的PG只是相当于小感冒而已，并不是严重的问题，而另一个词undersized,我的理解就是当前存活的PG 0.44数为2，小于副本数3，将其做此标记，也不是严重的问题。

Peered

peered，我们这里可以将它理解成它在等待其他兄弟姐妹上线，这里也就是osd.4和osd.7的任意一个上线就可以去除这个状态了，处于peered状态的PG是不能响应外部的请求的，外部就会觉得卡卡的。

Remapped
ceph强大的自我恢复能力，是我们选择它的一个重要原因，在上面的试验中，我们关闭了两个OSD，但是至少还有一个PG 0.44存活在osd.0上，如果那两个盘真的坏了，ceph还是可以将这份仅存的数据恢复到别的OSD上的。

在OSD挂掉5min(default 300s)之后，这个OSD会被标记为out状态，可以理解为ceph认为这个OSD已经不属于集群了，然后就会把PG 0.44 map到别的OSD上去，这个map也是按照一定的规则的，重映射之后呢，就会在另外两个OSD上找到0.44这个PG，而这只是创建了这个目录而已，丢失的数据是要从仅存的OSD上回填到新的OSD上的，处于回填状态的PG就会被标记为backfilling。

所以当一个PG处于remapped+backfilling状态时，可以认为其处于自我克隆复制的自愈过程。

Backfill

Stale

mon检测到当前PG的Primary所在的osd宕机。
Primary超时未向mon上报pg相关的信息(例如网络阻塞)。
PG内三个副本都挂掉的情况。

Inconsistent

Incomplete


## Ceph OSD 状态对照表


## Ceph PG 状态对照表

|---------------------+----------------------------------|
|          状态        |                描述              |
|:-------------------:|:---------------------------------|
|	active			| 当前拥有最新状态数据的pg正在工作中，能正常处理来自客户端的读写请求。 |
|	inactive		|	正在等待具有最新数据的OSD出现，即当前具有最新数据的pg不在工作中，不能正常处理来自客户端的读写请求。 |
|	activating		|	Peering 已经完成，PG 正在等待所有 PG 实例同步并固化 Peering 的结果 (Info、Log 等) |
|	clean	    	|	pg所包含的object达到指定的副本数量，即object副本数量正常 |
|	unclean			|	PG所包含的object没有达到指定的副本数量，比如一个PG没在工作，另一个PG在工作，object没有复制到另外一个PG中。 |
|	peering			|	PG所在的OSD对PG中的对象的状态达成一个共识（维持对象一致性） |
|	peered			|	peering已经完成，但pg当前acting set规模小于存储池规定的最小副本数（min_size） |
|	degraded		|	主osd没有收到副osd的写完成应答，比如某个osd处于down状态 |
|	stale			|	主osd未在规定时间内向mon报告其pg状态，或者其它osd向mon报告该主osd无法通信 |
|	inconsistent	|	PG中存在某些对象的各个副本的数据不一致，原因可能是数据被修改 |
|	incomplete      |	peering过程中，由于无法选出权威日志，通过choose_acting选出的acting set不足以完成数据修复，导致peering无法正常完成 |
|	repair		|	pg在scrub过程中发现某些对象不一致，尝试自动修复 |
|	undersized		|	pg的副本数少于pg所在池所指定的副本数量，一般是由于osd down的缘故 |
|	scrubbing		|	pg对对象meta的一致性进行扫描 |
|	deep			|	pg对对象数据的一致性进行扫描 |
|	creating		|	pg正在被创建 |
|	recovering		|	pg间peering完成后，对pg中不一致的对象执行同步或修复，一般是osd down了或新加入了osd |
|	recovering-wait	|	等待 Recovery 资源预留 |
|	backfilling		|	一般是当新的osd加入或移除掉了某个osd后，pg进行迁移或进行全量同步 |
|	down			|	包含必备数据的副本挂了，pg此时处理离线状态，不能正常处理来自客户端的读写请求 |
|	remapped		|	重新映射态。PG 活动集任何的一个改变，数据发生从老活动集到新活动集的迁移。在迁移期间还是用老的活动集中的主 OSD 处理客户端请求，一旦迁移完成新活动集中的主 OSD 开始处理 |
|   misplaced		|	有一些回填的场景：PG被临时映射到一个OSD上。而这种情况实际上不应太久，PG可能仍然处于临时位置而不是正确的位置。这种情况下个PG就是misplaced。这是因为正确的副本数存在但是有个别副本保存在错误的位置上。 |
|---------------------+----------------------------------|