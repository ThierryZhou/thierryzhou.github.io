---
title: Golang数据结构之Interface
tag: interview
---

## 背景

除了了与基础性能息息相关的网络和内存管理之外，Golang 给人印象最深的一个特性就是 Inerface 数据结构了，Interface 距离业务系统非常近，其独特的静态编译，动态检测的类型定义方式为提供了非常好的编程灵活性，大大简化了业务系统设计的复杂程度。

## 概述

通过 Interface 你可以像使用Python、JavaScript这类动态类型那样的完成对象的类型动态转换，与此同时作为一门传统编译型语言，在编译过程中，编译器会帮你找到程序中类型不匹配的问题。

下面我们尝试通过一个简单的例子，看看 Interface 如何使用：
```go
// 首先，声明一个拥有两个函数的接口类型 ReadCloser，以及一个接收 ReadCloser 接口类型参数的函数
type ReadCloser interface {
    Read(b []byte) (n int, err os.Error)
    Close()
}

func ReadAndClose(r ReadCloser, buf []byte) (n int, err os.Error) {
    for len(buf) > 0 && err == nil {
        var nr int
        nr, err = r.Read(buf)
        n += nr
        buf = buf[nr:]
    }
    r.Close()
    return
}

// 接下来，然后声明一个拥有一个函数的接口类型 Stringer，以及一个接口 interface{} 接口类型的函数
type Stringer interface {
    String() string
}

func ToString(any interface{}) string {
    if v, ok := any.(Stringer); ok {
        return v.String()
    }
    switch v := any.(type) {
    case int:
        return strconv.Itoa(v)
    case float:
        return strconv.Ftoa(v, 'g', -1)
    }
    return "???"
}

// 最后是测试代码
type stringer struct {
    data string
}

func test1() {
    stringer s
    t := "hello world"
    ReadAndClose(s, t)
}

func test2() {
    stringer s
    ToString(s)
}
```

函数 test1 中由于我们的 stringer 数据结构并没有实现 Read 和 Close 函数，此处会引起编译时的报错，而 test2 中由于使用 interface{} 编译器不会它为绑定任何静态类型检测，因此编译不会出错，函数体中第一句是一个类型断言，如果 any 对象可以转换成 Stringer 接口类，则 ok 为 true；如果不能完成转换，则 ok 为 false。如果类型转换成功，则调用 String 函数并返回结果，如果转换失败，则做一个类型判断，判断 any 的类型是否是 int 或者 float，如果是则调用 strconv 将数值转换成字符串，如果不是则返回 "???"。

编译器是如何判断 any 对象是否可以完成类型转换呢？

编译器通过检查 any 所对应的函数表中是否存在 String 这个函数，如果存在则可以完成类型转换，如果不存在则无法完成。

PS: 需要说明的是 "switch v := any.(type)" 一般也成为 type-switch，中文翻译为类型分支，可以算作做 type assertion 的语法糖，每个分支会被编译器解释为一句包含 type assertion 的 case 语句，示例如下：

