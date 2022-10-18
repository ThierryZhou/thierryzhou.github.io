# Ceph FAQs
## Ceph Pg 状态对照表
|		状态		|		描述		|
| ------------------| --------------------------------- |
|		active		|	当前拥有最新状态数据的pg正在工作中，能正常处理来自客户端的读写请求。 |
|		inactive	|	正在等待具有最新数据的OSD出现，即当前具有最新数据的pg不在工作中，不能正常处理来自客户端的读写请求。 |
|		activating	|	Peering 已经完成，PG 正在等待所有 PG 实例同步并固化 Peering 的结果 (Info、Log 等) |
|		clean	    |	pg所包含的object达到指定的副本数量，即object副本数量正常 |
|		unclean		|	PG所包含的object没有达到指定的副本数量，比如一个PG没在工作，另一个PG在工作，object没有复制到另外一个PG中。 |
|		peering		|	PG所在的OSD对PG中的对象的状态达成一个共识（维持对象一致性） |
|		peered		|	peering已经完成，但pg当前acting set规模小于存储池规定的最小副本数（min_size） |
|		degraded	|	主osd没有收到副osd的写完成应答，比如某个osd处于down状态 |
|		stale		|	主osd未在规定时间内向mon报告其pg状态，或者其它osd向mon报告该主osd无法通信 |
|		inconsistent	|	PG中存在某些对象的各个副本的数据不一致，原因可能是数据被修改 |
|		incomplete		|	peering过程中，由于无法选出权威日志，通过choose_acting选出的acting set不足以完成数据修复，导致peering无法正常完成 |
|		repair		|	pg在scrub过程中发现某些对象不一致，尝试自动修复 |
|		undersized		|	pg的副本数少于pg所在池所指定的副本数量，一般是由于osd down的缘故 |
|		scrubbing		|	pg对对象meta的一致性进行扫描 |
|		deep		|	pg对对象数据的一致性进行扫描 |
|		creating	|	pg正在被创建 |
|		recovering	|	pg间peering完成后，对pg中不一致的对象执行同步或修复，一般是osd down了或新加入了osd |
|		recovering-wait	|	等待 Recovery 资源预留 |
|		backfilling	|	一般是当新的osd加入或移除掉了某个osd后，pg进行迁移或进行全量同步 |
|		down		|	包含必备数据的副本挂了，pg此时处理离线状态，不能正常处理来自客户端的读写请求 |
|		remapped		|	重新映射态。PG 活动集任何的一个改变，数据发生从老活动集到新活动集的迁移。在迁移期间还是用老的活动集中的主 OSD 处理客户端请求，一旦迁移完成新活动集中的主 OSD 开始处理 |
|		misplaced		|	有一些回填的场景：PG被临时映射到一个OSD上。而这种情况实际上不应太久，PG可能仍然处于临时位置而不是正确的位置。这种情况下个PG就是misplaced。这是因为正确的副本数存在但是有个别副本保存在错误的位置上。 |
## CephFS 报警对照表
|		状态		|		描述		|
| ------------------ | --------------------------------- |
| Behind on trimming... | 日志回写落后于日志裁剪。mds的日志机制：mds以日志方式先保存元数据，元数据保存在每条操作的事件（event）中，事件（通常是1024个）组成segment。当segment到达一定数量时（mds_log_max_segments默认32）对日志进行裁剪，即将部分日志关联的元数据写回。出现该条告警实际上表明回写速度慢或者遇到了bug，单纯地将配置提高并不是最理想的办法。 |
| Client name failing to respond to capability release | 客户端没有及时响应释放cap的请求。在cephfs中客户端需要向mds获得响应的操作能力，称为cap。获得cap则有相关的操作能力。如果其他客户端需要操作时，mds会要求当前客户端释放cap。如果客户端出现bug或者没有响应，则mds会在60秒（session_timeout 设置）会出现该告警。 |
| Client name failing to respond to cache pressure | 客户端没有及时相应（mds的）缓存压力。元数据缓存一部分元数据信息，同时mds会在自身内存中缓存同样的信息。如果其缓存的元数据超过了最大inode缓存量或者最大内存用量，mds会要求客户端释放一定数量的缓存。如果在规定时间内即60s（mds_recall_warning_decay_rate的值）没有释放32k（默认设置在mds_recall_warning_threshold中，随后会减少）则产生告警 。产生告警的原因可能是客户端存在bug或者无法及时响应。 |
| Client name failing to advance its oldest client/flush tid | 客户端没有更新其最久客户端tid值。tid是指客户端和mds直接通信的task id。每次客户端完成任务后更新该task id，告知mds mds可以不用管该id之前的任务了。mds即可释放相关的占用资源。否则，资源不会被主动释放。当mds端自行记录的任务完成数超过100K（max_completed_requests设置）时，客户端并没有更新id，则产生相应的告警。

