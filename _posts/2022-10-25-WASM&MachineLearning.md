---
layout: single
title:  "WASM和机器学习"
date:   2022-10-25 11:00
author_profile: false
---

# WebAssembly和机器学习

![webassembly](/assets/images/posts/webassembly.png)

## [什么是WebAssembly？](https://developer.mozilla.org/en-US/docs/WebAssembly/Concepts#what_is_webassembly)


WebAssembly 是一种可以在现代Web浏览器中运行的低级的类汇编语言，具有紧凑的二进制格式，接近本机的性能运行的。为了实现代码紧凑WebAssembly 被设计成了不容易手写，但是支持C、C++、C#、Golang、Rust 等源语言编写代码，使用相应工具链翻译源语言代码。

![webassembly](/assets/images/posts/webassembly-1.png)

WebAssembly旨在补充并与JavaScript一起运行，使用 WebAssemblyJavaScript API，你可以将WebAssembly模块加载到 JavaScript 应用程序中并在两者之间共享功能。这使您可以在相同的应用程序中利用WebAssembly的性能和功能以及 JavaScript 的表现力和灵活性。WebAssembly 模块甚至可以导入Node.js应用程序中来提供高性能的服务。

## 为什么需要 WebAssembly？

从历史上看，Web浏览器的VM 只能加载 JavaScript。这对我们来说效果很好，因为 JavaScript 足够强大，可以解决当今人们在 Web 上遇到的大多数问题。然而，当我们尝试将 JavaScript 用于更密集的用例时，例如 3D 游戏、虚拟和增强现实、计算机视觉、图像/视频编辑以及许多其他需要本机性能的领域时，我们遇到了性能问题。

## W3C WebAssembly 标准

[WebAssembly Core Specification](https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/)定义了一个低级虚拟机，模拟运行该虚拟机的许多微处理器的功能。 通过即时编译或解析，WebAssembly 引擎使编写的代码可以以接近本地平台的速度运行。.wasm 资源类似于 Java .class 文件，它包含静态数据和对该静态数据进行操作的代码段。