```go
package main

import (
    "fmt"
    "strconv"
)

type Stringer interface {
    String() string
}

type Binary uint64

func (i Binary) String() string {
    return strconv.Uitob64(i.Get(), 2)
}

func (i Binary) Get() uint64 {
    return uint64(i)
}

func main() {
    b := Binary(200)
    s := Stringer(b)
    fmt.Println(s.String())
}
```
使用 Golang 提供的编译工具转换成 Plan9 汇编，截取 test1 函数的代码段
```shell
go tool compile -S main.go
main.test1 STEXT size=261 args=0x10 locals=0x50 funcid=0x0 align=0x0
+	0x0000 00000 (main.go:7)	TEXT	main.test1(SB), ABIInternal, $80-16
+	0x0000 00000 (main.go:7)	CMPQ	SP, 16(R14)
+	0x0004 00004 (main.go:7)	PCDATA	$0, $-2
+	0x0004 00004 (main.go:7)	JLS	229
+	0x000a 00010 (main.go:7)	PCDATA	$0, $-1
+	0x000a 00010 (main.go:7)	SUBQ	$80, SP
+	0x000e 00014 (main.go:7)	MOVQ	BP, 72(SP)
+	0x0013 00019 (main.go:7)	LEAQ	72(SP), BP
+	0x0018 00024 (main.go:7)	MOVQ	AX, main.any+88(FP)
+	0x001d 00029 (main.go:7)	MOVQ	BX, main.any+96(FP)
+	0x0022 00034 (main.go:7)	FUNCDATA	$0, gclocals·IuErl7MOXaHVn7EZYWzfFA==(SB)
+	0x0022 00034 (main.go:7)	FUNCDATA	$1, gclocals·EXTrhv4b3ahawRWAszmcVw==(SB)
+	0x0022 00034 (main.go:7)	FUNCDATA	$2, main.test1.stkobj(SB)
+	0x0022 00034 (main.go:7)	FUNCDATA	$5, main.test1.arginfo1(SB)
+	0x0022 00034 (main.go:7)	FUNCDATA	$6, main.test1.argliveinfo(SB)
+	0x0022 00034 (main.go:7)	PCDATA	$3, $1
+	0x0022 00034 (main.go:8)	TESTQ	AX, AX
+	0x0025 00037 (main.go:8)	JEQ	219
+	0x002b 00043 (main.go:8)	MOVL	16(AX), DX
+	0x002e 00046 (main.go:8)	CMPL	DX, $1810709754
+	0x0034 00052 (main.go:8)	JNE	137
++	0x0036 00054 (main.go:9)	LEAQ	type.int32(SB), DX
+	0x003d 00061 (main.go:9)	NOP
+	0x0040 00064 (main.go:9)	CMPQ	AX, DX
+	0x0043 00067 (main.go:9)	JNE	219
+	0x0049 00073 (main.go:10)	MOVUPS	X15, main..autotmp_17+56(SP)
++	0x004f 00079 (main.go:10)	LEAQ	type.string(SB), DX
+	0x0056 00086 (main.go:10)	MOVQ	DX, main..autotmp_17+56(SP)
+	0x005b 00091 (main.go:10)	LEAQ	main..stmp_0(SB), DX
+	0x0062 00098 (main.go:10)	MOVQ	DX, main..autotmp_17+64(SP)
+	0x0067 00103 (<unknown line number>)	NOP
+	0x0067 00103 ($GOROOT/src/fmt/print.go:294)	MOVQ	os.Stdout(SB), BX
+	0x006e 00110 ($GOROOT/src/fmt/print.go:294)	LEAQ	go.itab.*os.File,io.Writer(SB), AX
+	0x0075 00117 ($GOROOT/src/fmt/print.go:294)	LEAQ	main..autotmp_17+56(SP), CX
+	0x007a 00122 ($GOROOT/src/fmt/print.go:294)	MOVL	$1, DI
+	0x007f 00127 ($GOROOT/src/fmt/print.go:294)	MOVQ	DI, SI
+	0x0082 00130 ($GOROOT/src/fmt/print.go:294)	PCDATA	$1, $1
+	0x0082 00130 ($GOROOT/src/fmt/print.go:294)	CALL	fmt.Fprintln(SB)
+	0x0087 00135 (main.go:8)	JMP	219
+	0x0089 00137 (main.go:8)	CMPL	DX, $-1920832363
+	0x008f 00143 (main.go:8)	JNE	219
++	0x0091 00145 (main.go:11)	LEAQ	type.float32(SB), DX
+	0x0098 00152 (main.go:11)	CMPQ	AX, DX
+	0x009b 00155 (main.go:11)	JNE	219
+	0x009d 00157 (main.go:12)	MOVUPS	X15, main..autotmp_19+40(SP)
++	0x00a3 00163 (main.go:12)	LEAQ	type.string(SB), DX
+	0x00aa 00170 (main.go:12)	MOVQ	DX, main..autotmp_19+40(SP)
+	0x00af 00175 (main.go:12)	LEAQ	main..stmp_1(SB), DX
+	0x00b6 00182 (main.go:12)	MOVQ	DX, main..autotmp_19+48(SP)
+	0x00bb 00187 (<unknown line number>)	NOP
+	0x00bb 00187 ($GOROOT/src/fmt/print.go:294)	MOVQ	os.Stdout(SB), BX
+	0x00c2 00194 ($GOROOT/src/fmt/print.go:294)	LEAQ	go.itab.*os.File,io.Writer(SB), AX
+	0x00c9 00201 ($GOROOT/src/fmt/print.go:294)	LEAQ	main..autotmp_19+40(SP), CX
+	0x00ce 00206 ($GOROOT/src/fmt/print.go:294)	MOVL	$1, DI
+	0x00d3 00211 ($GOROOT/src/fmt/print.go:294)	MOVQ	DI, SI
+	0x00d6 00214 ($GOROOT/src/fmt/print.go:294)	CALL	fmt.Fprintln(SB)
+	0x00db 00219 (main.go:14)	PCDATA	$1, $-1
+	0x00db 00219 (main.go:14)	MOVQ	72(SP), BP
+	0x00e0 00224 (main.go:14)	ADDQ	$80, SP
+	0x00e4 00228 (main.go:14)	RET
+	0x00e5 00229 (main.go:14)	NOP
+	0x00e5 00229 (main.go:7)	PCDATA	$1, $-1
+	0x00e5 00229 (main.go:7)	PCDATA	$0, $-2
+	0x00e5 00229 (main.go:7)	MOVQ	AX, 8(SP)
+	0x00ea 00234 (main.go:7)	MOVQ	BX, 16(SP)
+	0x00ef 00239 (main.go:7)	CALL	runtime.morestack_noctxt(SB)
+	0x00f4 00244 (main.go:7)	MOVQ	8(SP), AX
+	0x00f9 00249 (main.go:7)	MOVQ	16(SP), BX
+	0x00fe 00254 (main.go:7)	PCDATA	$0, $-1
+	0x00fe 00254 (main.go:7)	NOP
+	0x0100 00256 (main.go:7)	JMP	0

// 省略后面的非函数声明部分的源码
// ...
```
可以看到四行 type assertion 被标记出来了(行首标记为++)。

