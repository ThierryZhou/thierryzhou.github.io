---
title:  "C++面试题"
date:   2022-11-3 11:00
---
## C++ 基础

1. 引用和指针有什么区别？
一般指的是某块内存的地址，通过这个地址，我们可以寻址到这块内存；而引用是一个变量的别名。指针可以为空，引用不能为空。

2. #define, extern, static和const有什么区别？
#define主要是用于定义宏，编译器编译时做相关的字符替换工作，主要用来增加代码可读性；const定义的数据在程序开始前就在全局变量区分配了空间，生命周期内其值不可修改；static修饰局部变量时，该变量便存放在静态数据区，其生命周期一直持续到整个程序执行结束，static修饰全局变量，全局变量在本源文件中被访问到，也可以在同一个工程的其它源文件中被访问；extern用在变量或者函数的声明前，用来说明“此变量/函数是在别处定义的，要在此处引用”。

3. 静态链接和动态链接有什么区别？
静态链接，无论缺失的地址位于其它目标文件还是链接库，链接库都会逐个找到各目标文件中缺失的地址。采用此链接方式生成的可执行文件，可以独立载入内存运行；  
动态链接，链接器先从所有目标文件中找到部分缺失的地址，然后将所有目标文件组织成一个可执行文件。如此生成的可执行文件，仍缺失部分函数和变量的地址，待文件执行时，需连同所有的链接库文件一起载入内存，再由链接器完成剩余的地址修复工作，才能正常执行。

4. 变量的声明和定义有什么区别
变量的定义：用于为变量分配存储空间，还可以为变量指定初始值。在一个程序中，变量有且仅有一个定义。  
变量的声明：用于向程序表明变量的类型和名字。程序中变量可以声明多次，但只能定义一次。   

5. strcmp, strlen, strcpy等函数的源码
```c
int
STRCMP (const char *p1, const char *p2)
{
  const unsigned char *s1 = (const unsigned char *) p1;
  const unsigned char *s2 = (const unsigned char *) p2;
  unsigned char c1, c2;
  do
    {
      c1 = (unsigned char) *s1++;
      c2 = (unsigned char) *s2++;
      if (c1 == '\0')
	return c1 - c2;
    }
  while (c1 == c2);
  return c1 - c2;
}

char *
STRCPY (char *dest, const char *src)
{
  return memcpy (dest, src, strlen (src) + 1);
}

size_t
STRLEN (const char *str)
{
  const char *char_ptr;
  const unsigned long int *longword_ptr;
  unsigned long int longword, himagic, lomagic;
  /* Handle the first few characters by reading one character at a time.
     Do this until CHAR_PTR is aligned on a longword boundary.  */
  for (char_ptr = str; ((unsigned long int) char_ptr
			& (sizeof (longword) - 1)) != 0;
       ++char_ptr)
    if (*char_ptr == '\0')
      return char_ptr - str;
  /* All these elucidatory comments refer to 4-byte longwords,
     but the theory applies equally well to 8-byte longwords.  */
  longword_ptr = (unsigned long int *) char_ptr;
  /* Bits 31, 24, 16, and 8 of this number are zero.  Call these bits
     the "holes."  Note that there is a hole just to the left of
     each byte, with an extra at the end:
     bits:  01111110 11111110 11111110 11111111
     bytes: AAAAAAAA BBBBBBBB CCCCCCCC DDDDDDDD
     The 1-bits make sure that carries propagate to the next 0-bit.
     The 0-bits provide holes for carries to fall into.  */
  himagic = 0x80808080L;
  lomagic = 0x01010101L;
  if (sizeof (longword) > 4)
    {
      /* 64-bit version of the magic.  */
      /* Do the shift in two steps to avoid a warning if long has 32 bits.  */
      himagic = ((himagic << 16) << 16) | himagic;
      lomagic = ((lomagic << 16) << 16) | lomagic;
    }
  if (sizeof (longword) > 8)
    abort ();
  /* Instead of the traditional loop which tests each character,
     we will test a longword at a time.  The tricky part is testing
     if *any of the four* bytes in the longword in question are zero.  */
  for (;;)
    {
      longword = *longword_ptr++;
      if (((longword - lomagic) & ~longword & himagic) != 0)
	{
	  /* Which of the bytes was the zero?  If none of them were, it was
	     a misfire; continue the search.  */
	  const char *cp = (const char *) (longword_ptr - 1);
	  if (cp[0] == 0)
	    return cp - str;
	  if (cp[1] == 0)
	    return cp - str + 1;
	  if (cp[2] == 0)
	    return cp - str + 2;
	  if (cp[3] == 0)
	    return cp - str + 3;
	  if (sizeof (longword) > 4)
	    {
	      if (cp[4] == 0)
		return cp - str + 4;
	      if (cp[5] == 0)
		return cp - str + 5;
	      if (cp[6] == 0)
		return cp - str + 6;
	      if (cp[7] == 0)
		return cp - str + 7;
	    }
	}
    }
}
```
6. volatile 和 mutable 有什么作用
在C++中，mutable是为了突破const的限制而设置的。被mutable修饰的变量，将永远处于可变的状态，即使在一个const函数中，甚至结构体变量或者类对象为const，其mutable成员也可以被修改。  
象const一样，volatile是一个类型修饰符。volatile修饰的数据,编译器不可对其进行执行期寄存于寄存器的优化。这种特性,是为了满足多线程同步、中断、硬件编程等特殊需要。遇到这个关键字声明的变量，编译器对访问该变量的代码就不再进行优化，从而可以提供对特殊地址的直接访问。

