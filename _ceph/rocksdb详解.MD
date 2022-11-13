---
titile: rocksdb 详解
---
## 简介
RocksDB项目是起源于Facebook，是一款作为各种存储介质上的服务器工作负载的存储引擎，最初专注于快速存储（尤其是闪存存储）。它是一个 C++ 库，用于存储任意大小的字节流的键和值。它支持点查找和范围扫描，并提供不同类型的 ACID 保证。
在可定制性和自适应性之间取得平衡。RocksDB 具有高度灵活的配置设置，可以调整为在各种生产环境中运行，包括 SSD、硬盘、ramfs 或远程存储。它支持各种压缩算法和良好的生产支持和调试工具。另一方面，还努力限制旋钮的数量，提供足够好的开箱即用性能，并在适用的地方使用一些自适应算法。

RocksDB 借鉴了开源leveldb项目的重要代码以及Apache HBase的想法。最初的代码是从开源 leveldb 1.5 fork出来的。

很多项目都接纳了RocksDB作为其后端存储的一种解决方案，如Mysql, Ceph, Flink, MongoDB, TiDB等。
  
项目链接：

[leveldb](https://github.com/google/leveldb): https://github.com/google/leveldb

[rocksdb](https://github.com/facebook/rocksdb): https://github.com/facebook/rocksdb

### 关于LSM树
LSM树，即日志结构合并树(Log-Structured Merge-Tree)。其实它并不属于一个具体的数据结构，它更多是一种数据结构的设计思想。大多NoSQL数据库核心思想都是基于LSM来做的，只是具体的实现不同。

传统关系型数据库使用btree或一些变体作为存储结构，能高效进行查找。但保存在磁盘中时它也有一个明显的缺陷，那就是逻辑上相离很近但物理却可能相隔很远，这就可能造成大量的磁盘随机读写。随机读写比顺序读写慢很多，为了提升IO性能，我们需要一种能将随机操作变为顺序操作的机制，于是便有了LSM树。

从概念上说，最基本的LSM是很简单的 。将之前使用一个大的查找结构（造成随机读写，影响写性能），变换为将写操作顺序的保存到一些相似的有序文件（也就是sstable)中。所以每个文件包 含短时间内的一些改动。因为文件是有序的，所以之后查找也会很快。文件是不可修改的，他们永远不会被更新，新的更新操作只会写到新的文件中。读操作检查很 有的文件。通过周期性的合并这些文件来减少文件个数。

![示意图](/assets/images/ceph/LSM-archietcture.png)

# 架构
RocksDB 是一个基于键值对存储接口的存储引擎库，其中键和值是任意字节流。RocksDB 将所有数据按排序顺序组织起来，常用的操作有Get(key), NewIterator(), Put(key, val), Delete(key), 和SingleDelete(key)。

RocksDB 的三个基本结构是memtable、sstfile和logfile。memtable是一种内存数据结构 - 新的写入被插入到memtable中，并且可以选择写入日志文件（又名。Write Ahead Log(WAL)）。日志文件是存储上按顺序写入的文件。当 memtable 填满时，它会被刷新到存储上的sstfile，并且可以安全地删除相应的日志文件。对 sstfile 中的数据进行排序以方便查找键。

RocksDB使用布隆过滤器来判定键在哪个sst文件中。为了避免随机写，它将数据积累到内存中的memtable中，然后一次性刷写到硬盘中。RocksDB的文件是不可变的，一旦生成就不会继续写该文件。记录不会被更新或者删除，会生成一个新文件。这会在硬盘生成一些多余的数据，会需要数据库Compaction（压缩），Compaction文件会移除冗余的键值对并腾出空间，如图所示

![RocskDB 架构图](/assets/images/ceph/LSM.png)

RocksDB用不同的排列组织数据，也就是层level，每层都有个目标大小，每层的目标大小增长倍数是相同的，默认是10倍，因此，如果第一层目标大小1g，那么2,3,4层大小就是10g，100g，1000g，一个键可能出现在不同的层，随着compaction，但是越新的值层越高，越旧的值层越低。

![RocskDB Level](/assets/images/ceph/rocksdb-level.png)

# RocskDB和LevelDB的区别

结构和levelDB大同小异，只是多了一些改进

1. 增加了column family，有了列簇的概念，可把一些相关的key存储在一起  
2. 内存中有多个immute memtalbe，可防止Leveldb中的 write stall(写停顿)  
3. 可支持多线程同时compaction，理论上多线程同时compction会比一个线程compaction要快  
4. 支持TTL过期淘汰机制  
5. flush与compation分开不同的线程池来调度，并具有不同的优先级，flush要优于compation，这样可以加快flush，防止stall  
6. 对SSD存储做了优化，可以以in-memory方式运行  
7. 增加了对 write ahead log（WAL）的管理机制，更方便管理WAL，WAL是binlog文件  
8. 支持多种不同的compaction策略

## RocskDB 和 Ceph
### Ceph Monitor 中使用了 RocskDB

![Monitor架构图](/assets/images/ceph/ceph-monitor-stack.jpg)

Ceph Monitor的内部其实是一个分布式kv数据库。从下往上分别是MonitorDBStore、Paxos和PaxosService。PaxosService负责保证每次都只会有一个提案进入paxos流程。Paxos模块具体实现了multi-Paxos算法。MonitorDBStore是对底层DB的抽象封装，将DB的基本操作事务封装成统一接口，当前DB默认使用rocksdb。

#### Ceph OSD 中使用了 RocksDB
OSD的底层存储引擎BlueStore是基于RocksDB开发的。

![BlueStore逻辑架构](/assets/images/ceph/bluestore.png)

BlueStore的逻辑架构如上图所示，模块的划分都还比较清晰，我们来看下各模块的作用：

RocksDB：rocksdb是facebook基于leveldb开发的一款kv数据库，BlueStore将元数据全部存放至RocksDB中，这些元数据包括存储预写式日志、数据对象元数据、Ceph的omap数据信息、以及分配器的元数据。

BlueRocksEnv：这是RocksDB与BlueFS交互的接口；RocksDB提供了文件操作的接口EnvWrapper，用户可以通过继承实现该接口来自定义底层的读写操作，BlueRocksEnv就是继承自EnvWrapper实现对BlueFS的读写。

BlueFS：BlueFS是BlueStore针对RocksDB开发的轻量级文件系统，用于存放RocksDB产生的.sst和.log等文件。

BlockDecive：BlueStore抛弃了传统的ext4、xfs文件系统，使用直接管理裸盘的方式；BlueStore支持同时使用多种不同类型的设备，在逻辑上BlueStore将存储空间划分为三层：慢速（Slow）空间、高速（DB）空间、超高速（WAL）空间，不同的空间可以指定使用不同的设备类型，当然也可使用同一块设备，具体我们会在后面的文章进行说明。

Allocator：负责裸设备的空间管理，只在内存做标记，目前支持StupidAllocator和BitmapAllocator两种分配器,Stupid基于extent的方式实现。

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io