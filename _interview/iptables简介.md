---
title:  "iptables 简介"
tag:    interview
---

iptables是运行在用户空间的应用软件，通过控制Linux内核netfilter模块，来管理网络数据包的处理和转发。通常iptables需要内核模块支持才能运行，此处相应的内核模块通常是Xtables。netfilter/iptables 组成Linux平台下的包过滤防火墙，可以代替昂贵的商业防火墙解决方案，完成封包过滤、封包重定向和网络地址转换（NAT）等功能。

Iptables和netfilter的关系是一个很容易让人搞不清的问题。很多的知道iptables却不知道netfilter。其实iptables只是Linux防火墙的管理工具而已，位于/sbin/iptables。真正实现防火墙功能的是netfilter，它是Linux内核中实现包过滤的内部结构。

## iptables架构
iptables、ip6tables等都使用Xtables框架。存在“表（tables）”、“链（chain）”和“规则（rules）”三个层面。
表指的是不同类型的数据包处理流程，每个表中又可以存在多个“链”，系统按照预订的规则将数据包通过某个内建链。

![架构](/assets/posts/iptables-arch.png)

例如将从本机发出的数据通过OUTPUT链。在“链”中可以存在若干“规则”，这些规则会被逐一进行匹配，如果匹配，可以执行相应的动作，如修改数据包，或者跳转。跳转可以直接接受该数据包或拒绝该数据包，也可以跳转到其他链继续进行匹配，或者从当前链返回调用者链。当链中所有规则都执行完仍然没有跳转时，将根据该链的默认策略（“policy”）执行对应动作；如果也没有默认动作，则是返回调用者链。

### 规则
规则（rules）其实就是网络管理员预定义的条件，规则一般的定义为“如果数据包头符合这样的条件，就这样处理这个数据包”。规则存储在内核空间的信息 包过滤表中，这些规则分别指定了源地址、目的地址、传输协议（如TCP、UDP、ICMP）和服务类型（如HTTP、FTP和SMTP）等。当数据包与规则匹配时，iptables就根据规则所定义的方法来处理这些数据包。实际工作中，配置防火墙的主要工作就是添加、修改和删除这些规则。  
ACCEPT ：接收数据包。  
DROP ：丢弃数据包。  
REDIRECT ：重定向、映射、透明代理。  
SNAT ：源地址转换。  
DNAT ：目标地址转换。  
MASQUERADE ：IP伪装（NAT），用于ADSL。  
LOG ：日志记录。  
SEMARK : 添加SEMARK标记以供网域内强制访问控制（MAC）。  

一条规则无法单独生效，它先需要被加入到规则表和规则链中。大多数的Linux发型版中，一般是四表五链的结构。

### 规则链
规则链（chains）是数据包传播的路径，每一条链其实就是众多规则中的一个检查清单，每一条链中可以有一条或数条规则。当一个数据包到达一个链时，iptables就会从链中第一条规则开始检查，看该数据包是否满足规则所定义的条件。如果满足，系统就会根据 该条规则所定义的方法处理该数据包；否则iptables将继续检查下一条规则，如果该数据包不符合链中任一条规则，iptables就会根据该链预先定 义的默认策略来处理数据包。

INPUT链 ：处理输入数据包。  
OUTPUT链 ：处理输出数据包。  
FORWARD链 ：处理转发数据包。  
PREROUTING链 ：用于目标地址转换（DNAT）。  
POSTOUTING链 ：用于源地址转换（SNAT）。

规则链之间的先后顺序是 RREROUTING -> INPUT —> FORWARD -> OUTPUT -> POSTROUTING

![架构](/assets/posts/iptables-lists.png)

#### 规则表
按照业务规则分别是filter表负责进行数据包过滤，nat负责进行地址转换操作，raw表负责异常处理，mangle表负责数据处理。他们之前的先后顺序是raw -> mangle -> nat -> filter。

![架构](/assets/posts/iptables-tables.png)

1. filter表  
filter表是默认的表，如果不指明表则使用此表。其通常用于过滤数据包。其中的内建链包括：  
INPUT，输入链。发往本机的数据包通过此链。  
OUTPUT，输出链。从本机发出的数据包通过此链。  
FORWARD，转发链。本机转发的数据包通过此链。
2. nat表  
nat表如其名，用于地址转换操作。其中的内建链包括：  
PREROUTING，路由前链，在处理路由规则前通过此链，通常用于目的地址转换（DNAT）。  
POSTROUTING，路由后链，完成路由规则后通过此链，通常用于源地址转换（SNAT）。  
OUTPUT，输出链，类似PREROUTING，但是处理本机发出的数据包。
3. mangle表  
mangle表用于处理数据包。其和nat表的主要区别在于，nat表侧重连接而mangle表侧重每一个数据包。其中内建链列表如下：  
PREROUTING  
INPUT  
FORWARD  
OUTPUT  
POSTROUTING
4. raw表  
raw表用于处理异常，有如下两个内建链：  
PREROUTING  
OUTPUT  