7. 全局变量和局部变量有什么区别？操作系统和编译器是怎么知道的？
全局变量是整个程序都可访问的变量，生存期从程序开始到程序结束；局部变量存在于模块中(比如某个函数)，只有在模块中才可以访问，生存期从模块开始到模块结束。  
全局变量分配在全局数据段，在程序开始运行的时候被加载。局部变量则分配在程序的堆栈中。因此，操作系统和编译器可以通过内存分配的位置来知道来区分全局变量和局部变量。

8. shared_ptr, weak_ptr, unique_ptr分别是什么？
unique_ptr 实现独占式拥有或严格拥有的智能指针，通过禁用拷贝构造和赋值的方式保证同一时间内只有一个智能指针可以指向该对象；shared_ptr增加了引用计数，每次有新的shared_ptr指向同一个资源时计数会增加，当计数为0时自动释放资源；构造新的weak_ptr指针不会增加shared_ptr的引用计数，是用来解决shared_ptr循环引用的问题。

9. RAII是什么？
RAII技术的核心是获取完资源就马上交给资源管理。标准库中的智能指针和锁便是比较常用的RAII工具。RAII类需要慎重考虑资源拷贝的合理性。

10. 右值引用有什么作用？
普通引用为左值引用，无法指向右值，但是const左值引用可以指向右值；右值引用指向的是右值，本质上也是把右值提升为一个左值，并定义一个右值引用通过std::move指向该左值。右值引用和std::move被广泛用于在STL和自定义类中实现移动语义，避免拷贝，从而提升程序性能。 

11. 函数重载和函数重写
重写（覆盖）的规则：
1、重写方法的参数列表必须完全与被重写的方法的相同,否则不能称其为重写而是重载。  
2、重写方法的访问修饰符一定要大于被重写方法的访问修饰符（public>protected>default>private）。  
3、重写的方法的返回值必须和被重写的方法的返回一致。  
4、重写的方法所抛出的异常必须和被重写方法的所抛出的异常一致，或者是其子类。  
5、被重写的方法不能为private，否则在其子类中只是新定义了一个方法，并没有对其进行重写。  
6、静态方法不能被重写为非静态的方法（会编译出错）。
重载的规则：
1、在使用重载时只能通过相同的方法名、不同的参数形式实现。不同的参数类型可以是不同的参数类型，不同的参数个数，不同的参数顺序（参数类型必须不一样）。  
2、不能通过访问权限、返回类型、抛出的异常进行重载。  
3、方法的异常类型和数目不会对重载造成影响。

12. C++的顶层const和底层const？
顶层 const 表示指针本身是个常量；
底层 const 表示指针所指的对象是一个常量。

13. 拷贝初始化、直接初始化、列表初始化?
直接初始化实际上是要求编译器使用普通的函数匹配来选择与我们提供的参数最匹配的构造函数。  
拷贝初始化实际上是要求编译器将右侧运算对象拷贝到正在创建的对象中，通常用拷贝构造函数来完成。  
C++11标准中{}的初始化方式是对聚合类型的初始化，是以拷贝的形式来赋值的。

## C++面向对象
1. 纯虚函数和虚函数表
如果类中存在虚函数，那么该类的大小就会多4个字节，然而这4个字节就是一个指针的大小，这个指针指向虚函数表，这个指针将被放置与类所有成员之前。对于多重继承的派生类来说，它含有与父类数量相对应的虚函数指针。

2. 为什么基类的构造函数不能定义为虚函数？
从存储空间角度，虚函数对应一个指向vtable虚函数表的指针，这大家都知道，可是这个指向vtable的指针其实是存储在对象的内存空间的。问题出来了，如果构造函数是虚的，就需要通过 vtable来调用，可是对象还没有实例化，也就是内存空间还没有，怎么找vtable呢？所以构造函数不能是虚函数。

