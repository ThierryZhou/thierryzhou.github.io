---
title:  "Mysql存储引擎之InnoDB"
tag:    interview
---

## 概述

### Mysql 架构

mysql基本架构组成：客户端，Server层和存储引擎层。其中，只有Server层和存储引擎层是属于Mysql。

![mysql](/assets/images/posts/mysql-stack.png)

Server层：连接器，查询缓存，分析器，优化器，执行器等，也包括mysql的大多数核心功能区以及所有内置函数。
1. 内置函数：日期，时间，数学函数，加密函数等
2. 所有跨存储引擎的功能都在这一层实现，如存储过程，触发器，视图等
3. 通用的日志模块binglog日志模块

存储引擎：负责数据的存储和提取，目前最新版本的 Mysql 使用的存储引擎是 InnoDB。

## InnoDB 架构
从InnoDB存储引擎的存储结构看，所有数据都被逻辑地放在一个空间中，称之为表空间(tablespace)、段(Segment)、区(extent)、页(page)、行(Row)组成，页在一些文档中也被称之为块(block)。
表空间内，segment , extent 和 page 之间的关系是环环相扣的，其中页(page)是表空间的最小分配单位。一个页(page) 默认大小是16 KB，一个区(extent)管理64个page, 大小为1 MB，而段(Segment)可以管理很多 extent 以及额外的32个碎页(frag page), 碎页是为了节省空间而定义的。

![mysql-innodb](/assets/images/posts/mysql-innodb.png)

##### 1. 表空间(tablespace)

表空间可以看做 InnoDB 逻辑结构的最高层，所有的数据都放在表空间中。表空间由三种段构成：叶子节点数据段，即数据段；
非叶子节点数据段，即索引段；回滚段。

在默认情况下， InnoDB 存储引擎都有一个共享表空间 ibdata1 ，即所有数据都存放在这个表空间内。如果用户启用了参数 innodb_file_per_table ，则每张表内的数据可以单独放到一个表空间内。如果启用了 innodb_file_per_table 参数，也需要注意，每张表的表空间存放的只是数据、索引和插入缓冲 Bitmap 页，其它类的数据，如回滚 (undo) 信息，插入缓冲索引页、系统事务信息，二次写缓冲等还是存放在原来的共享表空间内。

##### 2. 段(segment)

对于大的数据段，InnoDB存储引擎最多每次可以申请4个区，以此来保证数据的顺序性能。

InnoDB 存储引擎表是索引组织 (index organized) 的，因此数据即索引，索引即数据。那么数据段即为 B+ 树的叶子节点 (Leaf node segment) ，索引段即为 B+ 树的非索引节点 (Non-leaf node segment) ，这些内容在后面的索引学习里会详细介绍。

##### 3. 区(extend)

区是由连续页组成的空间，在任何情况下每个区的大小都为1MB。为了保证区中页的连续性，InonoDB存储引擎一次从磁盘申请4-5个区。在默认情况下，InnoDB存储引擎的页的大小为16KB，即一个区中应有64个连续的页。

但是有时候为了节约磁盘容量的开销，创建表默认大小是96KB，区中是64个连续的页。(对于一些小表)

##### 4. 页(page)

一个页由多个行组成，从物理空间上来看一个页就是一个磁盘块，是io操作的最小物理存储单元，也就是我们读取一页内的数据时候，实际上才发生了一次IO，这个理论对于索引的数据结构设计非常有帮助。

InnoDB 存储引擎中，常见的页类型有：
1. 数据页(B+ tree Node)
2. undo页(undo Log Page)
3. 系统页 (System Page)
4. 事务数据页 (Transaction System Page)
5. 插入缓冲位图页(Insert Buffer Bitmap)
6. 插入缓冲空闲列表页(Insert Buffer Free List)
7. 未压缩的二进制大对象页(Uncompressed BLOB Page)
8. 压缩的二进制大对象页 (compressed BLOB Page)

###### InnoDB 数据页结构

页是 InnoDB 存储引擎管理数据库最小磁盘单位。页类型为 B+ tree Node 的页存放的即是表中行的实际数据了。

![mysql-innodb-header](/assets/images/posts/mysql-innodb-header.png)

InnoDB 数据页由以下 7 个部分组成:
1. File Header (文件头)
2. Page Header (页头)
3. Infimun 和 Supremum Records
4. User Records (用户记录，即行记录)
5. Free Space (空闲空间)
6. Page Directory (页目录)
7. File Trailer (文件结尾信息)

##### 5. 行(row)

InnoDB 存储引擎和大多数数据库一样(如 Oracle 和 Microsoft SQL Server 数据库)，记录是以行的形式存储的。这意味着页中保存着表中一行行的数据。在 InnoDB 1.0x 版本之前，InnoDB 存储引擎提供了 Compact 和 Redundant 两种格式来存放行记录数据，这也是目前使用最多的一种格式。Redundant 是 MySQL 5.0 版本之前 InnoDB 的行记录存储方式，这里就不展开。

###### Compact 行记录格式

Compact 行记录是在 MySQL 5.0 中引人的，其设计目标是髙效地存储数据。简单来说,一个页中存放的行数据越多，其性能就越髙。

下图显示了 Compact 行记录的存储方式：

![mysql-innodb-row](/assets/images/posts/mysql-innodb-row.png)

Compact 行记录格式的首部是一个非 NULL 变长字段长度列表，并且其是按照列的顺序逆序放置的，其长度为：若列的长度小于 255 字节，用 1 字节表示;若大于 255 个字节，用 2 字节表示。

变长字段的长度最大不可以超过 2 字节，这是因在 MySQL 数据库中 VARCHAR 类型的最大长度限制为 65535。变长字段之后的第二个部分是 NULL 标志位，该位指示了该行数据中是否有 NULL 值，有则用 1 表示。

需要特别注意的是，NULL 不占该部分任何空间，即 NULL 除了占有 NULL 标志位，实际存储不占有任何空间。另外有一点需要注意的是，每行数据除了用户定义的列外，还有两个隐藏列，事务 1D 列和回滚指针列,分别为 6 字节和 7 字节的大小。若 InnoDB 表没有定义主键，每行还会增加一个 6 字节的 rowid 列。

###### 行溢出数据

InnoDB 存储引擎可以将一条记录中的某些数据存储在真正的数据页之外。因为一般数据页默认大小为 16 KB，假如一个数据页存储不了插入的数据，这时肯定就会发生行溢出。

![mysql-innodb-page](/assets/images/posts/mysql-innodb-page.png)

一般认为 BLOB 这样的大对象列类型的存储会把数据存放在数据页之外。但是，BLOB 也可以不将数据放在溢出页面，而且即便是 VARCHAR 列数据类型，依然有可能被存放为行溢出数据。

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io

## 参考
- [1] [MySQL高级进阶：关于InnoDB存储结构，一文深入分析讲解](https://www.51cto.com/article/673996.html)