![架构](/assets/posts/iptables-packages.png)

iptables传输数据包的过程：  
1. 当一个数据包进入网卡时，它首先进入PREROUTING链，内核根据数据包目的IP判断是否需要转送出去。  
2. 如果数据包就是进入本机的，它就会沿着图向下移动，到达INPUT链。数据包到了INPUT链后，任何进程都会收到它。本机上运行的程序可以发送数据包，这些数据包会经过OUTPUT链，然后到达POSTROUTING链输出。  
3. 如果数据包是要转发出去的，且内核允许转发，数据包就会如图所示向右移动，经过FORWARD链，然后到达POSTROUTING链输出。

## iptables实例
清空当前的所有规则和计数
```shell
iptables -F  # 清空所有的防火墙规则
iptables -X  # 删除用户自定义的空链
iptables -Z  # 清空计数
```
配置允许ssh端口连接
```shell
iptables -A INPUT -s 192.168.1.0/24 -p tcp --dport 22 -j ACCEPT
# 22为你的ssh端口， -s 192.168.1.0/24表示允许这个网段的机器来连接，其它网段的ip地址是登陆不了你的机器的。 -j ACCEPT表示接受这样的请求
```
允许本地回环地址可以正常使用
```shell
iptables -A INPUT -i lo -j ACCEPT
#本地圆环地址就是那个127.0.0.1，是本机上使用的,它进与出都设置为允许
iptables -A OUTPUT -o lo -j ACCEPT
```
设置默认的规则
```shell
iptables -P INPUT DROP # 配置默认的不让进
iptables -P FORWARD DROP # 默认的不允许转发
iptables -P OUTPUT ACCEPT # 默认的可以出去
```
配置白名单
```shell
iptables -A INPUT -p all -s 192.168.1.0/24 -j ACCEPT  # 允许机房内网机器可以访问
iptables -A INPUT -p all -s 192.168.140.0/24 -j ACCEPT  # 允许机房内网机器可以访问
iptables -A INPUT -p tcp -s 183.121.3.7 --dport 3380 -j ACCEPT # 允许183.121.3.7访问本机的3380端口
```
开启相应的服务端口
```shell
iptables -A INPUT -p tcp --dport 80 -j ACCEPT # 开启80端口，因为web对外都是这个端口
iptables -A INPUT -p icmp --icmp-type 8 -j ACCEPT # 允许被ping
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT # 已经建立的连接得让它进来
```
保存规则到配置文件中
```shell
cp /etc/sysconfig/iptables /etc/sysconfig/iptables.bak # 任何改动之前先备份，请保持这一优秀的习惯
iptables-save > /etc/sysconfig/iptables
cat /etc/sysconfig/iptables
```
列出已设置的规则
```shell
# iptables -L 示例
iptables -L [-t 表名] [链名]

#
iptables -L -t nat                  # 列出 nat 上面的所有规则
#            ^ -t 参数指定，必须是 raw， nat，filter，mangle 中的一个
iptables -L -t nat  --line-numbers  # 规则带编号
iptables -L INPUT

iptables -L -nv  # 查看，这个列表看起来更详细
```
清除已有规则
```shell
iptables -F INPUT  # 清空指定链 INPUT 上面的所有规则
iptables -X INPUT  # 删除指定的链，这个链必须没有被其它任何规则引用，而且这条上必须没有任何规则。
                   # 如果没有指定链名，则会删除该表中所有非内置的链。
iptables -Z INPUT  # 把指定链，或者表中的所有链上的所有计数器清零。
```
删除已添加的规则
```shell
# 添加一条规则
iptables -A INPUT -s 192.168.1.5 -j DROP
将所有iptables以序号标记显示，执行：

iptables -L -n --line-numbers
比如要删除INPUT里序号为8的规则，执行：

iptables -D INPUT 8
开放指定的端口
iptables -A INPUT -s 127.0.0.1 -d 127.0.0.1 -j ACCEPT               #允许本地回环接口(即运行本机访问本机)
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT    #允许已建立的或相关连的通行
iptables -A OUTPUT -j ACCEPT         #允许所有本机向外的访问
iptables -A INPUT -p tcp --dport 22 -j ACCEPT    #允许访问22端口
iptables -A INPUT -p tcp --dport 80 -j ACCEPT    #允许访问80端口
iptables -A INPUT -p tcp --dport 21 -j ACCEPT    #允许ftp服务的21端口
iptables -A INPUT -p tcp --dport 20 -j ACCEPT    #允许FTP服务的20端口
iptables -A INPUT -j reject       #禁止其他未允许的规则访问
iptables -A FORWARD -j REJECT     #禁止其他未允许的规则访问
屏蔽IP
iptables -A INPUT -p tcp -m tcp -s 192.168.0.8 -j DROP  # 屏蔽恶意主机（比如，192.168.0.8
iptables -I INPUT -s 123.45.6.7 -j DROP       #屏蔽单个IP的命令
iptables -I INPUT -s 123.0.0.0/8 -j DROP      #封整个段即从123.0.0.1到123.255.255.254的命令
iptables -I INPUT -s 124.45.0.0/16 -j DROP    #封IP段即从123.45.0.1到123.45.255.254的命令
iptables -I INPUT -s 123.45.6.0/24 -j DROP    #封IP段即从123.45.6.1到123.45.6.254的命令是
```
指定数据包出去的网络接口
只对 OUTPUT，FORWARD，POSTROUTING 三个链起作用。
```shell
iptables -A FORWARD -o eth0
```
查看已添加的规则
```shell
iptables -L -n -v
Chain INPUT (policy DROP 48106 packets, 2690K bytes)
 pkts bytes target     prot opt in     out     source               destination
 5075  589K ACCEPT     all  --  lo     *       0.0.0.0/0            0.0.0.0/0
 191K   90M ACCEPT     tcp  --  *      *       0.0.0.0/0            0.0.0.0/0           tcp dpt:22
1499K  133M ACCEPT     tcp  --  *      *       0.0.0.0/0            0.0.0.0/0           tcp dpt:80
4364K 6351M ACCEPT     all  --  *      *       0.0.0.0/0            0.0.0.0/0           state RELATED,ESTABLISHED
 6256  327K ACCEPT     icmp --  *      *       0.0.0.0/0            0.0.0.0/0

Chain FORWARD (policy ACCEPT 0 packets, 0 bytes)
 pkts bytes target     prot opt in     out     source               destination

Chain OUTPUT (policy ACCEPT 3382K packets, 1819M bytes)
 pkts bytes target     prot opt in     out     source               destination
 5075  589K ACCEPT     all  --  *      lo      0.0.0.0/0            0.0.0.0/0
```
启动网络转发规则
公网210.14.67.7让内网192.168.188.0/24上网
```shell
iptables -t nat -A POSTROUTING -s 192.168.188.0/24 -j SNAT --to-source 210.14.67.127
```
端口映射
本机的 2222 端口映射到内网 虚拟机的22 端口
```shell
iptables -t nat -A PREROUTING -d 210.14.67.127 -p tcp --dport 2222  -j DNAT --to-dest 192.168.188.115:22
```
字符串匹配
比如，我们要过滤所有TCP连接中的字符串test，一旦出现它我们就终止这个连接，我们可以这么做：
```shell
iptables -A INPUT -p tcp -m string --algo kmp --string "test" -j REJECT --reject-with tcp-reset
iptables -L

# Chain INPUT (policy ACCEPT)
# target     prot opt source               destination
# REJECT     tcp  --  anywhere             anywhere            STRING match "test" ALGO name kmp TO 65535 reject-with tcp-reset
#
# Chain FORWARD (policy ACCEPT)
# target     prot opt source               destination
#
# Chain OUTPUT (policy ACCEPT)
# target     prot opt source               destination
```

