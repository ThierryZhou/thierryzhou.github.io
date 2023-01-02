---
titile: Golang内存管理之GC
excerpt: Golang在GC的演进过程中也经历了很多次变革，Go V1.3之前的标记-清除(mark and sweep)算法，Go V1.3之前的标记-清扫(mark and sweep)的缺点
---

## 概述

垃圾回收(Garbage Collection，简称GC)是编程语言中提供的自动的内存管理机制，自动释放不需要的内存对象，让出存储器资源。GC过程中无需程序员手动执行。GC机制在现代很多编程语言都支持，GC能力的性能与优劣也是不同语言之间对比度指标之一。

Golang在GC的演进过程中也经历了很多次变革

Go V1.3 之前使用的是标记-清除(mark and sweep)算法

Go V1.5 开始使用三色并发标记法

Go V1.8 使用三色标记法+混合写屏障机制

## 标记-清扫算法 (mark-sweep)

标记清除算法是最常见的垃圾收集算法，标记清除收集器是跟踪式垃圾收集器，其执行过程可以分成标记(Mark)和清除(Sweep)，算法流程如下：

1. 标记阶段：暂停应用程序的执行，从根对象触发查找并标记堆中所有存活的对象；
2. 清除阶段：遍历堆中的全部对象，回收未被标记的垃圾对象并将回收的内存加入空闲链表，恢复应用程序的执行；

操作非常简单，但是有一点需要额外注意：mark and sweep 算法在执行的时候，需要程序暂停(stop the world)。

示例：

![mark-sweep](/assets/images/posts/mark-sweep.png)

## 三色标记算法

原始标记清除算法带来的长时间STW, 为了解决这一问题，Go从V1.5版本实现了基于三色标记清除的并发垃圾收集算法，在不暂停程序的情况下即可完成对象的可达性分析，三色标记算法将程序中的对象分成白色、黑色和灰色三类，算法流程如下：

1. 遍历根对象的第一层可达对象标记为灰色, 不可达默认白色。
2. 将灰色对象的下一层可达对象标记为灰色, 自身标记为黑色。
3. 多次重复步骤2, 直到灰色对象为0, 只剩下白色对象和黑色对象。
4. 回收白色对象。

示例：
1. 遍历根对象的第一层可达对象标记为灰色, 不可达默认白色
![3color-flow-1](/assets/images/posts/3color-flow-1.png)

2. 将灰色对象 A 的下一层可达对象标记为灰色, 自身标记为黑色
![3color-flow-2](/assets/images/posts/3color-flow-2.png)

3. 继续遍历灰色对象的下层对象,重复步骤2
![3color-flow-3](/assets/images/posts/3color-flow-3.png)

4. 继续遍历灰色对象的下层对象,重复步骤2
![3color-flow-4](/assets/images/posts/3color-flow-4.png)

5. 扫描结束后，回收所有白色的节点。

### 三色标记算法的问题

假如没有STW，那么也就不会再存在性能上的问题，那么接下来我们假设如果三色标记法不加入STW会发生什么事情？

我们还是基于上述的三色并发标记法来说, 他是一定要依赖STW的. 因为如果不暂停程序, 程序的逻辑改变对象引用关系, 这种动作如果在标记阶段做了修改，会影响标记结果的正确性，我们来看看一个场景，如果三色标记法, 标记过程不使用STW将会发生什么事情?

我们把初始状态设置为已经经历了第一轮扫描，目前黑色的有对象1和对象4， 灰色的有对象2和对象7，其他的为白色对象，且对象2是通过指针p指向对象3的，如图所示。

![problem-1](/assets/images/posts/gc-problem-1.png)

现在如何三色标记过程不启动STW，那么在GC扫描过程中，任意的对象均可能发生读写操作，如图所示，在还没有扫描到对象2的时候，已经标记为黑色的对象4，此时创建指针q，并且指向白色的对象3。

![problem-2](/assets/images/posts/gc-problem-2.png)

与此同时灰色的对象2将指针p移除，那么白色的对象3实则就是被挂在了已经扫描完成的黑色的对象4下，如图所示。

![problem-3](/assets/images/posts/gc-problem-3.png)

然后我们正常指向三色标记的算法逻辑，将所有灰色的对象标记为黑色，那么对象2和对象7就被标记成了黑色，如图所示。

![problem-4](/assets/images/posts/gc-problem-4.png)

那么就执行了三色标记的最后一步，将所有白色对象当做垃圾进行回收，如图所示。

![problem-5](/assets/images/posts/gc-problem-5.png)

但是最后我们才发现，本来是对象4合法引用的对象3，却被GC给“误杀”回收掉了。

可以看出，有两种情况，在三色标记法中，是不希望被发生的。

条件1: 一个白色对象被黑色对象引用(白色被挂在黑色下)

条件2: 灰色对象与它之间的可达关系的白色对象遭到破坏(灰色同时丢了该白色)

如果当以上两个条件同时满足时，就会出现对象丢失现象! 在 Golang 比较早期的版本中，是使用 STW 的方案来保证一致性，这样做的坏处是效率非常低。


### 三色一致性

STW的过程有明显的资源浪费，对所有的用户程序都有很大影响。那么是否可以在保证对象不丢失的情况下合理的尽可能的提高GC效率，减少STW时间呢？

目前 Golang 使用通过引入三色一致性的机制，尝试去破坏上面的两个必要条件就可以了，分为强三色一致性和弱三色一致性。

强三色不变性（strong tri-color invariant）：黑色对象不会指向白色对象，只会指向灰色对象或者黑色对象。

