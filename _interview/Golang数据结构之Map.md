---
title: Golang数据结构之Map
tag: interview
---

## 概述

Map 作为日常工作最常被使用到的一种数据结构，我们很有必要通过了解它的实现方式来认识其特性，正确的使用它。在 Golang 中，Map 是通过哈希表的方式实现的，接下来我们通过分析源码的方式，来解答我们的疑问。

## 定义

map的定义位于 src/runtime/map.go 中，首先我们看下 map 和 bucket 的定义：

```go
// A header for a Go map.
type hmap struct {
	// Note: the format of the hmap is also encoded in cmd/compile/internal/reflectdata/reflect.go.
	// Make sure this stays in sync with the compiler's definition.
	count     int       // 元素个数，len(map) 返回 count 值
	flags     uint8
	B         uint8     // 位数，最大元素个数为 loadFactor * 2^B
	noverflow uint16    // 溢出个数
	hash0     uint32    // 哈希种子

	buckets    unsafe.Pointer // 桶地址. may be nil if count==0.
	oldbuckets unsafe.Pointer // 旧桶地址
	nevacuate  uintptr        // 搬迁进度，小于 nevacuate 的已经搬迁

	extra *mapextra     // 扩展--+属性
}

// 不是所有 map 都拥有 mapextra 
type mapextra struct {
	// If both key and elem do not contain pointers and are inline, then we mark bucket
	// type as containing no pointers. This avoids scanning such maps.
	// However, bmap.overflow is a pointer. In order to keep overflow buckets
	// alive, we store pointers to all overflow buckets in hmap.extra.overflow and hmap.extra.oldoverflow.
	// overflow and oldoverflow are only used if key and elem do not contain pointers.
	// overflow contains overflow buckets for hmap.buckets.
	// oldoverflow contains overflow buckets for hmap.oldbuckets.
	// The indirection allows to store a pointer to the slice in hiter.
	overflow    *[]*bmap
	oldoverflow *[]*bmap

	// nextOverflow holds a pointer to a free overflow bucket.
	nextOverflow *bmap
}

// Map Bucket
type bmap struct {
	// tophash generally contains the top byte of the hash value
	// for each key in this bucket. If tophash[0] < minTopHash,
	// tophash[0] is a bucket evacuation state instead.
	tophash [bucketCnt]uint8
	// Followed by bucketCnt keys and then bucketCnt elems.
	// NOTE: packing all the keys together and then all the elems together makes the
	// code a bit more complicated than alternating key/elem/key/elem/... but it allows
	// us to eliminate padding which would be needed for, e.g., map[int64]int8.
	// Followed by an overflow pointer.
}
```