### 自定义链
其实iptables中的自定义链已经够用了。但是我们为什么还要自定义链呢？当链上的规则足够多的时候，就不容易管理了。这个时候我们可以自定义分类，将不同的规则放在不同的链中。

1.创建自定义链
```shell
# 在filter表上创建一个自定义的链IN_WEB.
iptables -t filter -N IN_WEB
```
2.在自定义链上设置规则
```shell
# 在filter表中的IN_WEB链上创建一个规则，对原地址为192.168.1.1这个的连接进行阻止。
iptables -t filter -A IN_WEB -s 192.168.1.1 -j REJECT
```

3.这时候自定义链的规则还不能使用，必须借助于默认链来是实现。
当然，自定义链在哪里创建，应该被哪调默认的链引用，取决于应用场景，比如说要匹配入站报文，所以可以在INPUT链中引用
```shell
# 我们在INPUT链中添加了一些规则，访问本机80端口的tcp报文将会被这条规则匹配到。-j IN_WEB表示：访问80端口的tcp报文将由自定义链“IN_WEB”中的规则处理，没错，在之前的例子中-j 表示动作，当我们将动作替换成自定义链时，就表示被当前规则匹配到的报文将交由对应的自定义链中的规则处理，具体怎么处理，取决于自定义链中的规则。当IN_WEB被INPUT引用后，引用计数将会加1.
iptables -A INPUT -p tcp --dport 80 -j IN_WEB
```
3.重命名自定义链
```shell
iptables -E IN_WEB WEB
4.删除自定义链
```shell
iptables -X WEB
```