![consistent-1](/assets/images/posts/consistent-1.png)

弱三色不变性（weak tri-color invariant）：即便黑色对象指向白色对象，那么从灰色对象出发，总存在一条可以找到该白色对象的路径。

![consistent-2](/assets/images/posts/consistent-2.png)

## 写屏障

Golang 中使用三色一致性的方法是引入一个叫做写屏障的机制，来完成三色一致性，写屏障机制分为插入屏障和删除屏障。

##### 插入屏障

具体操作: 在A对象引用B对象的时候，B对象被标记为灰色。(将B挂在A下游，B必须被标记为灰色)

##### 删除屏障

具体操作: 被删除的对象，如果自身为灰色或者白色，那么被标记为灰色。

满足: 弱三色不变式. (保护灰色对象到白色对象的路径不会断)

## 混合写屏障

插入写屏障和删除写屏障的虽然大大的缩短的系统 GC 的 STW 时间，但是也有其短板：

1. 插入写屏障：结束时需要 STW 来重新扫描栈，标记栈上引用的白色对象的存活；
2. 删除写屏障：回收精度低， GC 开始时 STW 扫描堆栈来记录初始快照，这个过程会保护开始时刻的所有存活对象。

Go V1.8 版本引入了混合写屏障机制（hybrid write barrier），避免了对栈re-scan的过程，极大的减少了STW的时间。

具体操作:

1. GC开始将栈上的对象全部扫描并标记为黑色(之后不再进行第二次重复扫描，无需STW)，
2. GC期间，任何在栈上创建的新对象，均为黑色。
3. 被删除的对象标记为灰色。
4. 被添加的对象标记为灰色。

## GC 工作机制

#### GC 触发时机

在 Go 中主要会在三个地方触发 GC：

1. 监控线程 runtime.sysmon 定时调用；
2. 手动调用 runtime.GC 函数进行垃圾收集；
3. 申请内存时 runtime.mallocgc 会根据堆大小判断是否调用；

#### GC 流程分析

当 GC 被触发后，Golang 开始执行 GC 循环，循环分为四个阶段分别是：清理终止，标记，标记终止，清理。

需要注意的是实际 runtime 的源码中只定义了 _GCoff / _GCmark / _GCmarktermination 三种状态，GC关闭、清理终止和清理都对应 _GCoff 这一状态。

**清理终止(sweep termination)**

会触发 STW ，所有的 P（处理器） 都会进入 safe-point（安全点）；清理未被清理的内存对象.

**标记阶段(the mark phase)**

将 GC 状态 从 _GCoff  改成 _GCmark，开启 Write Barrier （写入屏障）、mutator assists（协助线程），将根对象入队；恢复程序执行，mark workers（标记进程）和 mutator assists（协助线程）会开始并发标记内存中的对象。对于任何指针写入和新的指针值，都会被写屏障覆盖，而所有新创建的对象都会被直接标记成黑色；GC 执行根节点的标记，这包括扫描所有的栈、全局对象以及不在堆中的运行时数据结构。扫描goroutine 栈绘导致 goroutine 停止，并对栈上找到的所有指针加置灰，然后继续执行 goroutine。

GC 在遍历灰色对象队列的时候，会将灰色对象变成黑色，并将该对象指向的对象置灰；
GC 会使用分布式终止算法（distributed termination algorithm）来检测何时不再有根标记作业或灰色对象，如果没有了 GC 会转为mark termination（标记终止），

**标记终止(mark termination)**

STW，然后将 GC 阶段转为 _GCmarktermination,关闭 GC 工作线程以及 mutator assists（协助线程）；执行清理，如 flush mcache。

**清理阶段(the sweep phase)**

将 GC 状态转变至 _GCoff，初始化清理状态并关闭 Write Barrier（写入屏障）；恢复程序执行，从此开始新创建的对象都是白色的；后台并发清理所有的内存管理单元。

### 源码分析

```go
func GC() {
	// 等待上一轮 GC 循环结束
	n := atomic.Load(&work.cycles)
	gcWaitOnMark(n)

	// 先完成第 N 轮 GC 循环，然后触发第 N + 1 轮 GC 循环
	gcStart(gcTrigger{kind: gcTriggerCycle, n: n + 1})

	// 等待第 N + 1 轮 GC 循环的 mark termination 结束
	gcWaitOnMark(n + 1)

	// 清理未完成前，先出让当前执行机会
	for atomic.Load(&work.cycles) == n+1 && sweepone() != ^uintptr(0) {
		sweep.nbgsweep++
		Gosched()
	}

	for atomic.Load(&work.cycles) == n+1 && !isSweepDone() {
		Gosched()
	}

	// 完成清理后，进入标记阶段
	mp := acquirem()
	cycle := atomic.Load(&work.cycles)
	if cycle == n+1 || (gcphase == _GCmark && cycle == n+2) {
		mProf_PostSweep()
	}
	releasem(mp)
}
```

### 关于 GC 的一些经验总结

事实上，虽然 Golang 的内存管理和 GC 机制已经非常完善并尽可能减少其资源消耗，但是当你的程序处于一些极端的负载中，这种编译器托管方式仍免不了性能下降，因此在传统非 GC 语言中总结的一些经验诸如尽可能的复用变量，尽量使用指针传值代替变量复制传值等依然适用。

Golang 特有的内存逃逸的问题可能是你的程序性能表现不佳的元凶；另外，由于 Golang 中 string 类型底层是常量数组，对它的修改也可能让你的程序性能极差。

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io