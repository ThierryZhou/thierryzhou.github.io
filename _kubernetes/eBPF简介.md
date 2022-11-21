## BPF概述e
eBPF的演进
最初的[Berkeley Packet Filter (BPF) PDF]是为捕捉和过滤符合特定规则的网络包而设计的，过滤器为运行在基于寄存器的虚拟机上的程序。

在内核中运行用户指定的程序被证明是一种有用的设计，但最初BPF设计中的一些特性却并没有得到很好的支持。例如，虚拟机的指令集架构(ISA)相对落后，现在处理器已经使用64位的寄存器，并为多核系统引入了新的指令，如原子指令XADD。BPF提供的一小部分RISC指令已经无法在现有的处理器上使用。

因此Alexei Starovoitov在eBPF的设计中介绍了如何利用现代硬件，使eBPF虚拟机更接近当代处理器，eBPF指令更接近硬件的ISA，便于提升性能。其中最大的变动之一是使用了64位的寄存器，并将寄存器的数量从2提升到了10个。由于现代架构使用的寄存器远远大于10个，这样就可以像本机硬件一样将参数通过eBPF虚拟机寄存器传递给对应的函数。另外，新增的BPF_CALL指令使得调用内核函数更加便利。

将eBPF映射到本机指令有助于实时编译，提升性能。3.15内核中新增的eBPF补丁使得x86-64上运行的eBPF相比老的BPF(cBPF)在网络过滤上的性能提升了4倍，大部分情况下会保持1.5倍的性能提升。很多架构 (x86-64, SPARC, PowerPC, ARM, arm64, MIPS, and s390)已经支持即时(JIT)编译。

使用eBPF可以做什么？
一个eBPF程序会附加到指定的内核代码路径中，当执行该代码路径时，会执行对应的eBPF程序。鉴于它的起源，eBPF特别适合编写网络程序，将该网络程序附加到网络socket，进行流量过滤，流量分类以及执行网络分类器的动作。eBPF程序甚至可以修改一个已建链的网络socket的配置。XDP工程会在网络栈的底层运行eBPF程序，高性能地进行处理接收到的报文。从下图可以看到eBPF支持的功能：

![eBPF](/assets/images/kubernetes/eBPF-stack.png)

BPF对网络的处理可以分为tc/BPF和XDP/BPF，它们的主要区别如下(参考该文档)：

XDP的钩子要早于tc，因此性能更高：tc钩子使用sk_buff结构体作为参数，而XDP使用xdp_md结构体作为参数，sk_buff中的数据要远多于xdp_md，但也会对性能造成一定影响，且报文需要上送到tc钩子才会触发处理程序。由于XDP钩子位于网络栈之前，因此XDP使用的xdp_buff(即xdp_md)无法访问sk_buff元数据。
```cpp
struct xdp_buff {  /* Linux 5.8*/
	void *data;
	void *data_end;
	void *data_meta;
	void *data_hard_start;
	struct xdp_rxq_info *rxq;
	struct xdp_txq_info *txq;
	u32 frame_sz; /* frame size to deduce data_hard_end/reserved tailroom*/
};

struct xdp_rxq_info {
	struct net_device *dev;
	u32 queue_index;
	u32 reg_state;
	struct xdp_mem_info mem;
} ____cacheline_aligned; /* perf critical, avoid false-sharing */

struct xdp_txq_info {
	struct net_device *dev;
};
```

data指向page中的数据包的起始位置，data_end指向数据包的结尾。由于XDP允许headroom(见下文)，data_hard_start指向page中headroom的起始位置，即，当对报文进行封装时，data会bpf_xdp_adjust_head()通过向data_hard_start移动。相同的BPF辅助函数也可以用以解封转，此时data会远离data_hard_start。

data_meta一开始指向与data相同的位置，但bpf_xdp_adjust_meta() 能够将其朝着 data_hard_start 移动，进而给用户元数据提供空间，这部分空间对内核网络栈是不可见的，但可以被tc BPF程序读取( tc 需要将它从 XDP 转移到 skb)。反之，可以通过相同的BPF程序将data_meta远离data_hard_start来移除或减少用户元数据大小。 data_meta 还可以地单纯用于在尾调用间传递状态，与tc BPF程序访问的skb->cb[]控制块类似。

对于struct xdp_buff中的报文指针，有如下关系 ：data_hard_start <= data_meta <= data < data_end。

rxq字段指向在ring启动期间填充的额外的与每个接受队列相关的元数据。

BPF程序可以检索queue_index，以及网络设备上的其他数据(如ifindex等)。

tc能够更好地管理报文：tc的BPF输入上下文是一个sk_buff，不同于XDP使用的xdp_buff，二者各有利弊。当内核的网络栈在XDP层之后接收到一个报文时，会分配一个buffer，解析并保存报文的元数据，这些元数据即sk_buff。该结构体会暴露给BPF的输入上下文，这样tc ingress层的tc BPF程序就能够使用网络栈从报文解析到的元数据。使用sk_buff，tc可以更直接地使用这些元数据，因此附加到tc BPF钩子的BPF程序可以读取或写入skb的mark，pkt_type， protocol, priority, queue_mapping, napi_id, cb[] array, hash, tc_classid 或 tc_index, vlan metadata等，而XDP能够传输用户的元数据以及其他信息。tc BPF使用的 struct __sk_buff定义在linux/bpf.h头文件中。xdp_buff 的弊端在于，其无法使用sk_buff中的数据，XDP只能使用原始的报文数据，并传输用户元数据。

