---
title: 在生产环境中使用CephFS以及CephFS的 FAQ 总结
tag: ceph
---
## 前言

CephFS是目前为性能可靠并且较为成熟可靠的分布式文件系统，但是作为分布式文件系统其复杂成都也是相当高的，如果要在生产环境中使用它，首先我们需要做好各种灾备方案，并且熟悉CephFS的故障处理流程。

## 生产环境准备


### Ceph FAQ

##### 1.如何解决CephFS MDS服务无法进入active状态的问题？
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

##### 2. CephFS 如何恢复损毁的Metadata Pool？

1) 模拟数据损毁  
```shell
# 删除所有 metadata 中的数据
$ for i in `rados -p cephfs.a.meta ls`; rados -p cephfs.a.meta rm $i; done

# 查看集群状态
$ ceph -s
  cluster:
    id:     399aa358-dc80-4bed-a957-7e27f8943b67
    health: HEALTH_WARN
            1 filesystem is degraded
            insufficient standby MDS daemons available

  services:
    mon: 3 daemons, quorum a,b,c (age 31h)
    mgr: x(active, since 31h)
    mds: 1/1 daemons up
    osd: 3 osds: 3 up (since 31h), 3 in (since 11d)

  data:
    volumes: 0/1 healthy, 1 recovering
    pools:   4 pools, 81 pgs
    objects: 136 objects, 5.5 MiB
    usage:   3.0 GiB used, 300 GiB / 303 GiB avail
    pgs:     81 active+clean

# 查看 fs 状态
$ ceph fs status
a - 0 clients
=
RANK      STATE      MDS  ACTIVITY   DNS    INOS   DIRS   CAPS
 0    replay(laggy)   a                0      0      0      0
     POOL        TYPE     USED  AVAIL
cephfs.a.meta  metadata  1095k  98.9G
cephfs.a.data    data    14.3M  98.9G
```
2) 创建 recovery fs 从 data pool 中恢复一个可用的备份文件系统。
```shell
# 首先，将现有的文件系统停止，以防止对数据池的进一步修改。卸载所有客户端。
$ ceph fs fail a

# 接下来，创建一个恢复文件系统，我们将在其中填充由原始数据池支持的新元数据池。
$ ceph osd pool create cephfs_recovery_meta
$ ceph fs new cephfs_recovery cephfs_recovery_meta cephfs.a.data  --recover --allow-dangerous-metadata-overlay

# 恢复过程中我们将关闭MDS，因为我们不希望它与元数据池进一步交互。
$ ceph fs fail cephfs_recovery

# 接下来，我们将重置MDS创建的初始元数据:
$ cephfs-table-tool cephfs_recovery:0 reset session
$ cephfs-table-tool cephfs_recovery:0 reset snap
$ cephfs-table-tool cephfs_recovery:0 reset inode
$ cephfs-journal-tool --rank cephfs_recovery:0 journal reset --force

# 现在从数据池中恢复元数据池:
$ cephfs-data-scan init --force-init --filesystem cephfs_recovery --alternate-pool cephfs_recovery_meta
$ cephfs-data-scan scan_extents --alternate-pool cephfs_recovery_meta --filesystem a cephfs.a.data
$ cephfs-data-scan scan_inodes --alternate-pool cephfs_recovery_meta --filesystem a --force-corrupt cephfs.a.data
$ cephfs-data-scan scan_links --filesystem cephfs_recovery

# (注意，配置也可以是全局设置的，也可以是通过ceph.conf文件设置的。)现在，允许MDS加入恢复文件系统:
$ ceph fs set cephfs_recovery joinable true

# 最后，运行前向清除以修复统计信息。确保您有一个MDS正在运行并发出:
$ ceph fs status # get active MDS
$ ceph tell mds.<id> scrub start / recursive repair
```
3) 尝试恢复文件系统，成功者则流程结束，不成功继续第4)步
```shell
$ cephfs-journal-tool --rank=a:0 event recover_dentries list --alternate-pool cephfs_recovery_meta


$ ceph fs status
a - 0 clients
=
RANK  STATE   MDS     ACTIVITY     DNS    INOS   DIRS   CAPS
 0    active   c   Reqs:    0 /s    13     16     12      0
     POOL        TYPE     USED  AVAIL
cephfs.a.meta  metadata   207k  98.9G
cephfs.a.data    data    14.3M  98.9G
cephfs_recovery - 0 clients
===============
RANK  STATE   MDS     ACTIVITY     DNS    INOS   DIRS   CAPS
 0    active   b   Reqs:    0 /s    10     13     12      0
        POOL            TYPE     USED  AVAIL
cephfs_recovery_meta  metadata  96.0k  98.9G
   cephfs.a.data        data    14.3M  98.9G
STANDBY MDS
     a
```

4) 利用 recovery fs 恢复原来已经被损毁的 metadata pool
```shell
# 确保整个过程文件系统处于关闭状态
ceph fs fail a
ceph fs set a joinable false

# 接下来，我们将重置MDS创建的初始元数据
cephfs-table-tool a:0 reset session
cephfs-table-tool a:0 reset snap
cephfs-table-tool a:0 reset inode
cephfs-journal-tool --rank a:0 journal reset --force

# 利用数据池和已经创建好的recovery fs恢复元数据池
cephfs-data-scan init --force-init --filesystem a --alternate-pool cephfs.a.meta
cephfs-data-scan scan_extents --alternate-pool cephfs.a.meta --filesystem cephfs_recovery cephfs.a.data
cephfs-data-scan scan_inodes --alternate-pool cephfs.a.meta --filesystem cephfs_recovery --force-corrupt cephfs.a.data
cephfs-data-scan scan_links --filesystem a

# (注意，配置也可以是全局设置的，也可以是通过ceph.conf文件设置的。)现在，允许MDS加入恢复文件系统:
$ ceph fs set a joinable true

# 最后，运行前向清除以修复统计信息。确保您有一个MDS正在运行并发出:
$ ceph fs status # get active MDS
$ ceph tell mds.<id> scrub start / recursive repair
a - 0 clients
=
RANK  STATE   MDS     ACTIVITY     DNS    INOS   DIRS   CAPS
 0    active   c   Reqs:    0 /s    13     16     12      0
     POOL        TYPE     USED  AVAIL
cephfs.a.meta  metadata   207k  98.9G
cephfs.a.data    data    14.3M  98.9G
cephfs_recovery - 0 clients
===============
RANK  STATE   MDS     ACTIVITY     DNS    INOS   DIRS   CAPS
 0    active   b   Reqs:    0 /s    10     13     12      0
        POOL            TYPE     USED  AVAIL
cephfs_recovery_meta  metadata  96.0k  98.9G
   cephfs.a.data        data    14.3M  98.9G
STANDBY MDS
     a
```
删除 recovery fs 和 metadata pool
```shell
$ ceph fs fail cephfs_recovery
$ ceph fs rm cephfs_recovery --yes-i-really-mean-it
$ ceph osd pool cephfs_recovery_meta cephfs_recovery_meta --yes-i-really-really-mean-it
$ ceph fs status
RANK  STATE   MDS     ACTIVITY     DNS    INOS   DIRS   CAPS
 0    active   c   Reqs:    0 /s    13     16     12      0
     POOL        TYPE     USED  AVAIL
cephfs.a.meta  metadata   207k  98.9G
cephfs.a.data    data    14.3M  98.9G
STANDBY MDS
     a
     b
```

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io