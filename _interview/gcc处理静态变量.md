---
title:  "gccc如何处理static变量初始化"
tag:    interview
---
## 局部/全局变量
局部变量在C++中的使用要频繁的多，并且功能也强大的多，但是这些强大功能的背后无疑会引入问题的复杂性，不想让马儿吃草只想让马儿跑的事大家表乱想。这些初始化的实现就需要C++的库执行更多的动作来完成，虽然各种编译器都是像如今开展的“学雷锋”活动一样干了很多好事都没有留名，但是作为一个程序员，还是要对别人的贡献进行表彰。
我们看一下下面的一段代码，本文将会围绕这个代码进行展开，可以看到这个简单的程序，让C++生成了非常多的代码让人应接不暇
```shell
$ cat localstatic.c
extern int foo();
int globvar = foo();
int bar()
{
static int localvar = foo();
return localvar;
}
$ gcc localstatic.c  -c
localstatic.c:2: error: initializer element is not constant
localstatic.c: In function ‘bar’:
localstatic.c:5: error: initializer element is not constant
$ g++ localstatic.c  -c
$ objdump -rdCh localstatic.o 

localstatic.o:     file format elf32-i386

Sections:
Idx Name          Size      VMA       LMA       File off  Algn
  0 .text         000000b1  00000000  00000000  00000034  2**2
                  CONTENTS, ALLOC, LOAD, RELOC, READONLY, CODE
  1 .data         00000000  00000000  00000000  000000e8  2**2
                  CONTENTS, ALLOC, LOAD, DATA
  2 .bss          00000014  00000000  00000000  000000e8  2**3
                  ALLOC
  3 .gcc_except_table 0000000c  00000000  00000000  000000e8  2**0
                  CONTENTS, ALLOC, LOAD, READONLY, DATA
  4 .ctors        00000004  00000000  00000000  000000f4  2**2
                  CONTENTS, ALLOC, LOAD, RELOC, DATA
  5 .comment      0000002d  00000000  00000000  000000f8  2**0
                  CONTENTS, READONLY
  6 .note.GNU-stack 00000000  00000000  00000000  00000125  2**0
                  CONTENTS, READONLY
  7 .eh_frame     000000ac  00000000  00000000  00000128  2**2
                  CONTENTS, ALLOC, LOAD, RELOC, READONLY, DATA

Disassembly of section .text:

00000000 <bar()>:
   0:    55                       push   %ebp
   1:    89 e5                    mov    %esp,%ebp
   3:    57                       push   %edi
   4:    56                       push   %esi
   5:    53                       push   %ebx
   6:    83 ec 1c                 sub    $0x1c,%esp
   9:    b8 08 00 00 00           mov    $0x8,%eax
            a: R_386_32    .bss
   e:    0f b6 00                 movzbl (%eax),%eax
  11:    84 c0                    test   %al,%al
  13:    75 52                    jne    67 <bar()+0x67>
  15:    c7 04 24 08 00 00 00     movl   $0x8,(%esp)
            18: R_386_32    .bss
  1c:    e8 fc ff ff ff           call   1d <bar()+0x1d>
            1d: R_386_PC32    __cxa_guard_acquire当获得锁之后再次判断，这里也是避免多线程竞争的关键一步，此时可以保证之后操作原子性。
  21:    85 c0                    test   %eax,%eax
  23:    0f 95 c0                 setne  %al
  26:    84 c0                    test   %al,%al
  28:    74 3d                    je     67 <bar()+0x67>
  2a:    bb 00 00 00 00           mov    $0x0,%ebx
  2f:    e8 fc ff ff ff           call   30 <bar()+0x30>
            30: R_386_PC32    foo()
  34:    a3 10 00 00 00           mov    %eax,0x10
            35: R_386_32    .bss
  39:    c7 04 24 08 00 00 00     movl   $0x8,(%esp)
            3c: R_386_32    .bss
  40:    e8 fc ff ff ff           call   41 <bar()+0x41>
            41: R_386_PC32    __cxa_guard_release
  45:    eb 20                    jmp    67 <bar()+0x67>
  47:    89 d6                    mov    %edx,%esi
  49:    89 c7                    mov    %eax,%edi
  4b:    84 db                    test   %bl,%bl
  4d:    75 0c                    jne    5b <bar()+0x5b>
  4f:    c7 04 24 08 00 00 00     movl   $0x8,(%esp)
            52: R_386_32    .bss
  56:    e8 fc ff ff ff           call   57 <bar()+0x57>
            57: R_386_PC32    __cxa_guard_abort
  5b:    89 f8                    mov    %edi,%eax
  5d:    89 f2                    mov    %esi,%edx
  5f:    89 04 24                 mov    %eax,(%esp)
  62:    e8 fc ff ff ff           call   63 <bar()+0x63>
            63: R_386_PC32    _Unwind_Resume
  67:    a1 10 00 00 00           mov    0x10,%eax
            68: R_386_32    .bss
  6c:    83 c4 1c                 add    $0x1c,%esp
  6f:    5b                       pop    %ebx
  70:    5e                       pop    %esi
  71:    5f                       pop    %edi
  72:    5d                       pop    %ebp
  73:    c3                       ret    

00000074 <__static_initialization_and_destruction_0(int, int)>:
  74:    55                       push   %ebp
  75:    89 e5                    mov    %esp,%ebp
  77:    83 ec 08                 sub    $0x8,%esp
  7a:    83 7d 08 01              cmpl   $0x1,0x8(%ebp)
  7e:    75 13                    jne    93 <__static_initialization_and_destruction_0(int, int)+0x1f>
  80:    81 7d 0c ff ff 00 00     cmpl   $0xffff,0xc(%ebp)
  87:    75 0a                    jne    93 <__static_initialization_and_destruction_0(int, int)+0x1f>
  89:    e8 fc ff ff ff           call   8a <__static_initialization_and_destruction_0(int, int)+0x16>
            8a: R_386_PC32    foo()
  8e:    a3 00 00 00 00           mov    %eax,0x0
            8f: R_386_32    globvar
  93:    c9                       leave  
  94:    c3                       ret    

00000095 <global constructors keyed to globvar>:
  95:    55                       push   %ebp
  96:    89 e5                    mov    %esp,%ebp
  98:    83 ec 18                 sub    $0x18,%esp
  9b:    c7 44 24 04 ff ff 00     movl   $0xffff,0x4(%esp)
  a2:    00 
  a3:    c7 04 24 01 00 00 00     movl   $0x1,(%esp)
  aa:    e8 c5 ff ff ff           call   74 <__static_initialization_and_destruction_0(int, int)>
  af:    c9                       leave  
  b0:    c3                       ret    
```
这里可以看出几点比较有趣的内容：
1. 非常量变量对于全局变量和静态局部变量的初始化使用gcc无法编译通过，但是使用g++可以编译通过。而两者的区别在于gcc会把这个.c后缀的程序看做一个C程序，而g++则把这个.c后缀的看做c++文件，而c++语法是允许对变量进行更为复杂的初始化。
2. 全局变量的初始化实现使用了.ctors节，该节中保存了该编译单元中所有需要在main函数之前调用的初始化函数，其中对于globvar的赋值就在该函数中完成。
3. 局部静态变量的初始化，它要保证任意多个函数被调用，它只初始化一次，并且只能被初始化一次，并且这个初始化只能在执行到的时候执行，假设说这个bar函数从来没有在运行时执行过，那么这个局部变量的赋值就用完不能被执行到。
4. 里面有一些比较复杂的__cxa_guard_acquire操作，它在哪里定义，用来做什么？
## 全局变量的初始化
1. 初始化代码位置确定
这个正如之前说过的，它需要在main函数执行之前执行，
```shell
$ objdump -r localstatic.o

RELOCATION RECORDS FOR [.ctors]:
OFFSET   TYPE              VALUE 
00000000 R_386_32          .text
```
然后通过hexdump看一下这个地方的内容
```shell
$ hexdump localstatic.o 
0000000 457f 464c 0101 0001 0000 0000 0000 0000
0000010 0001 0003 0001 0000 0000 0000 0000 0000
0000020 0248 0000 0000 0000 0034 0000 0000 0028
0000030 000f 000c 8955 57e5 5356 ec83 b81c 0008
0000040 0000 b60f 8400 75c0 c752 2404 0008 0000
0000050 fce8 ffff 85ff 0fc0 c095 c084 3d74 00bb
0000060 0000 e800 fffc ffff 10a3 0000 c700 2404
0000070 0008 0000 fce8 ffff ebff 8920 89d6 84c7
0000080 75db c70c 2404 0008 0000 fce8 ffff 89ff
0000090 89f8 89f2 2404 fce8 ffff a1ff 0010 0000
00000a0 c483 5b1c 5f5e c35d 8955 83e5 08ec 7d83
00000b0 0108 1375 7d81 ff0c 00ff 7500 e80a fffc
00000c0 ffff 00a3 0000 c900 55c3 e589 ec83 c718
00000d0 2444 ff04 00ff c700 2404 0001 0000 c5e8
00000e0 ffff c9ff 00c3 0000 ffff 0801 052f 0047
00000f0 0562 0000 0095 0000 4700 4343 203a 4728
```
由于在开始可以看到这个.ctors节位于0xf4地址，所以可以看到这个地方的内容为0x95(考虑到386的小端结构)，结合上面的重定位项
```shell
RELOCATION RECORDS FOR [.ctors]:
OFFSET   TYPE              VALUE 
00000000 R_386_32          .text
```
可以看到，在.ctors节中要放上该文件中.text节起始地址(最终连接生成的可执行文件中)位置，也就是说最终生成的可执行文件的.ctors节中将会有一项，它的地址就是
```shell
00000095 <global constructors keyed to globvar>:
```
代码的地址。
2. .ctors节
```shell
$ ld --verbose
GNU ld version 2.19.51.0.14-34.fc12 20090722
  Supported emulations:
   elf_i386
   i386linux
   elf_x86_64
using internal linker script:
==================================================
/* Script for -z combreloc: combine and sort reloc sections */

.ctors          :
  {
    /* gcc uses crtbegin.o to find the start of
       the constructors, so we make sure it is
       first.  Because this is a wildcard, it
       doesn't matter if the user does not
       actually link against crtbegin.o; the
       linker won't look for a file to match a
       wildcard.  The wildcard also means that it
       doesn't matter which directory crtbegin.o
       is in.  */
    KEEP (*crtbegin.o(.ctors))
    KEEP (*crtbegin?.o(.ctors))
    /* We don't want to include the .ctor section from
       the crtend.o file until after the sorted ctors.
       The .ctor section from the crtend file contains the
       end of ctors marker and it must be last */
    KEEP (*(EXCLUDE_FILE (*crtend.o *crtend?.o ) .ctors))
    KEEP (*(SORT(.ctors.*)))
    KEEP (*(.ctors))
  }
```
也就是连接时，所有的连接输入文件的.ctors节将会被放在一个统一的.ctors节中，放入最终可执行文件中。
3. 如何定位该节
这个在链接时使用的可执行文件就是我们比较常见的crtbegin.o和crtend.o这两个文件，当然大家可能没有注意到过着两个文件，因为通常我们执行g++编译的时候会由编译器来自动添加，这里我就不举比方、打例子了，可以使用g++ -v看一下完整的连接命令。而对应于这两个函数，它的定义在gcc的gcc-4.1.0\gcc\crtstuff.c中，它会处理所有文件中的.ctors和.dctors节，
```c
#ifdef CTOR_LIST_END
CTOR_LIST_END;
#elif defined(CTORS_SECTION_ASM_OP)
/* Hack: force cc1 to switch to .data section early, so that assembling
   __CTOR_LIST__ does not undo our behind-the-back change to .ctors.  */
static func_ptr force_to_data[1] __attribute__ ((__unused__)) = { };
asm (CTORS_SECTION_ASM_OP);
STATIC func_ptr __CTOR_END__[1]
  __attribute__((aligned(sizeof(func_ptr))))
  = { (func_ptr) 0 };
#else
STATIC func_ptr __CTOR_END__[1]
  __attribute__((section(".ctors"), aligned(sizeof(func_ptr))))
  = { (func_ptr) 0 };
#endif
```
这里是一个变量，它主动要求把自己放在.ctors节中，并且初始化了一个定界符，也就是这个万能的0.后面我们会看到，这个链表的遍历是通过从后向前遍历的，所以这个零还是非常重要的。并且这个__CTOR_END是放在crtend.o中的，而这个开始则是放在了.crtbegin中的__CTOR_LIST__链表队列中，有兴趣的同学可以自己确认一下
```c
static void __attribute__((used))
__do_global_ctors_aux (void)
{
  func_ptr *p;
  for (p = __CTOR_END__ - 1; *p != (func_ptr) -1; p--)
    (*p) ();
}
```
这里可以看到把.ctors中的函数作为一个函数指针进行遍历，所以那个初始化是会被执行到的。
4、谁来调用这个__do_global_ctors_aux数组，同样是gcc-4.1.0\gcc\crtstuff.c文件
```c
/* Stick a call to __do_global_ctors_aux into the .init section.  */
CRT_CALL_STATIC_FUNCTION (INIT_SECTION_ASM_OP, __do_global_ctors_aux)
```
这个宏将会展开，它将这个__do_global_ctors_aux放入.init节，然后由crti中的init遍历来完成。
5. init节如何遍历
这个实现位于C库中glibc-2.7\sysdeps\generic\initfini.c
这里的处理使用了脚本，这个文件同样将会生成两个文件，分别是crti.o和crtn.o，它们同样是通过节来完成对各个目标中的init节的夹击的，在真正的_start函数将会调用_init函数，这个函数就卡在所有的init节的两侧从而相当于使用连接器在它们直接加入了所有的连接输入文件的.init节，所以.init节中必须不能主动return，否则会跳过其它节的初始化。
```c
// glibc-2.7\csu\elf-init.c
__libc_csu_init (int argc, char **argv, char **envp)
 ...
  _init ();
```
这里将会调用_init函数。
## 局部变量运行时初始化
函数多线程问题
这里最为简单的思路就是编译器添加伪代码
```c
if(localvar not initialized)
{
initialize localvar
set localvar initialized
}
```
但是这里有一个问题，就是它不是多线程安全的，如果这个函数在if之后被切换并且由另一个函数执行这个代码，那么变量被初始化两次，所以可能会出现我们例子中的foo函数被调用两次。
这里解决的办法和我们写程序实现代码方法相似，那就是加锁，你没有看错，编译将会自动添加mutex互斥锁操作，这里也就是我们看到的__cxa_guard_acquire之类的UFO调用。它的实现位于gcc-4.1.0\libstdc++-v3\libsupc++\guard.cc
这个文件后缀是cc，所以通常的sourceinght可能没有添加，所以大家可以手动打开围观一下。和通常的gnu软件一样，它比较晦涩，但是对于一个已经相当习惯的同学例如我来说，还是没有啥大问题的，所以我就不解释的，知道这里为了防止多线程，gcc直接使用了锁就好了，你也不用担心这里的多线程问题，Let the compiler take care of all these craps。
```c
// The IA64/generic ABI uses the first byte of the guard variable.
// The ARM EABI uses the least significant bit.

// Thread-safe static local initialization support.
#ifdef __GTHREADS
namespace
{
  // static_mutex is a single mutex controlling all static initializations.
  // This is a static class--the need for a static initialization function
  // to pass to __gthread_once precludes creating multiple instances, though
  // I suppose you could achieve the same effect with a template.
  class static_mutex
  {
    static __gthread_recursive_mutex_t mutex;

#ifdef __GTHREAD_RECURSIVE_MUTEX_INIT_FUNCTION
    static void init();
#endif

  public:
    static void lock();
    static void unlock();
  };

  __gthread_recursive_mutex_t static_mutex::mutex
#ifdef __GTHREAD_RECURSIVE_MUTEX_INIT
  = __GTHREAD_RECURSIVE_MUTEX_INIT
#endif
  ;

#ifdef __GTHREAD_RECURSIVE_MUTEX_INIT_FUNCTION
  void static_mutex::init()
  {
    __GTHREAD_RECURSIVE_MUTEX_INIT_FUNCTION (&mutex);
  }
#endif

  void static_mutex::lock()
  {
#ifdef __GTHREAD_RECURSIVE_MUTEX_INIT_FUNCTION
    static __gthread_once_t once = __GTHREAD_ONCE_INIT;
    __gthread_once (&once, init);
#endif
    __gthread_recursive_mutex_lock (&mutex);
  }

  void static_mutex::unlock ()
  {
    __gthread_recursive_mutex_unlock (&mutex);
  }
}

#ifndef _GLIBCXX_GUARD_TEST_AND_ACQUIRE
inline bool
__test_and_acquire (__cxxabiv1::__guard *g)
{
  bool b = _GLIBCXX_GUARD_TEST (g);
  _GLIBCXX_READ_MEM_BARRIER;
  return b;
}
#define _GLIBCXX_GUARD_TEST_AND_ACQUIRE(G) __test_and_acquire (G)
#endif

#ifndef _GLIBCXX_GUARD_SET_AND_RELEASE
inline void
__set_and_release (__cxxabiv1::__guard *g)
{
  _GLIBCXX_WRITE_MEM_BARRIER;
  _GLIBCXX_GUARD_SET (g);
}
#define _GLIBCXX_GUARD_SET_AND_RELEASE(G) __set_and_release (G)
#endif

#else /* !__GTHREADS */

#undef _GLIBCXX_GUARD_TEST_AND_ACQUIRE
#undef _GLIBCXX_GUARD_SET_AND_RELEASE
#define _GLIBCXX_GUARD_SET_AND_RELEASE(G) _GLIBCXX_GUARD_SET (G)

#endif /* __GTHREADS */

namespace __gnu_cxx
{
  // 6.7[stmt.dcl]/4: If control re-enters the declaration (recursively)
  // while the object is being initialized, the behavior is undefined.

  // Since we already have a library function to handle locking, we might
  // as well check for this situation and throw an exception.
  // We use the second byte of the guard variable to remember that we're
  // in the middle of an initialization.
  class recursive_init: public std::exception
  {
  public:
    recursive_init() throw() { }
    virtual ~recursive_init() throw ();
  };

  recursive_init::~recursive_init() throw() { }
}

namespace __cxxabiv1 
{
  static inline int
  recursion_push (__guard* g)
  {
    return ((char *)g)[1]++;
  }

  static inline void
  recursion_pop (__guard* g)
  {
    --((char *)g)[1];
  }

  static int
  acquire_1 (__guard *g)
  {
    if (_GLIBCXX_GUARD_TEST (g))
      return 0;

    if (recursion_push (g))
      {
#ifdef __EXCEPTIONS
    throw __gnu_cxx::recursive_init();
#else
    // Use __builtin_trap so we don't require abort().
    __builtin_trap ();
#endif
      }
    return 1;
  }

  extern "C"
  int __cxa_guard_acquire (__guard *g) 
  {
#ifdef __GTHREADS
    // If the target can reorder loads, we need to insert a read memory
    // barrier so that accesses to the guarded variable happen after the
    // guard test.
    if (_GLIBCXX_GUARD_TEST_AND_ACQUIRE (g))
      return 0;

    if (__gthread_active_p ())
      {
    // Simple wrapper for exception safety.
    struct mutex_wrapper
    {
      bool unlock;
      mutex_wrapper (): unlock(true)
      {
        static_mutex::lock ();
      }
      ~mutex_wrapper ()
      {
        if (unlock)
          static_mutex::unlock ();
      }
    } mw;

    if (acquire_1 (g))
      {
        mw.unlock = false;
        return 1;
      }

    return 0;
      }
#endif

    return acquire_1 (g);
  }

  extern "C"
  void __cxa_guard_abort (__guard *g)
  {
    recursion_pop (g);
#ifdef __GTHREADS
    if (__gthread_active_p ())
      static_mutex::unlock ();
#endif
  }

  extern "C"
  void __cxa_guard_release (__guard *g)
  {
    recursion_pop (g);
    _GLIBCXX_GUARD_SET_AND_RELEASE (g);
#ifdef __GTHREADS
    if (__gthread_active_p ())
      static_mutex::unlock ();
#endif
  }
}
```