[WebAssembly Web API](https://www.w3.org/TR/2019/REC-wasm-web-api-1-20191205/)定义了一个基于 Promise 的接口，用于请求和执行 .wasm 资源。 .wasm 资源的结构经过优化，允许在检索整个资源之前开始执行。

[WebAssembly JavaScript Interface](https://www.w3.org/TR/2019/REC-wasm-js-api-1-20191205/)提供了一个 JavaScript 接口，用于调用并将参数传递给 WebAssembly 函数。 在 Web 浏览器中，WebAssembly 与主机环境的交互都通过 JavaScript 进行管理。

## [WebAssembly 关键概念](https://developer.mozilla.org/en-US/docs/WebAssembly/Concepts#webassembly_key_concepts)

了解 WebAssembly 如何在浏览器中运行需要几个关键概念。[所有这些概念都在](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/WebAssembly)[WebAssembly JavaScript API](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/WebAssembly)中 1:1 反映。

Module ： 表示已被浏览器编译成可执行机器代码的 WebAssembly 二进制文件。Module 是无状态的，因此，像
一样[Blob](https://developer.mozilla.org/en-US/docs/Web/API/Blob)，可以在 windows 和 worker 之间显式共享（通过[postMessage()](https://developer.mozilla.org/en-US/docs/Web/API/MessagePort/postMessage)）。Module 声明导入和导出就像 ES 模块一样。

Memory ： 一个可调整大小的ArrayBuffer，其中包含由WebAssembly的低级内存访问指令读取和写入的线性字节数组。

Table ： 一个可调整大小的类型化引用数组（例如，函数），否则无法将其作为原始字节存储在内存中（出于安全和可移植性原因）。

Instance ：一个模块与它在运行时使用的所有状态配对，包括内存、表和一组导入的值。Instance 就像一个 ES 模块，它已通过一组特定的导入加载到特定的全局中。

JavaScript API 为开发人员提供了创建模块(Module)、内存(Memory)、表(Table)和实例(Instance)的能力。给定一个WebAssembly实例，JavaScript代码可以同步调用它的导出，这些导出作为普通的JavaScript函数公开。通过将这些JavaScript函数作为导入传递给WebAssembly实例，WebAssembly代码也可以同步调用任意JavaScript函数。

## WebAssembly 工作流程

如果您使用C/C++，您可能使用过gcc或类似的编译器。为了获得Webassembly二进制文件，我们需要一些其他特殊的编译器。可用的不止一种，但目前最好的一种是Emscripten它是开源的。
与"普通"汇编语言不同，Webassembly 不是特定于 CPU 的，因此可以在多个平台上运行，从手机等嵌入式系统到计算机的 CPU。
一旦我们用 Emscripten 编译了我们的 C/C++ 代码，我们就获得了一个可以在浏览器上运行的合适的 WASM 文件，很简单吧？
实际上，还有更多细节需要考虑，但我们将逐步介绍它们。
WASM WebApp 工作的步骤是：

1. 使用Emscripten编译C/C++代码，以获得WASM二进制文件。

2. 使用JavaScript"胶水代码"将WASM二进制文件绑定到页面。

3. 运行您的应用程序并让浏览器实例化您的WASM模块、内存和引用表。一旦完成，您的WebApp就可以完全运行了。

![webassembly](/assets/images/posts/webassembly-2.png)

### [从C/C++代码生成](https://developer.mozilla.org/en-US/docs/WebAssembly/Concepts#porting_from_cc)

![c/c++](/assets/images/posts/webassembly-cplusplus.png)

Emscripten 首先将 C/C++ 输入到 clang+LLVM( C/C++ 编译器工具链)，将C/C++代码编译成.wasm 二进制文件。

WebAssembly 本身目前无法直接访问 DOM；它只能调用 JavaScript，传入整数和浮点原始数据类型。因此，要访问任何 Web API，WebAssembly 需要调用 JavaScript，然后 JavaScript 会调用 Web API。因此，Emscripten 创建了实现此目的所需的 HTML 和 JavaScript 粘合代码。

要使 WebAssembly 可用，我们需要两个主要组件：将代码编译成 WebAssembly 的工具链，以及可以执行该输出的浏览器。这两个组件都依赖于完成 WebAssembly[规范](https://github.com/WebAssembly/spec)的进展，但除此之外，很大程度上是独立的工程工作。这种分离是一件好事，因为它将使编译器能够发出在任何浏览器中运行的 WebAssembly，并且无论是哪个编译器生成它，浏览器都可以运行 WebAssembly；换句话说，它允许多个工具链和多个浏览器协同工作，改善用户选择。分离还允许两个组件的工作立即并行进行。

```shell
git clone https://github.com/emscripten-core/emsdk.git

cd emsdk

./emsdk install latest

./emsdk activate latest

source ./emsdk\_env.sh
```

在你的IDE中复制如下代码：
```cpp
// hello.c
#include <stdio.h>
int main() {
    printf("hello, world!\n");
    return 0;
}
```

然后编译这个文件
```shell
emcc hello.c -o hello.js
```
您可以使用[node.js](https://emscripten.org/docs/site/glossary.html#term-node-js)运行它们：
```shell
node hello.js
```
Emscripten 还可以生成用于测试嵌入式 JavaScript 的 HTML。要生成 HTML，请使用-o( [output](https://emscripten.org/docs/tools_reference/emcc.html#emcc-o-target) ) 命令并指定一个 html 文件作为目标文件：
```shell
emcc hello.c -O3 -o hello.html
```
请注意，除了发出WebAssembly之外，我们在此模式下发出的构建通常使用Emscripten工具链中的所有其他内容：Emscripten的musl libc端口和访问它的系统调用、OpenGL/WebGL 代码、浏览器集成代码、node.js 集成代码，等等。因此，它支持Emscripten已经做的所有事情，并且使用Emscripten的现有项目只需轻按一下开关即可切换到发出WebAssembly。这是让现有的 C++ 项目在WebAssembly启动时从WebAssembly中受益的关键部分，而他们几乎不需要付出任何努力。

### WebAssembly 在机器学习中的应用

仅仅使用 WebAssembly 协议很难满足机器学习所需的各种矩阵运算所需的计算指令，因此有了很多不同补充协议实现协机器学习的功能，下面介绍三种比较主流的实现方式WebAssembly SIMD, WASI-NN, Apache TVM。

**WebAssembly SIMD**

SIMD代表单指令多数据。SIMD 指令是一类特殊的指令，它通过同时对多个数据元素执行相同的操作来利用应用程序中的数据并行性。音频/视频编解码器、图像处理器等计算密集型应用程序都是利用 SIMD 指令来加速性能的应用程序示例。大多数现代架构支持 SIMD 指令的一些变体。

WebAssembly SIMD 提案中包含的一组操作由在各种平台上得到很好支持的操作组成，并且被证明是高性能的。为此，目前的提议仅限于标准化固定宽度 128 位 SIMD 操作。

当前的提议引入了一种新的v128值类型，以及对这种类型进行操作的许多新操作。用于确定这些操作的标准是：

这些操作应该在多个现代架构中得到很好的支持。

在一个指令组内的多个相关架构中，性能优势应该是积极的。

所选的一组操作应尽量减少性能悬崖（如果有）。

**在Chrome 中的 SIMD 支持**

默认情况下，Chrome 91 提供 WebAssembly SIMD 支持。确保使用最新版本的工具链，如下所述，以及最新的 wasm-feature-detect 来检测支持最终版本规范的引擎。如果某些地方看起来不正确，请[提交错误](https://crbug.com/v8)。

**在 Firefox 中启用实验性 SIMD 支持**

WebAssembly SIMD 在 Firefox 的标志后面可用。目前它仅在 x86 和 x86-64 架构上受支持。要在 Firefox 中试用 SIMD 支持，请转到about:config并启用javascript.options.wasm\_simd. 请注意，此功能仍处于试验阶段，正在开发中。

# WASI-NN

如同WASM SIMD一样，[wasi-nn](https://github.com/WebAssembly/wasi-nn)也允许WebAssembly 程序访问主机提供的机器学习(ML)功能。为保持足够的兼容性WASM SIMD只能使用CPU作为计算但愿，但是许多机器学习模型利用了一些其他辅助处理单元（例如GPU、TPU）。目前[很难](https://github.com/WebAssembly/design/issues/273)找到一种合适的方法使用WASM编译到这样的设备上的，因此在WASM基础上提供一种使用这些设备的方法，wasi-nn就是为了实现这一目的而被设计出来的更高级别的 API 。

最后，将 ML 推理部署到 Wasm 运行时已经足够困难了，而无需将翻译的复杂性添加到较低级别的抽象中。因此，wasi-nn允许程序员直接部署模型，将针对适当设备编译模型的工作转移到其他工具（例如OpenVINO、TF）。如果在某个时候有一个WASM提案可以使用机器的完整 ML 性能（例如灵活向量、GPU），那么可以想象，wasi-nn可以仅使用WASM原语"在后台"实现——直到到那时，ML程序员仍然可以使用此处描述的方法执行推理，并且如果发生这种切换，应该会看到最小的变化。

经过训练的机器学习模型都会被部署在具有不同体系结构和操作系统类型的各类不同设备上。而借助于 wasi-nn，.wasm 文件便能够以一种可移植的方式去执行诸如"描述张量"及"执行推理请求"等操作，而无视底层具体的指令集架构（ISA）及操作系统（OS）差异。

![tvm](/assets/images/posts/tvm-1.png)

### Apache TVM
![Apache TVM 将机器学习编译为 WASM 和 WebGPU](https://www.cnblogs.com/wujianming-110117/p/14811667.html)

在Apache TVM深度学习编译器中引入了WASM和WebGPU的支持。实验表明，在将模型部署到Web时，TVM的WebGPU后端可以接近本机 GPU的性能。

![tvm](/assets/images/posts/tvm-2.png)

计算是现代机器学习应用程序的支柱之一。GPU的引入加快了深度学习的工作量，极大地提高了运行速度。部署机器学习的需求不断增长，浏览器已成为部署智能应用程序的自然之所。

TensorFlow.js和ONNX.js将机器学习引入浏览器，但是由于缺乏对Web上GPU的标准访问和高性能访问的方式，他们使用了WASM SIMD优化CPU计算，通过过WebGL提供GPU计算部分。但是WebGL缺少高性能着色学习所需的重要功能，例如计算着色器和通用存储缓冲区。

WebGPU是下一代Web图形标准。与最新一代的图形API（例如Vulkan和Metal）一样，WebGPU提供了一流的计算着色器支持。

为了探索在浏览器中使用WebGPU进行机器学习部署的潜力，增强了深度学习编译器Apache（incubating）TVM，以WASM（用于计算启动参数并调用设备启动的主机代码）和WebGPU（用于设备）为目标。使用TVM在Web上部署机器学习应用程序时，仍能接近GPU的本机性能。

![tvm](/assets/images/posts/tvm-3.png)

WebGPU的传统工作流程是为深度神经网络（矩阵乘法和卷积）中的原始算子编写着色器，然后直接优化性能。这是现有框架（TensorFlow.js）最新版本中使用了这种工作模式。

TVM则与之相反，采用了基于编译的方法。TVM自动从TensorFlow，Keras，PyTorch，MXNet和ONNX等高级框架中提取模型，使用机器学习驱动的方法自动生成低级代码，在这种情况下，将以SPIR-V格式计算着色器。然后可以为可部署模块生成的代码打包。

编译的方法的一个重要优点是基础架构的重用。通过重用基础结构来优化CUDA，Metal和OpenCL等本机平台的GPU内核，能够轻松地以Web为目标。如果WebGPU API到本机API的映射有效，可以通过很少的工作获得类似的性能。更重要的是，[AutoTVM](https://tvm.apache.org/2018/10/03/auto-opt-all)基础架构，能够针对特定模型专门化计算着色器，从而能够为感兴趣的特定模型生成最佳的计算着色器。

TVM已经有Vulkan的SPIR-V目标，使用LLVM生成主机代码。可以仅将二者的用途重新生成设备和主机程序。

主要挑战是runtime。需要一个runtime来加载着色器代码，并使主机代码对话能够正确地与着色器通信。TVM具有最低的基于C ++的runtime。构建了一个最小的Web runtime库，生成的着色器和主机驱动代码链接，生成一个WASM文件。但是，此WASM模块仍然包含两个未知的依赖项：

runtime需要调用系统库调用（malloc，stderr）。

wasmruntime需要与WebGPU驱动程序进行交互（在Javascript中，WebGPU API是the first-class citizen）。

WASI是解决第一个问题的标准解决方案。尽管网络上还没有成熟的WASI，使用Emscripten生成类似WASI的库，提供这些系统库。

通过在TVM的JS runtime内部构建WebGPU runtime来解决第二个问题，在调用GPU代码时，从WASM模块中回调这些功能。使用TVM runtime系统中的[PackedFunc](https://tvm.apache.org/docs/dev/runtime.html#packedfunc)机制，可以通过将JavaScript闭包传递到WASM接口，直接公开高级runtime原语。这种方法将大多数runtime代码保留在JavaScript中，随着WASI和WASM支持的成熟，可以将更多JS代码引入WASM runtime。  ![](RackMultipart20221025-1-lpyni_html_375fa580a4159c19.png)

未来的某个时候，当WebGPU成熟，通过WASI标准化时，可以将其定位为WebGPU的本机API，使用WebGPU的独立WASM应用程序。

## Reference

[WebAssembly-Wiki](https://zh.wikipedia.org/wiki/WebAssembly)

[WebAssembly](https://developer.mozilla.org/zh-CN/docs/WebAssembly)

[WebAssembly – Where is it going?](https://opencredo.com/blogs/webassembly-where-is-it-going/)

[WebAssembly SIMD](https://v8.dev/features/simd)

[WasmEdge](https://wasmedge.org/)

[WebGPU-wiki](https://en.wikipedia.org/wiki/WebGPU)

[Apache TVM](https://tvm.apache.org/docs/)

## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io