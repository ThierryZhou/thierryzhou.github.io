---
title: Ceph 集群 FAQ 总结
tag: ceph
---

## 生产环境如何部署Ceph
### 1. 在kubernetes系统中使用CephCSI时，是Fuse好，还是KernelMod好？
  由于容器在技术文件系统层面不存在隔离，而kubernetes中pod运行时kubelet组件
依赖各种fs相关的工具加载和检测文卷，因此Ceph产生系统异常时，会影响整个工作节点。
因此推荐CephCSI部署是mount方式改为fuse，提升系统稳定性。

### 2. CephFS有没有必要部署为多活(MDS)模式，还需要做其他哪些准备？
  由于CephFS单活模式下，一个MDS服务管理整个文件系统树，部分文件出错会拖累整个
文件系统，因此在条件允许的情况下尽量部署多活(MDS)模式。此外，建议做好metadata
pool的数据备份，

### 3. 编排工具使用cephadm和rook哪个更好，他们之间有什么区别？
cephadm是ceph项目自带的编排工具，部署Ceph这个组件时可以通过修改配置脚本的方式使得各个节点的物理资源得到更加充分的利用，并且当前版本已经使用docker容器来运行ceph组件，对比传统的ansible等运维工具更贴合如今的潮流；rook-ceph则是第三方团队为ceph打造的编码工具，更贴合kubernetes业务，可以使用云原生的可观测组件，对容器、日志、网络等基础设施进行更好的监控，它更适应云上的编排环境，它将ceph组件运行在kubernetes的容器中，后续使用中可依据云厂商提供的富容器增加系统的安全性和健壮性。
 
## 问题处理
#### 1.客户端挂载ceph查看的容量与实际容量不符？如何解决osd之间数据不均衡问题？
  此问题一般是由于部分OSD使用率远高于其他OSD的使用率导致，属于数据平衡性问题的一种。
利用osd utilization命令优化平衡性即可。
```shell
ceph osd reweight-by-utilization
```

#### 2.如何解决osd crash down以后系统中的报警？
```shell
ceph crash  archive-all
```

#### 3.如何解决ceph集群恢复速度慢的问题？
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

#### 4.如何修复有问题的pg？
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

#### 5.如何手动移动pg从osd1到osd2？
```shell
ceph osd pg-upmap-items a.b 1 2
```

#### 6.如何暂停Ceph中正在运行的服务，进行调试或是优化？
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

#### 7.如果关闭一些ceph集群后台执行的任务
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

#### 8. 如何解决部分pg调用ceph osd force-create-pg并且经过长期等待仍无法重建的问题？
```shell
```

#### 9. 如何解决pg长期处于unknown状态？
```shell
# 重建pg对象
ceph osd force-create-pg a.b

# 当 force create 无效时执行下面的命令
ceph osd pg-temp a.b 1 10 14
ceph osd pg-upmap a.b 1 10 14
```

#### 10. 如何显示更详尽的日志以方便调试和追踪问题
```shell
ceph tell osd.0 injectargs --debug-osd 0/5
ceph tell mon.a injectargs --debug-osd 0/5
ceph tell mds.a injectargs --debug-osd 0/5
```

#### 11.如何打开和关闭本地ceph调试日志？
```shell
echo "module libceph +p" >/sys/kernel/debug/dynamic_debug/control
echo "module ceph +p" >/sys/kernel/debug/dynamic_debug/control

echo "module libceph -p" >/sys/kernel/debug/dynamic_debug/control
echo "module ceph -p" >/sys/kernel/debug/dynamic_debug/control
```

#### 12. 修改解决因为rook ceph operator重置后的monitor集群后secret和configmap中mon_host不匹配的问题？
```shell
mon_host=$(kubectl -n rook-ceph get svc rook-ceph-mon-b -o jsonpath='{.spec.clusterIP}')
kubectl -n rook-ceph patch secret rook-ceph-config -p '{"stringData": {"mon_host": "[v2:'"${mon_host}"':3300,v1:'"${mon_host"':6789]", "mon_initial_members": "'"${good_mon_id}"'"}}'
```


### 14.CephFS Monitor出现容量报警。
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
#### 2.解决办法2: 下乡正在产生预警的OSD

```shell
ceph osd.* down
```

#### 3.解决办法3:  增大报警阈值

```shell
ceph config set global mon_data_size_warn 30GiB
```


### 15.Ceph Cluster Monitor出现Slow OPS报警。
```shell
[WRN] SLOW_OPS: 10319 slow ops, oldest one blocked for 736 sec, daemons [mon.c,mon.d,mon.e,mon.h] have slow ops.
```
如果出现Monintor处于正常工作状态中时不存在大量读写的情况，暂停正在进行数据同步的客户端，依次排查其他组件是否出现报警或者频繁的重启及上电。

### 16.Ceph 出现 Operator 修改了 Mon 列表如何处理。

修改ConfigMap rook-ceph-csi-config rook-ceph-mon-endpoints
修改Secret rook-ceph-config