---
title: Ceph Monitor和Paxos
tag: ceph
---

## Paxos算法
##### 1. 角色和名词
proposers： 意为提案者，它可以提出一个提案，提案信息包括提案编号和提议的 value。

acceptors： 是提案的受理者，收到提案后可以接受（accept）提案，若提案获得多数派（majority）的 acceptors 的接受，则称该提案被批准 (chosen)。

learner：需要知道(学习)被批准 (chosen) 的提案信息的那些人.

proposal： 提案，由Proposer提出。一个提案由一个编号及value形成的对组成，编号是为了防止混淆保证提案的可区分性，value即代表了提案本身的内容。

choose： 提案被选定，在本文中当有半数以上Acceptor接受该提案时，就认为该提案被选定了，被选定的提案。


##### 2. Paxos 需要解决哪些问题?
划分角色后，就可以更精确的定义问题：

1. 决议（value）只有在被 proposers 提出后才能被批准（未经批准的决议称为“提案（proposal）”）。
2. 在一次 Paxos 算法的执行实例中，只批准（chosen）一个 value。
3. learners 只能获得被批准（chosen）的 value。

##### 3. acceptor 接受 proposal 有什么规则:
1) p1: 一个 acceptor 必须接受 (accept) 它收到的第一个提案。

   p1a: 当且仅当 acceptor 没有回应过编号大于 n 的 prepare 请求时，acceptor 接受（accept）编号为 n 的提案。

2) p2: 如果具有 value 值 v 的提案被选定 (chosen) 了，那么所有比它编号更高的被选定的提案的 value 值也必须是 v。

   p2c: 如果一个编号为 n 的提案具有 value v，该提案被提出（issued），那么存在一个多数派，要么他们中所有人都没有接受（accept）编号小于 n 的任何提案，要么他们已经接受（accept）的所有编号小于 n 的提案中编号最大的那个提案具有 value v。

##### 3. proposer 如何产生提案 (proposal) ?

1) proposer选择一个新的提案编号n，然后向某个acceptors集合的成员发送请求，要求acceptor做出如下回应：
  a). 保证不再通过任何编号小于n的提案
  b). 当前它已经通过的编号小于n的最大编号的提案，如果存在的话

2) 如果proposer收到了来自半数以上的acceptor的响应结果，那么它就可以产生编号为n，value值为v的提案，这里v是所有响应中编号最大的提案的value值，如果响应中不包含任何的提案那么这个值就可以由proposer任意选择。

我们把这样的一个请求称为编号为n的prepare请求。

Proposer通过向某个acceptors集合发送需要被通过的提案请求来产生一个提案(此时的acceptors集合不一定是响应prepare阶段请求的那个acceptors集合)。我们称此请求为accept请求。

##### 4. acceptor 如何响应上述算法？

Acceptor 可以忽略任何请求而不用担心破坏其算法的安全性。  
Acceptor 必须记住这些信息即使是在出错或者重启的情况下。  
Proposer 可以总是可以丢弃提案以及它所有的信息—只要它可以保证不会产生具有相同编号的提案即可。  

##### 5. 将 proposer 和 acceptor 放在一块，我们可以得到算法的如下两阶段执行过程：

Phase1.a) proposer选择一个提案编号n，然后向acceptors的某个majority集合的成员发送编号为n的prepare请求。

Phase1.b) 如果一个acceptor收到一个编号为n的prepare请求，且n大于它已经响应的所有prepare请求的编号。那么它就会保证不会再通过(accept)任何编号小于n的提案，同时将它已经通过的最大编号的提案(如果存在的话)作为响应{!?此处隐含了一个结论，最大编号的提案肯定是小于n的}。

Phase2.a) 如果proposer收到来自半数以上的acceptor对于它的prepare请求(编号为n)的响应，那么它就会发送一个针对编号为n，value值为v的提案的accept请求给acceptors，在这里v是收到的响应中编号最大的提案的值，如果响应中不包含提案，那么它就是任意值。

Phase2.b) 如果acceptor收到一个针对编号n的提案的accept请求，只要它还未对编号大于n的prepare请求作出响应，它就可以通过这个提案。

##### 6. 很容易构造出一种情况，在该情况下，两个proposers持续地生成编号递增的一系列提案。
为了保证进度，必须选择一个特定的proposer来作为一个唯一的提案提出者。

如果系统中有足够的组件(proposer，acceptors及通信网络)工作良好，通过选择一个特定的proposer，活性就可以达到。著名的FLP结论指出，一个可靠的proposer选举算法要么利用随机性要么利用实时性来实现—比如使用超时机制。然而，无论选举是否成功，安全性都可以保证。{!即即使同时有2个或以上的proposers存在，算法仍然可以保证正确性}

##### 7. 不同的proposers会从不相交的编号集合中选择自己的编号，这样任何两个proposers就不会有相同编号的提案了。

##### 8. 关于leader election算法：