回到类型断言的问题来，接下来看一个简单的例子。

```go
type Binary uint64

var _ Stringer = (*Binary)(nil)

func (i Binary) String() string {
    return strconv.Uitob64(i.Get(), 2)
}

func (i Binary) Get() uint64 {
    return uint64(i)
}
```

如果我们定义一个 Binary 类型的变量将其传入 ToString() 函数时，由于我们为 Binary 类型定义了 String() 函数，因此 any.(Stringer) 可以将 Binary 转换成 Stringer 接口类型。这就是我们说的 Golang 接口可以实现 ”鸭子类型“ 的威力，我们无需显示的声明接口类型，编译器会通过接口比对的方式为我们校验类型是否匹配。需要主要是工程上，Golang 鸭子类型虽然有灵活性的优点，但是一般跨包或是项目去实现某个接口时，程序会变的可读性非常差并且极容器出错，那我们能不能像 Java 中的接口一样，让我们知道一个接口中我们需要实现具体定义哪些函数，类似于”接口继承”的语法。

## Interface 实现

Golang 的类型设计原则中，一般包含 type 和 value 两部分， Interface 的实现也遵循这个原则，不过，golang 编译器会根据 interface 是否包含有 method，实现上用两种不同数据结构来：一种是有 method 的 interface 对应的数据结构为 iface；一种是没有 method 的 empty interface 对应的数据结构为 eface。

```go
// eface 数据结构
type eface struct {
    _type *_type                // 类型信息
    data  unsafe.Pointer        // 原数据存放的位置
}

// 大多数的 Golang 中的数据结构其底层都会对应一种 _type 类型的数据结构
type _type struct {
    size       uintptr // type size
    ptrdata    uintptr // size of memory prefix holding all pointers
    hash       uint32  // hash of type; avoids computation in hash tables
    tflag      tflag   // extra type information flags
    align      uint8   // alignment of variable with this type
    fieldalign uint8   // alignment of struct field with this type
    kind       uint8   // enumeration for C
    alg        *typeAlg  // algorithm table
    gcdata    *byte    // garbage collection data
    str       nameOff  // string form
    ptrToThis typeOff  // type for pointer to this type, may be zero
}

// iface 数据结构
type iface struct {
    tab  *itab              // 可以理解为含有接口函数表的类型信息
    data unsafe.Pointer     // 原数据存放的位置
}

type itab struct {
    inter  *interfacetype       // interface 类型信息
    _type  *_type               // 原数据结构的类型信息
    link   *itab
    bad    int32
    inhash int32      // 只有 itab 被拥有记录在了 hash 表中
    fun    [1]uintptr // 函数表入口指针
}

// 函数名声明
type imethod struct {
	name nameOff
	ityp typeOff
}

type interfacetype struct {
	typ     _type                   // interface type
	pkgpath name                    // 包路径
	mhdr    []imethod               // 接口函数名声明表
}
```

需要说明：
1. 对于非空的接口，其对应的类型有两个，一般称为 interface type 和 concrete type， interface type 保存在 itab.inter.typ 中，concrete type 保存在 itab._type 中。
2. itab 中存放了两张函数表，一张表对应接口对应实际接收到的函数，通过 itab.fun[0] 指向第一个函数对应的函数指针，后续函数根据函数名的字典值有小到大的排列，与C++中对象的虚函数表及虚函数的定义方式非常类似，另外一张表对应接口定制是函数声明。

下面通过一段代码样例，展示 Golang 如何实现 interface 对象的创建及类型转换。

```go
package main

import (
    "fmt"
    "strconv"
)

type Stringer interface {
    String() string
}

type Binary uint64

func (i Binary) String() string {
    return strconv.Uitob64(i.Get(), 2)
}

func (i Binary) Get() uint64 {
    return uint64(i)
}

func main() {
1    b := Binary(200)
2    var a interface{} = b
3    s := Stringer(b)
4    fmt.Println(s.String())
}
```
main 函数代码第一行，由于 Binary 没有实现任何函数时，因此 对象 b 只是一个普通的数据结构，其对应的内存区域存放对应的 uint64 数值。

