---
layout: post
title:  "GPU 虚拟化"
date:   2022-10-24 12:00
---

# GPU 多场景混部介绍

## 在涂鸦，GPU的主要使用场景

#### 1. 实验场景

#### 该场景特点是用戶会⻓期占有GPU资源，但仅偶尔调试使用，资源利用率低。

#### 我们主要的训练场景为单机多卡，所以用戶调试的时候也需要多卡环境，即我们期望的是能够分

```
配n个0.x卡。
目前开源的GPU虚拟化方案仅支持分配单个0.x卡，不支持分配n个0.x卡，所以我们需要对开
源方案进行改造。
2. 训练场景
该场景负载高，希望能够独占整张显卡。
3. 推理场景
即允许单张显卡上部署多个应用。
```
## GPU虚拟化的常⻅方案

#### 分资源隔离和不隔离两种

```
资源不隔离：代表项目为阿里的GPU-Share，即简单的将单个GPU分配给多个POD，每个POD实际都
能使用全部的GPU资源，需要应用自身根据环境变量中的值自己限制显存的使用。
资源隔离：
驱动劫持，劫持libcuda.so 中的API来实现资源隔离，代表项目为腾讯的vcuda，每个POD只
能使用预先分配的算力和显存。
内核劫持，腾讯和阿里目前主推的模式，但不开源，只能在各自的云环境中才能使用。
硬件层隔离，包括MIG和vGPU等，英伟达官方提供的虚拟化方案，但仅部分高端型号支持，且需
要额外付费。
```
## 我们的改进目标

#### 我们希望达成以下目标：

#### 1. 允许用戶分配两张半卡，即两张GPU可以同时给两个用戶使用，且各自只能使用一半显存，算力不做限

#### 制。

#### 2. 支持三种常⻅的虚拟化方案，且能够混部。

#### 3. 优先将实验和推理场景的任务分配到算力低的GPU节点上。

#### 如此不仅可以提高资源利用率，还能大幅降低运维难度。实现上，包括三个部分：

```
1. vcuda用于劫持libcuda.so，以实现资源隔离。
2. DevicePlugin实现资源注册和分配。
3. Scheduler扩展插件实现POD的正确和优化调度。
```

## 两个半卡的实现

### 1. 资源隔离的实现

```
我们基于腾讯开源的vcuda-controller 进行改造，移除对算力的限制，在显存分配相关的API中插
入对显存的检查，会获取当前POD所有的进程ID，以及显卡中所有进程的ID和已使用显存量，对比得
出当前POD所占用的全部显存资源，从而确定是否已会超额。
原项目中始终假设为第 1 块GPU，而我们改为会查询当前实际的GPU设备ID，从而支持多GPU的显存
隔离。
vcuda-controller项目自 20 年初就很少更新了，我们进行了API更新、加入缓存、直接通过socket
与deviceplugin通信等改造。
```
### 2. DevicePlugin的实现

```
对每张物理GPU，DevicePlugin会向kubelet注册两份名为tuya.com/sgpu的资源。
分别两张半卡时，会向两张物理GPU各取一份tuya.com/sgpu资源。
如果一个节点只剩 2 份tuya.com/sgpu资源，且在同一张物理GPU上，这时分配两张半卡会失败，为
避免这种情况，需要添加scheduler插件，对节点进行过滤。
```
## 混部方案的实现

```
1. 假如一个节点有 4 张物理GPU，DevicePlugin会向kubelet同时注册以下三种资源：
```
```
2. 假如这时有一个声明了tuya.com/sgpu: 2的POD分配到了该节点，则DevicePlugin会将其中 2 张
物理GPU分配给该POD，同时这 2 张物理GPU锁定为只允许分配tuya.com/sgpu资源，而从
nvidia.com/gpu和tuya.com/vcuda-memory中剔除，变成：
```
```
Capacity:
nvidia.com/gpu: 4
tuya.com/sgpu: 8
tuya.com/vcuda-memory: 344
Allocatable:
nvidia.com/gpu: 4
tuya.com/sgpu: 8
tuya.com/vcuda-memory: 344
Allocated resources:
Resource Requests Limits
-------- -------- ------
nvidia.com/gpu 0 0
tuya.com/sgpu 0 0
tuya.com/vcuda-memory 0 0
```
```
1 2 3 4 5 6 7 8 9
```
```
10
11
12
13
14
```
```
Capacity:
nvidia.com/gpu: 2
tuya.com/sgpu: 8
tuya.com/vcuda-memory: 172
```
```
1
2
3
4
```

