---
title:  "Mysql存储引擎之MyISAM"
tag:    interview
---

## 概述

### Mysql 基本架构

mysql基本架构组成：客户端，Server层和存储引擎层。其中，只有Server层和存储引擎层是属于Mysql。

![mysql](/assets/images/posts/mysql-stack.png)

Server层：连接器，查询缓存，分析器，优化器，执行器等，也包括mysql的大多数核心功能区以及所有内置函数。
1）内置函数：日期，时间，数学函数，加密函数等
2）所有跨存储引擎的功能都在这一层实现，如存储过程，触发器，视图等
3）通用的日志模块binglog日志模块
存储引擎：负责数据的存储和提取

## MyISAM 详解

MyISAM在磁盘上存储成3个文件，其中文件名和表名都相同，但是扩展名分别为：
.frm(存储表定义)  
.MYD(MYData，存储数据)  
.MYI(MYIndex，存储索引)  

![MyISAM](/assets/images/posts/mysql-myisam.png)

MyISAM引擎的索引结构为B+Tree，其中B+Tree的数据域存储的内容为实际数据的地址，也就是说它的索引和实际的数据是分开的，只不过是用索引指向了实际的数据，这种索引就是所谓的非聚集索引。如下图所示：

![MyISAM](/assets/images/posts/mysql-myisam-2.png)

同样也是一颗B+Tree，data域保存数据记录的地址。因此，MyISAM中索引检索的算法为首先按照B+Tree搜索算法搜索索引，如果指定的Key存在，则取出其data域的值，然后以data域的值为地址，读取相应数据记录。

在设计之时就考虑到数据库被查询的次数要远大于更新的次数。因此，ISAM执行读取操作的速度很快，而且不占用大量的内存和存储资源。由于数据索引和存储数据分离，MyISAM引擎的索引结构是B+Tree，其中B+Tree的数据域存储的内容为实际数据的地址，也就是说他的索引和实际数据是分开的。不过索引指向实际的数据，这种索引也就是非聚合索引。因此，MyISAM中索引检索的算法为首先按照B+Tree搜索算法搜索索引，如果指定的Key存在，则取出其data域的值，然后以data域的值为地址，读取相应数据记录。

但是它没有提供对数据库事务的支持，是表级锁（插入修改锁表），因此当INSERT(插入)或UPDATE(更新)数据时即写操作需要锁定整个表，效率便会低一些。不过和Innodb不同，MyIASM中存储了表的行数，于是SELECT COUNT(*) FROM TABLE时只需要直接读取已经保存好的值而不需要进行全表扫描。如果表的读操作远远多于写操作且不需要数据库事务的支持，那么MyIASM也是很好的选择。
