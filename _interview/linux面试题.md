---
title:  "操作系统面试题"
tag:    interview
---
## Linux 面试题

### 体系结构
##### 1. Linux 的体系结构

从大的方面讲，Linux 体系结构可以分为两块：  
![linux-stack](/assets/images/posts/linux-stack.png)  
用户空间(User Space) ：用户空间又包括用户的应用程序(User Applications)、C 库(C Library) 。  
内核空间(Kernel Space) ：内核空间又包括系统调用接口(System Call Interface)、内核(Kernel)、平台架构相关的代码(Architecture-Dependent Kernel Code) 。

Linux内核主要包括由5个子系统组成：进程调度(SCHED)，内存管理(MM)，虚拟文件系统(VFS)，网络接口(NET) ，进程间通信(IPC)。

### VFS(虚拟文件系统)

##### 1. Linux 磁盘 I/O 的三种方式对比
标准I/O、直接 I/O、mmap

##### 2. Linux Inode和Dentry

dentry 保存文件和目录的名称和相互之间的包含关系， inode 节点表将文件的逻辑结构和物理结构进行转换。

inode 节点是一个 64 字节长的表，表中包含了文件的相关信息，其中有文件的大小、文件所有者、文件的存取许可方式以及文件的类型等重要信息。在 inode 节点表中最重要的内容是磁盘地址表。在磁盘地址表中有 13 个块号，文件将以块号在磁盘地址表中出现的顺序依次读取相应的块。

Linux 文件系统通过把 inode 节点和文件名进行连接，当需要读取该文件时，文件系统在当前目录表中查找该文件名对应的项，由此得到该文件相对应的 inode 节点号，通过该 inode 节点的磁盘地址表把分散存放的文件物理块连接成文件的逻辑结构。

##### 3. 什么是硬链接和软链接？

硬链接：本质是一个 dentry 资源，但是其所对应的 inode 复用原始文件的 inode 资源。

软链接：在文件系统中新建一个链接文件，并将其内容设置为原始文件绝对路径或者相对路径，当链接文件被访问时会请求会被重定向到原始文件。

##### 4. Linux虚拟文件系统结构

super_block 超级块

inode 索引节点

dentry 目录项

file 文件

##### 5. Linux 中的文件包括哪些？
可执行文件，普通文件，目录文件，链接文件，设备文件，管道文件

##### 6.  TLB 中缓存的是什么内容

translation lookaside buffer, 也叫快表，用作页表缓冲。记录虚拟地址和物理地址的对应关系，用于加快地址转换。

##### 7. Linux 中有哪几种设备？

字符设备和块设备

### MMU(存储管理单元)

##### 1. malloc、vmalloc 和 kmalloc 有什么区别？
malloc 用户空间下的内存管理接口，保证的是在虚拟地址空间上的连续。stack 和 heap 中间。小于128M的通过brk申请，大于的通过 mmap 申请。

vmalloc 用于申请大块内存，虚拟地址连续，物理地址不一定连续，不能直接用于DMA，在进程地址空间有专门的一块。对应释放函数 vfree()。

kmalloc 用于申请小内存，由 slab 管理实现，一般至少小于4KB（page）。不能申请大于128K的数据。物理地址和虚拟地址都连续，可用于DMA操作。

##### 2. Linux 内核空间布局

x86架构中将内核地址空间划分三部分：ZONE_DMA、ZONE_NORMAL和 ZONE_HIGHMEM。ZONE_HIGHMEM即为高端内存，这就是内存高端内存概念的由来。

ZONE_DMA 内存开始的16MB

ZONE_NORMAL 16MB~896MB

ZONE_HIGHMEM 896MB ~ 结束（1G）

当内核想访问高于896MB物理地址内存时，从0xF8000000 ~ 0xFFFFFFFF地址空间范围内找一段相应大小空闲的逻辑地址空间，借用一会。借用这段逻辑地址空间，建立映射到想访问的那段物理内存（即填充内核PTE页面表），临时用一会，用完后归还。这样别人也可以借用这段地址空间访问其他物理内存，实现了使用有限的地址空间，访问所有所有物理内存。

![kernel] (/assets/images/linux/kernel-memory.png)

用户进程没有高端内存概念。只有在内核空间才存在高端内存。用户进程最多只可以访问3G物理内存，而内核进程可以访问所有物理内存。

目前现实中，64位Linux内核不存在高端内存，因为64位内核可以支持超过512GB内存。若机器安装的物理内存超过内核地址空间范围，就会存在高端内存。

##### 3. Linux 用户内存空间布局

![user](/assets/images/linux/user-memory.png)

text段:就是放程序代码的,编译时确定,只读。

data段:存放在编译阶段(而非运行时)就能确定的数据,可读可写就是通常所说的静态存储区,赋了初值的全局变量和静态变量存放在这个域,常量也存放在这个区域。

