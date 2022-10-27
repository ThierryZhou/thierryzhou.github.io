## Ceph Pg 状态对照表
|          状态       |                描述               |
| :----------------: | :-------------------------------: |
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