这里nvidia.com/gpu和tuya.com/vcuda-memory都减半了，tuya.com/sgpu不变还是等于 8 ，所以这
里的Allocatable是指所有可分配的资源，包括已分配的。

```
3. 假如这时再有一个声明了tuya.com/vcuda-memory: 10的POD分配到该节点，则会再有一个物理
GPU被锁定，变成：
```
```
nvidia.com/gpu和tuya.com/sgpu减少，而tuya.com/vcuda-memory保持不变。
4. 这时假如再分配一个独占GPU，则变成：
```
```
Allocatable:
nvidia.com/gpu: 2
tuya.com/sgpu: 8
tuya.com/vcuda-memory: 172
Allocated resources:
Resource Requests Limits
-------- -------- ------
nvidia.com/gpu 0 0
tuya.com/sgpu 2 2
tuya.com/vcuda-memory 0 0
```
```
5
6
7
8
9
10
11
12
13
14
```
```
Capacity:
nvidia.com/gpu: 1
tuya.com/sgpu: 6
tuya.com/vcuda-memory: 172
Allocatable:
nvidia.com/gpu: 1
tuya.com/sgpu: 6
tuya.com/vcuda-memory: 172
Allocated resources:
Resource Requests Limits
-------- -------- ------
nvidia.com/gpu 0 0
tuya.com/sgpu 2 2
tuya.com/vcuda-memory 10 10
```
```
1 2 3 4 5 6 7 8 9
```
```
10
11
12
13
14
```
```
Capacity:
nvidia.com/gpu: 1
tuya.com/sgpu: 4
tuya.com/vcuda-memory: 86
Allocatable:
nvidia.com/gpu: 1
tuya.com/sgpu: 4
tuya.com/vcuda-memory: 86
Allocated resources:
Resource Requests Limits
-------- -------- ------
nvidia.com/gpu 1 1
tuya.com/sgpu 2 2
tuya.com/vcuda-memory 10 10
```
```
1 2 3 4 5 6 7 8 9
```
```
10
11
12
13
14
```

#### 5. 当POD被删除时，与上同理把相关资源加回去即可。

## 调度器扩展插件

#### 扩展插件主要实现两个接口：

#### 1. 过滤节点：需要对符合要求的节点进行过滤，尤其是资源看起来够，实际无法分配的节点。

#### 2. 节点打分：包括以下规则等

```
优先将tuya.com/sgpu和tuya.com/vcuda-memory分配到算力低的节点
优先将tuya.com/sgpu分配到已分配过tuya.com/sgpu的节点
优先凑整
```
而后在kube-scheduler-config.yaml中加入该插件的地址即可

#### 最终分配案例：

```
apiVersion: kubescheduler.config.k8s.io/v1beta
kind: KubeSchedulerConfiguration
clientConnection:
kubeconfig: /etc/kubernetes/scheduler.conf
leaderElection:
leaderElect: true
extenders:
```
- urlPrefix: "http://nvidia-device-scheduler.kube-system.svc/scheduler"
filterVerb: "filter"
prioritizeVerb: "prioritize"
weight: 1
enableHTTPS: false
nodeCacheCapable: false
ignorable: false
managedResources:
- name: "nvidia.com/gpu"
ignoredByScheduler: false
- name: "tuya.com/sgpu"
ignoredByScheduler: false
- name: "tuya.com/vcuda-core"
ignoredByScheduler: false
- name: "tuya.com/vcuda-memory"
ignoredByScheduler: false

```
1 2 3 4 5 6 7 8 9
```
```
10
11
12
13
14
15
16
17
18
19
20
21
22
23
```

#### 实验和推理任务都分配在比较旧的GPU上，并尽可能优先分配满一个节点，再分配下一个节点


