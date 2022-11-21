## 概述

Informer 是 client-go 中的核心工具包，已经被 kubernetes 中众多组件所使用。所谓 Informer，其实就是一个带有本地缓存和索引机制的、可以注册 EventHandler 的 client，本地缓存被称为 Store，索引被称为 Index。使用 informer 的目的是为了减轻 apiserver 数据交互的压力而抽象出来的一个 cache 层, 客户端对 apiserver 数据的 "读取" 和 "监听" 操作都通过本地 informer 进行。

Informer 的主要功能：

同步数据到本地缓存

根据对应的事件类型，触发事先注册好的 ResourceEventHandle

为什么需要 Informer 机制？
我们知道Kubernetes各个组件都是通过REST API跟API Server交互通信的，而如果每次每一个组件都直接跟API Server交互去读取/写入到后端的etcd的话，会对API Server以及etcd造成非常大的负担。 而Informer机制是为了保证各个组件之间通信的实时性、可靠性，并且减缓对API Server和etcd的负担。

Informer 需要满足哪些要求？
消息可靠性

消息实时性

消息顺序性

高性能


## 工作流程

Informer 首先会 list/watch apiserver，Informer 所使用的 Reflector 包负责与 apiserver 建立连接，Reflector 使用 ListAndWatch 的方法，会先从 apiserver 中 list 该资源的所有实例，list 会拿到该对象最新的 resourceVersion，然后使用 watch 方法监听该 resourceVersion 之后的所有变化，若中途出现异常，reflector 则会从断开的 resourceVersion 处重现尝试监听所有变化，一旦该对象的实例有创建、删除、更新动作，Reflector 都会收到"事件通知"，这时，该事件及它对应的 API 对象这个组合，被称为增量（Delta），它会被放进 DeltaFIFO 中。

Informer 会不断地从这个 DeltaFIFO 中读取增量，每拿出一个对象，Informer 就会判断这个增量的时间类型，然后创建或更新本地的缓存，也就是 store。

如果事件类型是 Added（添加对象），那么 Informer 会通过 Indexer 的库把这个增量里的 API 对象保存到本地的缓存中，并为它创建索引，若为删除操作，则在本地缓存中删除该对象。

DeltaFIFO 再 pop 这个事件到 controller 中，controller 会调用事先注册的 ResourceEventHandler 回调函数进行处理。

在 ResourceEventHandler 回调函数中，其实只是做了一些很简单的过滤，然后将关心变更的 Object 放到 workqueue 里面。

Controller 从 workqueue 里面取出 Object，启动一个 worker 来执行自己的业务逻辑，业务逻辑通常是计算目前集群的状态和用户希望达到的状态有多大的区别，然后孜孜不倦地让 apiserver 将状态演化到用户希望达到的状态，比如为 deployment 创建新的 pods，或者是扩容/缩容 deployment。

在worker中就可以使用 lister 来获取 resource，而不用频繁的访问 apiserver，因为 apiserver 中 resource 的变更都会反映到本地的 cache 中。

List & Watch

![worflow](/assets/images/kubernetes/informer-workflow.webp]

List所做的，就是向API Server发送一个http短链接请求，罗列所有目标资源的对象。而Watch所做的是实际的“监听”工作，通过http长链接的方式，其与API Server能够建立一个持久的监听关系，当目标资源发生了变化时，API Server会返回一个对应的事件，从而完成一次成功的监听，之后的事情便交给后面的handler来做。

这样一个List & Watch机制，带来了如下几个优势：

事件响应的实时性：通过Watch的调用，当API Server中的目标资源产生变化时，能够及时的收到事件的返回，从而保证了事件响应的实时性。而倘若是一个轮询的机制，其实时性将受限于轮询的时间间隔。

事件响应的可靠性：倘若仅调用Watch，则如果在某个时间点连接被断开，就可能导致事件被丢失。List的调用带来了查询资源期望状态的能力，客户端通过期望状态与实际状态的对比，可以纠正状态的不一致。二者结合保证了事件响应的可靠性。

高性能：倘若仅周期性地调用List，轮询地获取资源的期望状态并在与当前状态不一致时执行更新，自然也可以do the job。但是高频的轮询会大大增加API Server的负担，低频的轮询也会影响事件响应的实时性。Watch这一异步消息机制的结合，在保证了实时性的基础上也减少了API Server的负担，保证了高性能。

事件处理的顺序性：我们知道，每个资源对象都有一个递增的ResourceVersion，唯一地标识它当前的状态是“第几个版本”，每当这个资源内容发生变化时，对应产生的事件的ResourceVersion也会相应增加。在并发场景下，K8s可能获得同一资源的多个事件，由于K8s只关心资源的最终状态，因此只需要确保执行事件的ResourceVersion是最新的，即可确保事件处理的顺序性。

## ResourceVersion


