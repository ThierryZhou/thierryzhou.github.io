---
title: 分布式文件系统CephFileSystem
tag: ceph
---

## Metadata Server的启动流程

mds进程启动后，通过向MDSMonitor发送Beacon消息，并接收从MDSMonitor返回的MDSMap消息逐步启动自己。Beacon除了用于告知monitor关于mds的健康状态，还有want_state这一信息。want_state的值是在mds启动或者收到MDSMap后由mds自己决定的(如向resolve、reconnect、rejoin等转换）。
MDSMonitor接收mds发来的Beacon消息，更新pending_fsmap，写入store。MDSMonitor::tick()在每次写完store后被调用，对pending_fsmap进行遍历，尝试给fsmap关联mds，然后更新pending_fsmap，再次写入store，完成后向mds发送MDSMap。关联到fsmap的mds分两类：一类是具有rank的，最终mds daemon会进入到STATE_ACTIVE状态，其中rank值由MDSMonitor分配；另一类是STATE_STANDBY_REPLAY，每个fsmap最多有一个，没有rank值。关联动作是通过遍历已创建fs map（如果创建了多个filesystem，那么此处就会有多个fs map），寻找可用的处于STATE_STANDBY状态的mds实现的，具体在MDSMonitor::maybe_expand_cluster()。
在众多mds状态中，由MDSMonitor路径负责的状态转换只有一部分，所以在MDSMonitor的代码里不会看到全部mds状态。其他状态转换是在mds端做的决策，此时monitor只是作为一个记录者，负责将状态写入store中。

## CephFS 报警对照表

|----------------------+----------------------------------|
|			状态		|		描述		|
|:--------------------:|:---------------------------------|
| Behind on trimming... | 日志回写落后于日志裁剪。mds的日志机制：mds以日志方式先保存元数据，元数据保存在每条操作的事件（event）中，事件（通常是1024个）组成segment。当segment到达一定数量时（mds_log_max_segments默认32）对日志进行裁剪，即将部分日志关联的元数据写回。出现该条告警实际上表明回写速度慢或者遇到了bug，单纯地将配置提高并不是最理想的办法。 |
| Client name failing to respond to capability release | 客户端没有及时响应释放cap的请求。在cephfs中客户端需要向mds获得响应的操作能力，称为cap。获得cap则有相关的操作能力。如果其他客户端需要操作时，mds会要求当前客户端释放cap。如果客户端出现bug或者没有响应，则mds会在60秒（session_timeout 设置）会出现该告警。 |
| Client name failing to respond to cache pressure | 客户端没有及时相应（mds的）缓存压力。元数据缓存一部分元数据信息，同时mds会在自身内存中缓存同样的信息。如果其缓存的元数据超过了最大inode缓存量或者最大内存用量，mds会要求客户端释放一定数量的缓存。如果在规定时间内即60s（mds_recall_warning_decay_rate的值）没有释放32k（默认设置在mds_recall_warning_threshold中，随后会减少）则产生告警 。产生告警的原因可能是客户端存在bug或者无法及时响应。 |
| Client name failing to advance its oldest client/flush tid | 客户端没有更新其最久客户端tid值。tid是指客户端和mds直接通信的task id。每次客户端完成任务后更新该task id，告知mds mds可以不用管该id之前的任务了。mds即可释放相关的占用资源。否则，资源不会被主动释放。当mds端自行记录的任务完成数超过100K（max_completed_requests设置）时，客户端并没有更新id，则产生相应的告警。出现该告警可能代表客户端存在bug。也遇到过mds因为锁问题部分请求卡住，重启mds 锁状态正常后可以恢复。 |
| MDS in read-only mode | 字面翻译mds进入只读模式。只读模式意味着在客户端上创建文件等操作元数据的行为将不被允许。进入只读的原因可能是向元数据池写入时发生错误，或者通过命令强制mds进入只读模式。 |
| N slow requests are blocked | 字面翻译多个慢请求在阻塞状态。出现该条告警意味着客户端的消息没有处理完成，超过了mds_op_complaint_time所规定的时间（默认30s）。可能出现的原因是mds运行缓慢，或者向rados写入日志未确认（底层pg或者osd出现问题），或者是mds存在的bug。此时，通过ops命令查看当前正在执行的操作，可进一步分析出现阻塞请求的原因。 |
| Too many inodes in cache | 字面翻译在mds的缓存中缓存了太多inode。mds的缓存指两个方面：inode数量和内存占用量。inode默认值mds_cache_size为100K，mds_cache_memory_limit为1G。到达一个告警的阈值后产生告警，一般为50%（mds_health_cache_threshold）。通过调整参数可以避免告警的出现，但是这只是治标的办法，治本的办法需要跟踪业务，了解资源占用的具体原因，是否只是通过调整参数可以解决。|
|---------------------+----------------------------------|

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io