出现该告警可能代表客户端存在bug。也遇到过mds因为锁问题部分请求卡住，重启mds 锁状态正常后可以恢复。 |
| MDS in read-only mode | 字面翻译mds进入只读模式。只读模式意味着在客户端上创建文件等操作元数据的行为将不被允许。进入只读的原因可能是向元数据池写入时发生错误，或者通过命令强制mds进入只读模式。 |
| N slow requests are blocked | 字面翻译多个慢请求在阻塞状态。出现该条告警意味着客户端的消息没有处理完成，超过了mds_op_complaint_time所规定的时间（默认30s）。可能出现的原因是mds运行缓慢，或者向rados写入日志未确认（底层pg或者osd出现问题），或者是mds存在的bug。此时，通过ops命令查看当前正在执行的操作，可进一步分析出现阻塞请求的原因。 |
| Too many inodes in cache | 字面翻译在mds的缓存中缓存了太多inode。mds的缓存指两个方面：inode数量和内存占用量。inode默认值mds_cache_size为100K，mds_cache_memory_limit为1G。到达一个告警的阈值后产生告警，一般为50%（mds_health_cache_threshold）。通过调整参数可以避免告警的出现，但是这只是治标的办法，治本的办法需要跟踪业务，了解资源占用的具体原因，是否只是通过调整参数可以解决。|

## Ceph集群异常处理

### 1.客户端挂载ceph查看的容量与实际容量不符？
  此问题一般是由于部分OSD使用率远高于其他OSD的使用率导致，属于数据平衡性问题的一种。
利用osd utilization命令优化平衡性即可。
```shell
ceph osd reweight-by-utilization
```

### 2.如何解决osd crash down以后系统中的报警？
```shell
ceph crash  archive-all
```

### 3.如何解决ceph集群恢复速度慢的问题？
```shell
# 仅考虑数据恢复，不考虑数据访问速度
ceph tell 'osd.*' injectargs --osd_max_backfills 32
ceph tell 'osd.*' injectargs --osd_recovery_max_active_hdd 16
ceph tell 'osd.*' injectargs --osd_recovery_max_active_ssd 64
ceph tell 'osd.*' injectargs --osd_recovery_sleep_hdd 0
ceph tell 'osd.*' injectargs --osd_backfill_scan_min 32

# 恢复兼容性能模式
ceph tell 'osd.*' injectargs --osd_max_backfills 8
ceph tell 'osd.*' injectargs --osd_recovery_max_active_hdd 4
ceph tell 'osd.*' injectargs --osd_recovery_max_active_ssd 16
ceph tell 'osd.*' injectargs --osd_recovery_sleep_hdd 0.0001
ceph tell 'osd.*' injectargs --osd_recovery_max_single_start 8

# 生产环境
ceph tell 'osd.*' injectargs --osd_max_backfills 4
ceph tell 'osd.*' injectargs --osd_recovery_max_active_hdd 4
ceph tell 'osd.*' injectargs --osd_recovery_max_active_ssd 16
ceph tell 'osd.*' injectargs --osd_recovery_sleep_hdd 0.01
ceph tell 'osd.*' injectargs --osd_recovery_max_single_start 4
```

### 4.如何修复有问题的pg？
```shell
ceph health detail
# ...
# pg 65.168 is incomplete, acting [12,5,10] (reducing pool aifs-data0 min_size from 2 may help; search ceph.com/docs for 'incomplete')
# ...

# 修复pg
ceph pg scrub 65.168
ceph pg deep-scrub 65.168
ceph pg repair 65.168

# 修复pg所对应的osd
ceph osd repair 12
ceph osd repair 5
ceph osd repair 10
```

### 5.如何手动移动pg从osd1到osd2？
```shell
ceph osd pg-upmap-items a.b 1 2
```

### 6.如何暂停Ceph中正在运行的服务，进行调试或是优化？
```shell
# 备份当前配置
ceph osd set noout
kubectl get deployment rook-ceph-osd-$i -n rook-ceph > rook-ceph-osd-$i.yaml

# 服务暂停
kubectl -n rook-ceph patch deployment rook-ceph-osd-$i --type='json' -p '[{"op":"remove", "path":"/spec/template/spec/containers/0/livenessProbe"}]'
kubectl -n rook-ceph patch deployment rook-ceph-osd-$i -p '{"spec": {"template": {"spec": {"containers": [{"name": "osd", "command": ["sleep", "infinity"], "args": []}]}}}}'

#服务恢复
kubectl apply --replace -f rook-ceph-osd-$i.yaml
ceph osd unset noout
```

### 7.如果关闭一些ceph集群后台执行的任务
```shell
#
ceph osd set noout
#ceph osd unset noout
ceph osd set nocovery
#ceph osd unset nocovery
ceph osd set noscrub
#ceph osd unset noscrub
ceph osd set nodeep-scrub
#ceph osd unset nodeep-scrub
ceph osd set nobackfill
#ceph osd unset nobackfill

```

### 8. 如何解决部分pg调用ceph osd force-create-pg并且经过长期等待仍无法重建的问题？
```shell
```

### 9. 如何解决osd之间数据不均衡问题？
```shell
ceph osd reweight-by-utilization 0.85
```