rdata段：rdata是用来存放只读初始化变量的，当我们在源程序中的变量前面加了const后，编译器知道个字符串是永远不会改变的，或说是只读的，所以将其分配到.rdata段中。

bss段：定义而没有赋初值的全局变量和静态变量,放在这个区域

栈区（stack）：由编译器自动分配释放 ，存放函数的参数值，局部变量的值等。从高地址向地地址生长。

堆区（heap）：般由程序员分配释放， 若程序员不释放，程序结束时可能由OS回收 。从地地址想高地址生长。

程序内存段和进程地址空间中的内存区域是种模糊对应，也就是说，堆、bss、数据段（初始化过的）都在进程空间中由数据段内存区域表示。

##### 4. Linux 的内核空间和用户空间如何划分的？进程地址空间布局图？

32位可配置3G/1G, 2G/2G，一般是两级页表

64位可配置几级页表，一般可选3级/4级页表，256G/256G，或512T/512T

##### 5. 伙伴系统申请内存的函数有哪些？

alloc_page(gfp_mask, order)

__get_free_pages(gfp_mask, order)

##### 6. 通过 slab 分配器申请内存的函数有哪些？
自己构造对象：kmem_cache_create/kmem_cache_alloc

普通匿名内存申请：kmalloc


##### 7. 在支持并使能 MMU 的系统中，Linux 内核和用于程序分别运行在物理地址模式还是虚拟地址模式？
都运行在虚拟地址模式，页表转换对应由硬件单元MMU完成。

### SCHED(进程管理)

##### 1. 进程内存的分配与回收

创建进程fork()、程序载入execve()、映射文件mmap()、动态内存分配malloc()/brk()等进程相关操作都需要分配内存给进程。不过这时进程申请和获得的还不是实际内存，而是虚拟内存，准确的说是“内存区域”。进程对内存区域的分配最终都会归结到do_mmap（）函数上来（brk调用被单独以系统调用实现，不用do_mmap()），
内核使用do_mmap()函数创建一个新的线性地址区间。但是说该函数创建了一个新VMA并不非常准确，因为如果创建的地址区间和一个已经存在的地址区间相邻，并且它们具有相同的访问权限的话，那么两个区间将合并为一个。如果不能合并，那么就确实需要创建一个新的VMA了。但无论哪种情况， do_mmap()函数都会将一个地址区间加入到进程的地址空间中－－无论是扩展已存在的内存区域还是创建一个新的区域。
同样，释放一个内存区域应使用函数do_ummap()，它会销毁对应的内存区域。 

##### 2. 创建进程的系统调用有哪些？
clone, fork, vfork

##### 3. 调用 schedule() 进行进程切换的方式有几种？
do_fork/do_timer/wake_up_process/setscheduler/sys_sched_yield

##### 4. Linux 调度程序是根据进程的动态优先级还是静态优先级来调度进程的？
cfs 会计算虚拟时间，还有一个计算出来的优先级。

##### 5. Linux 主要有哪几种内核锁？Linux 内核的同步机制是什么？

自旋锁：自旋锁的主要特征是使用者在想要获得临界区执行权限时，如果临界区已经被加锁，那么自旋锁并不会阻塞睡眠，等待系统来主动唤醒，而是原地忙轮询资源是否被释放加锁。(spin_lock)

信号量：semxxx down/up write/read

互斥锁：加锁后，任何其他试图再次加锁的线程会被阻塞，直到当前进程解锁。

读写锁：读写锁也叫共享互斥锁：读模式共享和写模式互斥，本质上这种非常合理，因为在数据没有被写的前提下，多个使用者读取时完全不需要加锁的。  

RCU(read-copy update): 读写锁的扩展版本，简单来说就是支持多读多写同时加锁，多读没什么好说的，但是对于多写同时加锁，还是存在一些技术挑战的。

### NET(网络模块)

### IPC(进程间通信)

##### 1. 设备驱动程序包括哪些功能函数？

open/read/write/ioctl/release/llseek

##### 2. 如何唯一标识一个设备？

主设备号和次设备号。dev_t，12位表示主设备号，20位表示次设备号。

MKDEV(int major, int minor)用于生产一个 dev_t 类型的对象。

##### 3. Linux 通过什么方式实现系统调用？

软件中断。系统调用编号，异常处理程序

##### 4. Linux 软中断和工作队列的作用是什么？

软中断：不可睡眠阻塞，处于中断上下文，不能进程切换，不能被自己打断。

工作队列：处理进程上下文中，可以睡眠阻塞。

##### 5. 进程间通信主要有哪几种方式？

1. 管道：两个进程需要有共同的祖先，pipe/popen  
2. 命名管道：两个进程可以无关  
3. 信号  
4. 消息队列  
5. 共享内存  
6. 信号量  
7. 套接字