XDP的能够更快地修改报文：sk_buff包含很多协议相关的信息(如GSO阶段的信息)，因此其很难通过简单地修改报文数据达到切换协议的目的，原因是网络栈对报文的处理主要基于报文的元数据，而非每次访问数据包内容的开销。因此，BPF辅助函数需要正确处理内部sk_buff的转换。而xdp_buff 则不会有这种问题，因为XDP的处理时间早于内核分配sk_buff的时间，因此可以简单地实现对任何报文的修改(但管理起来要更加困难)。

tc/ebpf和xdp可以互补：如果用户需要修改报文，同时对数据进行比较复杂的管理，那么，可以通过运行两种类型的程序来弥补每种程序类型的局限性。XDP程序位于ingress，可以修改完整的报文，并将用户元数据从XDP BPF传递给tc BPF，然后tc可以使用XDP的元数据和sk_buff字段管理报文。

tc/eBPF可以作用于ingress和egress，但XDP只能作用于ingress：与XDP相比，tc BPF程序可以在ingress和egress的网络数据路径上触发，而XDP只能作用于ingress。

tc/BPF不需要改变硬件驱动，而XDP通常会使用native驱动模式来获得更高的性能。但tc BPF程序的处理仍作用于早期的内核网络数据路径上(GRO处理之后，协议处理和传统的iptables防火墙的处理之前，如iptables PREROUTING或nftables ingress钩子等)。而在egress上，tc BPF程序在将报文传递给驱动之前进行处理，即在传统的iptables防火墙(如iptables POSTROUTING)之后，但在内核的GSO引擎之前进行处理。一个特殊情况是，如果使用了offloaded的tc BPF程序(通常通过SmartNIC提供)，此时Offloaded tc/eBPF接近于Offloaded XDP的性能。

从下图可以看到TC和XDP的工作位置，可以看到XDP对报文的处理要先于TC：

![eBPF-packet](/assets/images/kubernetes/eBPF-packet.png)

内核执行的另一种过滤类型是限制进程可以使用的系统调用。通过seccomp BPF实现。

eBPF也可以用于通过将程序附加到tracepoints, kprobes,和perf events的方式定位内核问题，以及进行性能分析。因为eBPF可以访问内核数据结构，开发者可以在不编译内核的前提下编写并测试代码。对于工作繁忙的工程师，通过该方式可以方便地调试一个在线运行的系统。此外，还可以通过静态定义的追踪点调试用户空间的程序(即BCC调试用户程序，如Mysql)。

使用eBPF有两大优势：快速，安全。为了更好地使用eBPF，需要了解它是如何工作的。

内核的eBPF校验器
在内核中运行用户空间的代码可能会存在安全和稳定性风险。因此，在加载eBPF程序前需要进行大量校验。首先通过对程序控制流的深度优先搜索保证eBPF能够正常结束，不会因为任何循环导致内核锁定。严禁使用无法到达的指令；任何包含无法到达的指令的程序都会导致加载失败。

第二个阶段涉及使用校验器模拟执行eBPF程序(每次执行一个指令)。在每次指令执行前后都需要校验虚拟机的状态，保证寄存器和栈的状态都是有效的。严禁越界(代码)跳跃，以及访问越界数据。

校验器不会检查程序的每条路径，它能够知道程序的当前状态是否是已经检查过的程序的子集。由于前面的所有路径都必须是有效的(否则程序会加载失败)，当前的路径也必须是有效的，因此允许验证器“修剪”当前分支并跳过其模拟阶段。

校验器有一个"安全模式"，禁止指针运算。当一个没有CAP_SYS_ADMIN特权的用户加载eBPF程序时会启用安全模式，确保不会将内核地址泄露给非特权用户，且不会将指针写入内存。如果没有启用安全模式，则仅允许在执行检查之后进行指针运算。例如，所有的指针访问时都会检查类型，对齐和边界冲突。

无法读取包含未初始化内容的寄存器，尝试读取这类寄存器中的内容将导致加载失败。R0-R5的寄存器内容在函数调用期间被标记未不可读状态，可以通过存储一个特殊值来测试任何对未初始化寄存器的读取行为；对于读取堆栈上的变量的行为也进行了类似的检查，确保没有指令会写入只读的帧指针寄存器。

最后，校验器会使用eBPF程序类型(见下)来限制可以从eBPF程序调用哪些内核函数，以及访问哪些数据结构。例如，一些程序类型可以直接访问网络报文。

bpf()系统调用
使用bpf()系统调用和BPF_PROG_LOAD命令加载程序。该系统调用的原型为：
```cpp
int bpf(int cmd, union bpf_attr *attr, unsigned int size);
```

bpf_attr允许数据在内核和用户空间传递，具体类型取决于cmd参数。

cmd可以是如下内容：

```cpp
       BPF_MAP_CREATE
              Create a map and return a file descriptor that refers to the
              map.  The close-on-exec file descriptor flag (see fcntl(2)) is
              automatically enabled for the new file descriptor.

       BPF_MAP_LOOKUP_ELEM
              Look up an element by key in a specified map and return its
              value.

       BPF_MAP_UPDATE_ELEM
              Create or update an element (key/value pair) in a specified
              map.

       BPF_MAP_DELETE_ELEM
              Look up and delete an element by key in a specified map.

       BPF_MAP_GET_NEXT_KEY
              Look up an element by key in a specified map and return the
              key of the next element.

       BPF_PROG_LOAD
              Verify and load an eBPF program, returning a new file descrip‐
              tor associated with the program.  The close-on-exec file
              descriptor flag (see fcntl(2)) is automatically enabled for
              the new file descriptor.
```