### 10. 如何解决pg长期处于unknown状态？
```shell
# 重建pg对象
ceph osd force-create-pg a.b

# 当 force create 无效时执行下面的命令
ceph osd pg-temp a.b 1 10 14
ceph osd pg-upmap a.b 1 10 14
```

### 11. 如何显示更详尽的日志以方便调试和追踪问题
```shell
ceph tell osd.0 injectargs --debug-osd 0/5
ceph tell mon.a injectargs --debug-osd 0/5
ceph tell mds.a injectargs --debug-osd 0/5
```

### 12.如何打开和关闭本地ceph调试日志？
```shell
echo "module libceph +p" >/sys/kernel/debug/dynamic_debug/control
echo "module ceph +p" >/sys/kernel/debug/dynamic_debug/control

echo "module libceph -p" >/sys/kernel/debug/dynamic_debug/control
echo "module ceph -p" >/sys/kernel/debug/dynamic_debug/control
```

### 13. 修改解决因为rook ceph operator重置后的monitor集群后secret和configmap中mon_host不匹配的问题？
```shell
mon_host=$(kubectl -n rook-ceph get svc rook-ceph-mon-b -o jsonpath='{.spec.clusterIP}')
kubectl -n rook-ceph patch secret rook-ceph-config -p '{"stringData": {"mon_host": "[v2:'"${mon_host}"':3300,v1:'"${mon_host"':6789]", "mon_initial_members": "'"${good_mon_id}"'"}}'
```

### 14.如何解决CephFS MDS服务无法进入active状态的问题？
```shell
# Session table
cephfs-table-tool cephfs:all reset session
# SnapServer
cephfs-table-tool cephfs:all reset snap
# InoTable
cephfs-table-tool cephfs:all reset inode
# Journal
cephfs-journal-tool --rank cephfs:all journal reset
# Root inodes ("/" and MDS directory)
cephfs-data-scan init --force-init
```

### 15.CephFS Monitor出现容量报警。
```shell
[WRN] MON_DISK_BIG: mons a,b,c,d,e are using a lot of disk space
    mon.a is 18 GiB >= mon_data_size_warn (15 GiB)
    mon.b is 17 GiB >= mon_data_size_warn (15 GiB)
    mon.c is 18 GiB >= mon_data_size_warn (15 GiB)
    mon.d is 18 GiB >= mon_data_size_warn (15 GiB)
    mon.e is 18 GiB >= mon_data_size_warn (15 GiB)
```
排查此问题是需要观察集群是否处于HEALTH_OK状态，如果是的则使用‘解决办法1’，如果不是则需要进入问题排查流程
#### 1.解决办法1：使mon进入数据压缩模式

```shell
ceph tell mon.* compact
```
#### 2.解决办法2: 

### 16.Ceph Cluster Monitor出现Slow OPS报警。
```shell
[WRN] SLOW_OPS: 10319 slow ops, oldest one blocked for 736 sec, daemons [mon.c,mon.d,mon.e,mon.h] have slow ops.
```
如果出现Monintor 重启或暂停正在进行同步的客户端

### 17.Ceph 出现Operator修改了Mon列表如何处理。
修改ConfigMap rook-ceph-csi-config rook-ceph-mon-endpoints
修改Secret rook-ceph-config

### 18.Ceph OSD出现大量报警并且严重影响正常使用时如何处理。


### 19.CephFS 如何进行Metadata数据修复
```
# Worker 0
for i in {0..3}; do cephfs-data-scan scan_extents --worker_n $i --worker_m 16 <data pool>
# Worker 1
for i in {4..7}; do cephfs-data-scan scan_extents --worker_n $i --worker_m 16 <data pool>
# Worker 2
for i in {8..11}; do cephfs-data-scan scan_extents --worker_n $i --worker_m 16 <data pool>
# Worker 3
for i in {12..15}; do cephfs-data-scan scan_extents --worker_n $i --worker_m 16 <data pool>

# Worker 0
for i in {0..3}; do cephfs-data-scan scan_inodes --worker_n $i --worker_m 16 <data pool>
# Worker 1
for i in {4..7}; do cephfs-data-scan scan_inodes --worker_n $i --worker_m 16 <data pool>
# Worker 2
for i in {8..11}; do cephfs-data-scan scan_inodes --worker_n $i --worker_m 16 <data pool>
# Worker 3
for i in {12..15}; do cephfs-data-scan scan_inodes --worker_n $i --worker_m 16 <data pool>

# Worker 0
for i in {0..3}; do  cephfs-data-scan scan_links --worker_n $i --worker_m 16
# Worker 1
for i in {4..7}; do cephfs-data-scan scan_links --worker_n $i --worker_m 16
# Worker 2
for i in {8..11}; do cephfs-data-scan scan_links --worker_n $i --worker_m 16
# Worker 3
for i in {12..15}; do cephfs-data-scan scan_links --worker_n $i --worker_m 16

# 如何在Rook命名空间下查看data scan的执行进度
kg po -n rook-ceph | grep rook-ceph-tools | awk '{ print $1; }' | xargs -I{} kubectl exec -ti {} -- ps aux | grep cephfs-data-scan

```
