---
titile: cephFS锁分析
---

锁的作用
MDS中的锁是为了保护log的正常写入。每次对目录树进行操作前，需要先将目标path中涉及的节点加锁，在内存中修改完目录树（修改方式类似于RCU，即生成一个新节点，push_back到 队列 中）后，将新的目录树信息（只是此条path，不是整个目录树）记录到MDS的journal对象中，journal对象落盘后再将 队列 中的节点pop_front出来，至此，内存中的目录树已经能反映出之前的修改，加的锁也在此时开始释放，最后当前目录树的信息更新到meta pool的dir对象中。

锁的获取
加锁类型
加锁的类型分三类：rdlock（读）、wrlock（写）、xlock（互斥）。每次对目录树进行操作前都要将path上的节点进行适当地加锁。可从src/mds/Server.cc中观察这一操作：

    handle_client_xxx
      |-- rdlock_path_xlock_dentry或rdlock_path_pin_ref
      |-- mds->locker->acquire_locks(mdr, rdlocks, wrlocks, xlocks)
对于一个路径进行操作时，最后一个dentry之前的dentry都要加rdlock，避免别人进行修改。xlock用于创建或者修改节点时，比如mkdir时需要对新的dentry加xlock，创建新文件时需要对CInode::linklock（负责inode的nlink属性）加xlock。rdlock和xlock符合通常认知：共享读，互斥写。

wrlock比较特殊，主要用在CInode::filelock和CInode::nestlock上，前者负责保护当前目录的统计信息inode_t::dirstat，后者负责保护当前目录的递归统计信息inode_t::rstat；由于一个目录可以分成多个分片,甚至同一个分片也可以有多个副本分散于各个mds，为了允许对这些分片的统计信息同时进行修改，引入了wrlock，这些分散的被修改的信息将在后续的一个时间点上进行综合，最终传播到目录树的inode信息中（见CInode::preditry_journal_parents)。对于CInode::versionlock和CDentry::versionlock也会加wrlock锁，但由于是locallock sm，意义和simplelock的xlock一样，只是为了互斥写。

锁的种类和状态机
一个inode中的信息有很多种，每种由不同的锁来保护，每个锁的状态变化由遵循特定的规则——状态机。状态机的定义在src/mds/locks.c中，共有四种：

simaplelock state machine  
scatter_lock state machine  
file_lock state machine  
local_lock state machine  

CInode和CDentry中每种锁使用的状态机如下：
```cpp
struct LockType {
  int type;
  const sm_t *sm;
  explicit LockType(int t) : type(t) {
    switch (type) {
    case CEPH_LOCK_DN:
    case CEPH_LOCK_IAUTH:
    case CEPH_LOCK_ILINK:
    case CEPH_LOCK_IXATTR:
    case CEPH_LOCK_ISNAP:
    case CEPH_LOCK_IFLOCK:
    case CEPH_LOCK_IPOLICY:
      sm = &sm_simplelock;
      break;
    case CEPH_LOCK_IDFT:
    case CEPH_LOCK_INEST:
      sm = &sm_scatterlock;
      break;
    case CEPH_LOCK_IFILE:
      sm = &sm_filelock;
      break;
    case CEPH_LOCK_DVERSION:
    case CEPH_LOCK_IVERSION:
      sm = &sm_locallock;
      break;
    default:
      sm = 0;
    }
  }
};
```
其中locallock sm最简单，不做解释； 绝大多数锁使用simplelock sm，这些锁只需要“共享度、互斥写”功能；目录分片信息和递归统计信息则使用scatterlock sm，这种状态机能提供“共享读、共享写”功能；最复杂的是CInode::filelock使用的filelock sm，因为filelock既负责目录统计信息这种需要“共享读、共享写”的数据，也负责保护inode中的atime、mtime等需要“共享读、互斥写”的属性。

CInode种每种锁负责保护的数据可由CInode::encode_lock_state推断出来：
```cpp
void CInode::encode_lock_state(int type, bufferlist& bl)
{
  ...
  switch (type) {
  case CEPH_LOCK_IAUTH:
    encode(inode.version, bl);
    encode(inode.ctime, bl);
    encode(inode.mode, bl);
    encode(inode.uid, bl);
    encode(inode.gid, bl);  
    break;
  case CEPH_LOCK_ILINK:
    encode(inode.version, bl);
    encode(inode.ctime, bl);
    encode(inode.nlink, bl);
    break;
  case CEPH_LOCK_IDFT:
    ...
    encode(dirfragtree, bl);
    ...
  case CEPH_LOCK_IFILE:
    if (is_auth()) {
      encode(inode.version, bl);
      encode(inode.ctime, bl);
      encode(inode.mtime, bl);
      encode(inode.atime, bl);
      encode(inode.time_warp_seq, bl);
      if (!is_dir()) {
    encode(inode.layout, bl, mdcache->mds->mdsmap->get_up_features());
    encode(inode.size, bl);
    encode(inode.truncate_seq, bl);
    encode(inode.truncate_size, bl);
    encode(inode.client_ranges, bl);
    encode(inode.inline_data, bl);
      }
    ...
  case CEPH_LOCK_INEST:
    ...
```
锁的状态转换
有了状态机后就可根据预先定义的转换规则判断此次加锁是否可行，不可行的情况下要对锁的状态进行适当转换。锁状态的转换有两种驱动方式：

accquire_locks中根据当前sate判读是否能加锁，可以则直接变更锁的当前状态。
按照状态机中的sm_state_t::next指示逐步变换，这种方式一般有tick或者log flush回调等函数触发。
只有auth才有机会直接变更锁的当前状态，副本只能向auth发消息请求加锁

下图展示了加xlock时的状态变换，根据状态机描述，如果当前无法加xlock，则对锁进行一些转换，如调用Locker::simple_xlock()或Locker::simple_lock()，如果转换过程无法顺利进行（gather==true）则加锁失败。
加xlock时的状态转换
锁的释放
请求失败或完成后，Locker::drop_locks()负责锁的释放，其间会处理锁的等待队列，对锁的状态进行kick.

正常情况下要等到日志落盘后才会触发释放锁的动作，如果设置了mds_early_reply = true则提交完log就会释放rdlocks，但wrlock和xlock依然要等到log落盘后才释放。