从使用角度，虚函数主要用于在信息不全的情况下，能使重载的函数得到对应的调用。构造函数本身就是要初始化实例，那使用虚函数也没有实际意义呀。所以构造函数没有必要是虚函数。虚函数的作用在于通过父类的指针或者引用来调用它的时候能够变成调用子类的那个成员函数。而构造函数是在创建对象时自动调用的，不可能通过父类的指针或者引用去调用，因此也就规定构造函数不能是虚函数。

构造函数不需要是虚函数，也不允许是虚函数，因为创建一个对象时我们总是要明确指定对象的类型，尽管我们可能通过实验室的基类的指针或引用去访问它但析构却不一定，我们往往通过基类的指针来销毁对象。这时候如果析构函数不是虚函数，就不能正确识别对象类型从而不能正确调用析构函数。

从实现上看，vbtl在构造函数调用后才建立，因而构造函数不可能成为虚函数从实际含义上看，在调用构造函数时还不能确定对象的真实类型（因为子类会调父类的构造函数）；而且构造函数的作用是提供初始化，在对象生命期只执行一次，不是对象的动态行为，也没有必要成为虚函数。

当一个构造函数被调用时，它做的首要的事情之一是初始化它的VPTR。因此，它只能知道它是“当前”类的，而完全忽视这个对象后面是否还有继承者。当编译器为这个构造函数产生代码时，它是为这个类的构造函数产生代码——既不是为基类，也不是为它的派生类（因为类不知道谁继承它）。所以它使用的VPTR必须是对于这个类的VTABLE。而且，只要它是最后的构造函数调用，那么在这个对象的生命期内，VPTR将保持被初始化为指向这个VTABLE, 但如果接着还有一个更晚派生的构造函数被调用，这个构造函数又将设置VPTR指向它的 VTABLE，等.直到最后的构造函数结束。VPTR的状态是由被最后调用的构造函数确定的。这就是为什么构造函数调用是从基类到更加派生类顺序的另一个理由。但是，当这一系列构造函数调用正发生时，每个构造函数都已经设置VPTR指向它自己的VTABLE。如果函数调用使用虚机制，它将只产生通过它自己的VTABLE的调用，而不是最后的VTABLE（所有构造函数被调用后才会有最后的VTABLE）。

3. 什么时候需要定义虚析构函数？
一般基类的虚成员函数，子类重载的时候要求是完全一致，也就是除了函数体，都要一毛一样。而析构函数同样也是成员函数，虚析构函数也会进入虚表，唯一不同的是，函数名并不要求一致，而且，你如果不写，编译器也会帮你生成，而且如果基类有virtual，编译器也会默认给子类添加。但是不论如何它依旧遵守多态的规则，也就是说，如果你的析构函数是虚函数，调用虚函数的规则也遵守多态原则，也就是会调用子类的析构函数，这和其他虚函数的机制完全一致，并没有什么不同。而子类析构函数具有析构掉基类的职责，所以不会造成内存泄漏。而基类并不知道自己的子类。

4. 构造函数和析构函数能抛出异常吗？
不能。

5. 多继承存在什么问题？如何消除多继承中的二义性？
在继承时，基类之间或基类与派生类之间发生成员同名时，将出现对成员访问的不确定性，即同名二义性。解决二义性的方案：利用作用域运算符::，用于限定派生类使用的是哪个基类的成员；在派生类中定义同名成员，覆盖基类中的相关成员。

6. 如果类A是一个空类，那么sizeof(A)的值为多少？
1

7.  类型转换分为哪几种？各自有什么样的特点？

8.  RTTI是什么？其原理是什么？

9.  说一说c++中四种cast转换

10. C++的空类有哪些成员函数

11. 模板函数和模板类的特例化

12. 为什么析构函数一般写成虚函数

## C++ STL
1. vector, array, deque 的区别

2. Vector如何释放空间?

3. 如何在共享内存上使用STL标准库？

4.  map 、set、multiset、multimap 底层原理及其相关面试题

5.  unordered_map、unordered_set 底层原理及其相关面试题

6.  迭代器的底层机制和失效的问题

7.  为什么vector的插入操作可能会导致迭代器失效？

8.  vector的reserve()和resize()方法之间有什么区别？

9.  vector越界访问下标，map越界访问下标？vector删除元素时会不会释放空间？

10. STL内存优化？

11. emplace和push的区别？

## C++内存管理
1. 内存块太小导致malloc和new返回空指针，该怎么处理？

2. 内存泄漏的场景有哪些？

3. 内存的分配方式有几种？

4. 堆和栈有什么区别？

5. 静态内存分配和动态内存分配有什么区别？

6. 如何构造一个类，使得只能在堆上或只能在栈上分配内存？

7. 浅拷贝和深拷贝有什么区别？

8.  字节对齐的原则是什么？

9.  结构体内存对齐问题