![Binary](/assets/images/posts/gointer1.png)

代码第二行，尝试创建一个空接口对象，然后将 Binary 对象赋值给它，编译器会构造一个 eface 类型的数据结构，然后将 Binary 对应的 _type 以及 Binary 对应的数据保存下来。

![空接口](/assets/images/posts/gointer2.png)

代码第三行，创建一个 Stringer 类型的接口对象，然后将 Binary 对象赋值给它，编译器会构造一个 iface 类型的数据结构，itab中分别保存 Binary 和 Stringer 对应的 _type 以及 定义了 Binary 作为接收器的函数指针。

![Stringer](/assets/images/posts/gointer3.png)

## 类型断言(Type Assertion)

根据前面看过了的例子，我们知道类型断言的语句会被替换成 runtime 包中的 assert 函数，那我们把这两个函数的源码贴出来，需要说明一下 assertI2I 是 iface 类型的接口类型断言对应的函数，assertE2I 是 eface 类型的接口类型断言对应的函数。

类型断言的工作其实非常简单，先判断 itab.inter 这个接口类型是否相同，如果相同直接返回，如果不同则进入 getitab 进行处理，重点看一下这个函数。

```go
func assertI2I(inter *interfacetype, tab *itab) *itab {
	if tab == nil {
		// explicit conversions require non-nil interface value.
		panic(&TypeAssertionError{nil, nil, &inter.typ, ""})
	}
	if tab.inter == inter {
		return tab
	}
    // 
	return getitab(inter, tab._type, false)
}

func assertI2I2(inter *interfacetype, i iface) (r iface) {
	tab := i.tab
	if tab == nil {
		return
	}
	if tab.inter != inter {
		tab = getitab(inter, tab._type, true)
		if tab == nil {
			return
		}
	}
	r.tab = tab
	r.data = i.data
	return
}

func assertE2I(inter *interfacetype, t *_type) *itab {
	if t == nil {
		// explicit conversions require non-nil interface value.
		panic(&TypeAssertionError{nil, nil, &inter.typ, ""})
	}
	return getitab(inter, t, false)
}

func assertE2I2(inter *interfacetype, e eface) (r iface) {
	t := e._type
	if t == nil {
		return
	}
	tab := getitab(inter, t, true)
	if tab == nil {
		return
	}
	r.tab = tab
	r.data = e.data
	return
}

func getitab(inter *interfacetype, typ *_type, canfail bool) *itab {
    // _, ok := a.(interface{}) 这样的空接口断言直接抛出错误。
	if len(inter.mhdr) == 0 {
		throw("internal error - misuse of itab")
	}

	// 判断传入类型是否为 Uncomon type
    // Golang 类型定义这里就不展开做详细的讲解了，Uncommon Type可以简单理解为一个绑定了 Methods 的数据结构。
	if typ.tflag&tflagUncommon == 0 {
		if canfail {
			return nil
		}
		name := inter.typ.nameOff(inter.mhdr[0].name)
		panic(&TypeAssertionError{nil, typ, &inter.typ, name.name()})
	}

	var m *itab

    // itabTable 是编译器创建的 itab 缓存哈希表，先通过原子操作查表找到 itab 如果没有找到，则再通过加锁方式查找一遍
	t := (*itabTableType)(atomic.Loadp(unsafe.Pointer(&itabTable)))
	if m = t.find(inter, typ); m != nil {
		goto finish
	}

	// Not found.  Grab the lock and try again.
	lock(&itabLock)
	if m = itabTable.find(inter, typ); m != nil {
		unlock(&itabLock)
		goto finish
	}

	// 经过两次查表都没有找到 itab 对应的类型，就创建一个新的 itab 对象，并将其存入 itabTable 中
	m = (*itab)(persistentalloc(unsafe.Sizeof(itab{})+uintptr(len(inter.mhdr)-1)*goarch.PtrSize, 0, &memstats.other_sys))
	m.inter = inter
	m._type = typ
	m.hash = 0
	m.init()
	itabAdd(m)
	unlock(&itabLock)
finish:
	if m.fun[0] != 0 {
		return m
	}
	if canfail {
		return nil
	}
	// this can only happen if the conversion
	// was already done once using the , ok form
	// and we have a cached negative result.
	// The cached result doesn't record which
	// interface function was missing, so initialize
	// the itab again to get the missing function name.
	panic(&TypeAssertionError{concrete: typ, asserted: &inter.typ, missingMethod: m.init()})
}
```