---
title: Golang 内存管理
tag: Go
---

Go语言的内存分配器采用了跟 tcmalloc 库相同的多级缓存分配模型，该模型将引入了线程缓存（Thread Cache）、中心缓存（Central Cache）和页堆（Page Heap）三个组件分级管理内存。

![多级缓存](/assets/images/posts/multiLevelCache.png)

线程缓存属于每一个独立的线程，它能够满足线程上绝大多数的内存分配需求，因为不涉及多线程，所以也不需要使用互斥锁来保护内存，这能够减少锁竞争带来的性能损耗。当线程缓存不能满足需求时，运行时会使用中心缓存作为补充解决小对象的内存分配，在遇到大对象时，内存分配器会选择页堆直接分配大内存。

![golang内存模型](/assets/images/posts/go-memory.webp)

在 Golang 中, mcache , mspan , mcentral 和 mheap 是内存管理的四大组件，mspan是内管管理的基本单元，由mcache充当”线程缓存“，由mcentral充当”中心缓存“，由mheap充当“页堆”。下级组件内存不够时向上级申请一个或多个mspan。
根据对象的大小不同，内部会使用不同的内存分配机制，详细参考函数 mallocgo()。  
**<16KB**
会使用微小对象内存分配器从 P 中的 mcache 分配，主要使用 mcache.tinyXXX 这类的字段。  
**16-32KB**
从 P 中的 mcache 中分配。  
**>32KB**
直接从 mheap 中分配。

golang中的内存申请流程如下图所示。

![golang内存管理](/assets/images/posts/go-memory-stack.webp)

大约有 100 种内存块类别，每一个类别都有自己对象的空闲链表。小于 32KB 的内存分配被向上取整到对应的尺寸类别，从相应的空闲链表中分配。一页内存只可以被分裂成同一种尺寸类别的对象，然后由空间链表分配管理器。

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io