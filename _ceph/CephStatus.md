---
title: Cep PG 和 OSD 集群状态分析
tag: ceph
---

## Ceph OSD 上电

## Ceph OSD 状态分析

up、down代表OSD临时故障或下电，不会发起数据恢复；in、out代表被踢出集群，集群发起数据恢复。

|---------------------+----------------------------------|
|          状态        |                描述              |
|:-------------------:|:---------------------------------|
|          int        |           OSD在Ceph集群内         |
|          out        |           OSD在Ceph集群外         |
|          up         |         OSD(心跳)活着且在运行      |
|          down       |          OSD 挂了且不再运行        |
|---------------------+----------------------------------|

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


### Ceph PG 状态分析

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