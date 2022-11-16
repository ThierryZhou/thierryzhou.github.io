---
title: Ceph快照详解
---

## 简介
Ceph快照功能基于RADOS实现，但是从使用方法上分成三种情况：

1. Pool Snapshot 对整个Pool打快照，该Pool中所有的对象都会受影响。
2. Self Managed Snapshot 用户管理的快照，Pool受影响的对象是受用户控制的，这里的用户往往是应用，如librbd。常见的形式就是针对某一个rbd卷进行快照。
3. 用于CephFS的快照，其中基于CephFS的快照由于CephFS一直是不稳定的功能所以默认关闭并且被描述为实验性质的功能，不推荐使用。即使是在CephFS第一个正式版本的Jewel（2016-06）中，CephFS-snapshot仍然是不推荐使用的功能。

## 快照的使用
image快照与pool快照
image快照与pool快照是互斥的，创建了image的存储池无法创建存储池的快照，因为存储池当前已经为unmanaged snaps mode了，而没有创建image的就可以做存储池快照。而如果创建了pool快照则无法创建image快照。

image快照的创建命令形如：rbd snap create {pool-name}/{image-name}@{snap-name}回滚命令形如：rbd snap rollback {pool-name}/{image-name}@{snap-name}

## CephFS快照
该功能属于实验性质的功能，不能应用与生产环境中。基本创建方法为在集群被挂载之后执行mkdir .snap/snapname，其中snapname为快照的名称。恢复数据的命令形如cp -ra .snap/snap1/* ./，删除快照的命令形如rmdir .snap/snap1

## 快照的原理

Ceph的快照与其他系统的快照一样，是基于COW(copy-on-write)实现的。其实现由RADOS支持，基于OSD服务端——每次做完快照后再对卷进行写入时就会触发COW操作，即先拷贝出原数据对象的数据出来生成快照对象，然后对原数据对象进行写入。于此同时，每次快照的操作会更新卷的元数据，以及包括快照ID，快照链，parent信息等在内的快照信息。

需要注意的一点是克隆依赖快照的实现，克隆是在一个快照的基础上实现了可写的功能，类似于通常所说的可写快照，但是克隆和快照在实现层面上是完全不同的——快照是RADOS支持的，基于OSD客户端，而RBD的克隆操作是RBD客户端实现的一种COW操作，对于OSD的Server是无感知的。

此外image快照和pool快照的区别是由不同的使用方式导致的，底层的实现没有本质上的区别。从OSD的角度看，池快照和自管理的快照之间的区别在于SnapContext是通过客户端的MOSDOp还是通过最新的OSDMap到达osd。这一点将在快照的实现细节方面详述（OSD::make_writeable）。

#### 快照的实现
##### 快照的相关概念
pool： 每个池都是逻辑上的隔离单位，不同的pool可以有不同的数据处理方式，包括：副本数，Placement Groups，CRUSH。 Rules，快照，ownership都是通过池隔离。
head对象：卷原始对象，包含了SnapSet（详见关键数据结构部分）。
snap对象：卷打快照后通过cow拷贝出来的对象，该对象为只读。
snap_seq: 快照的序列号（详见关键数据结构部分）。
snapdir对象： head对象被删除后，仍然有snap和clone对象，系统自动创建一个snapdir对象来保存snapset的信息。（详见关键数据结构部分）。
rbd_header对象： 在rados中，对象里没有数据，卷的元数据都是作为这个对象的属性以omap方式记录到leveldb里。
关键数据结构
快照的关键数据结构如下：
SnapContext在客户端中保存snap相关的信息，是当前为对象定义的快照合集。这个结构持久化的存储在RBD的元数据中。

```cpp
//代码来源：librados/IoCtxImpl.h
struct librados::IoCtxImpl {
  // ...
  snapid_t snap_seq;   //根据是否有快照值为snap的快照序号或者CEPH_NOSNAP
  ::SnapContext snapc;
  // ...
};
```

在IoCtxImpl里的snap_seq也被称为快照的id，当打开一个image时，如果打开的是一个卷的快照，则该值为快照对应的序号，否则该值为CEPH_NOSNAP表示操作的不是卷的快照，是卷自身。

```cpp
// SnapContext在common/snap_types.h中定义：

struct SnapContext {
  snapid_t seq;            // 'time' stamp（最新的快照序列号）
  vector<snapid_t> snaps;  // 当前存在的快照序号，降序排列
  // ...
};
```
SnapSet是在ceph的服务端也就是osd端保存快照的对象（引自：osd_types.h）:

```cpp
struct SnapSet {
  snapid_t seq; //最新的快照序列号
  bool head_exists; //head对象是否存储/存在
  vector<snapid_t> snaps;    // 所有快照序号的降序列表
  vector<snapid_t> clones;   // 所有clone对象的序号升序列表。保存在做完快照后，对原对象进行写入时触发cow进行clone的快照序号，注意并不是每个快照都需要clone对象，只有做完快照后，对相应的对象进行写入操作时才会clone去拷贝数据；
  map<snapid_t, interval_set<uint64_t> > clone_overlap;  // 与上次clone对象的overlap的部分，记录在其clone数据对象后，也就是原数据对象上未写过的数据部分，是采用offset~len的方式进行记录的，比如{2=[0~1646592,1650688~12288,1667072~577536]}；
  map<snapid_t, uint64_t> clone_size;//clone对象的size（有的对象一开始并不是默认的对象大小）
  // ...
};
```
该数据结构保存在OSD端快照的相关信息，将会跟踪：
对象的全部的快照集合，当前存在的全部克隆，克隆的大小，其中的clone_overlap保存本次clone对象和上次clone对象的重叠（overlap）部分，clone操作之后，每次的写操作都要维护这个信息。这个信息用于在数据恢复阶段对象恢复的优化。

```cpp
// osd/ReplicatedBackend.h:

  struct OpContext {
    // ...
    const SnapSet *snapset; //旧的SnapSet，也就是OSD服务端保存的快照信息
    SnapSet new_snapset;  // 新的SnapSet，也就是写操作过后生成的结果
    SnapContext snapc;   //写操作带的客户端的SnapContext信息
    // ...
};
```

##### 快照的创建
创建rbd快照基本步骤如下：

向monitor发送请求，获取一个最新的快照序号snap_seq，monitor会递增该pool的snap_seq，然后将该值返回给librbd。
librbd将新的snap_seq替换到原来的image中，snap_name和snap_seq将会被保存到rbd的元数据中。
快照的写
当做了多次快照的情况下，Ceph采用的方法是旧有快照引用新的快照，这里举一个例子来说明这一情况：

假设有镜像（卷）中已经有a，b两个文件，此时进行第一次快照记做snap1，然后修改a文件，系统将会把原始镜像中的a文件的数据拷贝到snap1中，直接在原始镜像中进行读写。

这个时候我们进行第二次快照，记做snap2，然后我们修改a，b两个文件，a文件会直接生成a文件对应的快照，而对于b文件，由于在第一次快照后没有进行修改，系统会直接将原镜像的数据拷贝出来生成快照镜像。这就是所谓的旧快照引用新快照（当需要恢复到snap1节点的时候，snap1将引用snap2的数据来还原原始数据）

更具体的说每个快照都保留一个snap_seq，Image可以看成一个Head Version的Snapshot，客户端写操作，必须带SnapContex结构，也就是需要带最新的快照序号seq和所有的快照序号snaps。

在OSD端，对象的Snap相关的信息保存在SnapSet数据结构中，每次IO操作都会带上snap_seq发送给OSD，OSD会查询该IO操作涉及的object的snap_seq情况。当创建一个快照以后，对镜像中的对象进行写操作时会带上新的snap_seq，Ceph接到请求后会先检查对象的Head Version，如果发现该写操作所带有的snap_seq大于原本对象的snap_seq，那么就会对原来的对象克隆一个新的Object Head Version，原来的对象会作为Snapshot，新的Object Head会带上新的snap_seq，也就是librbd之前申请到的。

ceph也有一套Watcher回调通知机制，当别的的客户端做了快照，产生了以新的快照序号，当该客户端访问，osd端知道最新快照需要变化后，通知相应的连接客户端更新最新的快照序号。如果没有及时更新，也没有太大的问题，客户端会主动更新快照序号，然后重新发起写操作。

具体到代码流程如下：

判断服务端的快照序号，如果大于客户端的序号，则用服务端的快照信息更新客户端的信息，需要注意的是客户端的序号是不允许小于服务端的序号的，如发生服务端的序号大于客户端的序号则参见上述的watcher回调通知机制。
把已经删除的快照过滤掉。
如果head对象存在切snaps的size不为空，并且客户端的最新快照序号大于服务端的最新快照序号，则需要克隆对象。
克隆完成后修改clone_overlap和clone_size的记录。
更新服务端快照信息。
源码实现：osd/ReplicatedPG.cc
void ReplicatedPG::make_writeable(OpContext *ctx)

##### 快照的读
快照读取时，输入参数为rbd的name和快照的名字。rbd的客户端通过访问rbd的元数据，来获取快照对应的snap_id，也就是快照对应的snap_seq值。

在osd端，获取head对象保存的SnapSet数据结构。然后根据snaps和clones来计算快照所对应的正确的快照对象。

```cpp
// 源码实现：osd/ReplicatedPG.cc

int ReplicatedPG::find_object_context(const hobject_t& oid,
				      ObjectContextRef *pobc,
				      bool can_create,
				      bool map_snapid_to_clone,
				      hobject_t *pmissing);
```
##### 快照的回滚
快照的回滚，就是把当前的head对象，回滚到某个快照对象。 具体操作如下：

删除当前head对象的数据
copy 相应的snap对象到head对象
```cpp
// 源码实现：osd/ReplicatedPG.cc
ReplicatedPG::_rollback_to
```
##### 快照的删除
向monitor集群发出请求，将快照的id添加到已清除的快照的列表中。（或者将其从吃池快照集中删除）
删除快照时，直接删除SnapSet相关的信息，并删除相应的快照对象。需要计算该快照是否被其它快照对象共享。
ceph的删除是延迟删除，并不直接删除。当pg是clean状态并且没进行scrubbing时由由snap_trim_wq异步执行。
```cpp
// 相关源码：
struct SnapTrimmer : public boost::statechart::state_machine< SnapTrimmer, NotTrimming >
```
##### CephFS快照
CephFS通过在希望快照的目录下执行mkdir创建.snap目录来创建快照。

无论在任何时候创建快照，都会生成一个SnapRealm，它保留较少的数据，用于将SnapContex与每个打开的文件关联以进行写入。

SnapRealm中包含sr_t srnode，past_parents ，past_children，inodes_with_caps等属于快照的一部分的信息。

st_r：是磁盘上的元数据，包含序列计数器，时间戳，相关的快照id列表和past_parents。

当执行快照操作的时候，客户端会将请求发送到MDS服务器，然后在服务器的Server::handle_client_mksnap()中处理，它会从SnapServer中非配一个snapid，同时利用新的SnapRealm创建一个新的inode，然后将其提交到MDlog，提交后会触发MDCache::do_realm_invalidate_and_update_notify()，快照的大部分工作由这部分完成。快照的元数据会作为目录信息的一部分被存储在OSD端。

需要注意的是CephFS的快照和多个文件系统的交互是存在问题的——每个MDS集群独立分配snappid，如果多个文件系统共享一个池，快照会冲突。如果此时有客户删除一个快照，将会导致其他人丢失数据，并且这种情况不会抛出异常，这也是CephFS的快照不推荐使用的原因。

##### 克隆卷
克隆卷的操作必须要在快照被保护起来（无法删除）之后才能进行。在克隆卷生成之后，在librbd端开始读写时会先根据快照链构造出其父子关系，而在具体的I/O请求的时候这个父子关系会被用到。克隆出的卷在有新数据写入之前，读取数据的需求都是引用父卷和快照的数据。

对于克隆卷的读写会先去找这个卷的对象，如果未找到，就去寻找其parent对象，层层往上，直到找到位置。所以一旦快照链比较长就会导致效率较低，所以Ceph的克隆卷提供了flatten功能，这个功能会将所有的数据全部拷贝一份，然后生成一个新的卷。新生成的卷会完全独立存在，不再保持原有的父子关系。但是flatten本身是一个耗时比较大的操作。

