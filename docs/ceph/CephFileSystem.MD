# MDS JOURNALING
## CEPHFS METADATA POOL

CepFS使用一个独立的元数据存储池来管理CephFS中的文件元数据(inode和dentry)。元数据池中保存了包括文件系统层次结构在内CephFS中所有的文件信息。此外，CepFS维护的元信息还包括如文件系统日志、打开的文件表、会话映射等与文件系统有关的其他信息。

## CEPHFS MDS JOURNALING
在执行文件系统操作之前，CepFS元数据服务器将元数据事件的日志流到元数据池中的RADOS中。主MDS守护进程为CepFS中的文件和目录管理元数据。

CepFS使用日志有以下几个原因:

1. 一致性:在MDS故障转移时，日志事件可以被重播，以达到一致的文件系统状态。此外，需要对备份存储进行多次更新的元数据操作需要记录崩溃一致性(以及其他一致性机制，如锁定等)。

2. 性能:日志更新(大部分)是连续的，因此对日志的更新非常快。此外，可以将更新批处理为单个写操作，从而节省更新文件不同部分所涉及的磁盘寻道时间。拥有一个大的日志还可以帮助备MDS加热它的缓存，这间接地帮助MDS进行故障转移。

每个活动元数据服务器在元数据池中维护自己的日志。日记账在多个对象上有条纹。不需要的(视为旧的)日志条目由元数据服务器进行修剪。

## JOURNAL EVENTS
除了记录文件系统元数据更新之外，cepfs还记录各种其他事件，例如客户端会话信息和目录导入/导出状态。这些事件被元数据服务器用来根据需要重新建立正确的状态，例如，Ceph MDS尝试在重新启动时重新连接客户端，当日志事件被重放，并且日志中特定的事件类型指定客户端实体类型在重新启动之前与MDS有会话。

为了检查日志中记录的事件列表，CephFS提供了一个命令行实用程序CephFS -journal-tool，它可以如下所示使用:

cephfs-journal-tool --rank=<fs>:<rank> event get list

cepfs -journal-tool也用于发现和修复损坏的Ceph文件系统。

## JOURNAL EVENT TYPES

以下是MDS记录的各种事件类型。

EVENT_COMMITTED:将请求(id)标记为已提交。

EVENT_EXPORT:将目录映射到MDS级别。

EVENT_FRAGMENT:跟踪目录碎片(拆分/合并)的各个阶段。

EVENT_IMPORTSTART: MDS级别开始导入目录分片时记录。

EVENT_IMPORTFINISH: MDS级别导入目录片段完成时记录。

EVENT_NOOP:没有用于跳过日志区域的操作事件类型。

EVENT_OPEN:跟踪哪些索引节点有打开的文件句柄。

EVENT_RESETJOURNAL:用于将日志标记为重置后截断。

EVENT_SESSION:跟踪打开的客户端会话。

EVENT_SLAVEUPDATE:记录被转发到(从)mds的操作的各个阶段。

EVENT_SUBTREEMAP:目录索引到目录内容(子树分区)的映射。

EVENT_TABLECLIENT:记录客户端表(snap/anchor)的mds视图的转换状态。

EVENT_TABLESERVER:记录服务器表的mds视图的转换状态(snap/anchor)。

EVENT_UPDATE:记录inode上的文件操作。