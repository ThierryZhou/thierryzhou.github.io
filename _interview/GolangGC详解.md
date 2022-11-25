---
titile: Golang GC详解
---

## 概述

## 三色标记

算法流程：
1. 遍历根对象的第一层可达对象标记为灰色, 不可达默认白色
2. 将灰色对象的下一层可达对象标记为灰色, 自身标记为黑色
3. 多次重复步骤2, 直到灰色对象为0, 只剩下白色对象和黑色对象
4. sweep 白色对象

示例：
1. 遍历根对象的第一层可达对象标记为灰色, 不可达默认白色
![3color-flow-1](/assets/images/posts/3color-flow-1.png)

2. 将灰色对象 A 的下一层可达对象标记为灰色, 自身标记为黑色
![3color-flow-2](/assets/images/posts/3color-flow-2.png)

3. 继续遍历灰色对象的下层对象,重复步骤2
![3color-flow-3](/assets/images/posts/3color-flow-3.png)

4. 继续遍历灰色对象的下层对象,重复步骤2
![3color-flow-4](/assets/images/posts/3color-flow-4.png)

## 写屏障

## 混合写屏障