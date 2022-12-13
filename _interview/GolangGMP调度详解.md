---
titile: Golang GMP调度详解
---

基于CSP并发模型开发了GMP调度器，其中  
G (Goroutine) : 每个 Goroutine 对应一个 G 结构体，G 存储 Goroutine 的运行堆栈、状态以及任务函数。  
M (Machine) : 对OS内核级线程的封装，数量对应真实的CPU数(真正干活的对象)。  
P (Processor): 逻辑处理器,即为G和M的调度对象，用来调度G和M之间的关联关系，其数量可通过 GOMAXPROCS()来设置，默认为核心数。  
在单核情况下，所有Goroutine运行在同一个线程（M0）中，每一个线程维护一个上下文（P），任何时刻，一个上下文中只有一个Goroutine，其他Goroutine在runqueue中等待。

一个Goroutine运行完自己的时间片后，让出上下文，自己回到runqueue中。

当正在运行的G0阻塞的时候（可以需要IO），会再创建一个线程（M1），P转到新的线程中去运行。

当M0返回时，它会尝试从其他线程中“偷”一个上下文过来，如果没有偷到，会把Goroutine放到Global runqueue中去，然后把自己放入线程缓存中。 上下文会定时检查Global runqueue。