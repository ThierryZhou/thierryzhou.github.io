---
title: Kubernetes Istio详解
tag: kubernetes
---

## Service Mesh概述



## Istio 概述

Istio服务网格在逻辑上分为数据平面和控制平面。

数据平面由部署为Sidecar的一组智能代理(Envoy+Polit-agent)组成。这些代理承载并控制微服务之间的所有网络通信。他们还收集并报告所有服务网格流量的遥测。
控制平面管理和配置代理以路由流量。
下图大致展示了组成每个平面的所有组件：


下面我们详细讲述一下各个组件。

数据平面
Envoy 和 pilot-agent 打在同一个镜像中，即Proxy。

Envoy

目前Istio使用Envoy代理的扩展版本。 Envoy是使用C++开发的高性能代理，可为服务网格中的所有服务路由控制所有入站和出站流量。 Envoy代理是与数据平面流量交互的唯一Istio组件。

目前Istio使用的不是Envoy官方版本，不过随着Envoy对于wasm的支持已经merge到了master，也许未来的istio的版本会直接使用官方版本。
Envoy代理被部署为服务的Sidecar，通过Envoy的许多内置功能在逻辑上增强了服务，例如：

动态服务发现
负载均衡
TLS终止
HTTP / 2和gRPC代理
断路器
健康检查
分阶段升级，按百分比分配流量
故障注入
丰富的指标
这种Sidecar部署使Istio可以执行策略决策并提取丰富的遥测数据，然后将其发送到监视系统以提供有关整个网格行为的信息。

而且还允许您将Istio功能添加到现有部署中，而无需重新构造或重写代码。

在云原生的体系中，这种Sidecar的思维，实际上符合了软件工程解耦的思想，注定会越来越流行，最近流行的mecha架构印证了这一点。

Pilot-agent

pilot-agent负责的工作包括：

生成envoy的配置
启动envoy
监控并管理envoy的运行状况，比如envoy出错时pilot-agent负责重启envoy，或者envoy配置变更后reload envoy
istio-init or cni

Istio需要透明拦截服务的进站流量，需要用到这两个组件之一。istio-init 和 cni 实现底层原理没什么区别，均为写入iptables，让该 pod 所在的 network namespace 的网络流量转发到 proxy 进程。

默认情况下，Istio在网格中部署的Pod中注入一个initContainer istio-init。 istio-init容器设置了与Istio Sidecar代理之间的Pod网络流量重定向。这要求该Pod用户或服务帐户，必须具有足够的Kubernetes RBAC权限才能部署具有NET_ADMIN和NET_RAW功能的容器。对于某些组织的安全合规性，要求Istio用户具有提升的Kubernetes RBAC权限是有问题的。 Istio CNI插件是执行相同网络功能但不需要Istio用户启用提升的Kubernetes RBAC权限的istio-init容器的替代。

不过目前很多公有云托管的k8s并没有对istio cni进行完善的测试，所以生产环境使用的不是很多。
控制平面
从istio1.5 开始，简化了控制平面，将先前由 Pilot，Galley，Citadel 和 sidecar 注入器执行的功能统一为一个二进制文件istiod 。其提供服务发现，配置和证书管理功能。

Pilot

将控制流量行为的高级路由规则转换为Envoy特定的配置，并在运行时将其下发到Sidecar。 Pilot提取了特定于平台的服务发现机制，并将它们合成为标准格式，任何符合Envoy API的Sidecar都可以使用。

目前蚂蚁金服开源的mosn 就可以开箱即用，替代envoy。
Istio 提供的流量管理功能，主要是由pilot支持。

Galley

Galley提供配置管理的服务。实现原理是通过k8s提供的ValidatingWebhook对配置进行验证。比如我们在部署istio的时候，会部署如下的ValidatingWebhook ：

apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: istiod-istio-system
  labels:
    app: istiod
    release: istio
    istio: istiod
webhooks:
  - name: validation.istio.io
    clientConfig:
      service:
        name: istiod
        namespace: istio-system
        path: "/validate"
      caBundle: "" # patched at runtime when the webhook is ready.
    rules:
      - operations:
        - CREATE
        - UPDATE
        apiGroups:
        - config.istio.io
        - security.istio.io
        - authentication.istio.io
        - networking.istio.io
        apiVersions:
        - "*"
        resources:
        - "*"
    # Fail open until the validation webhook is ready. The webhook controller
    # will update this to `Fail` and patch in the `caBundle` when the webhook
    # endpoint is ready.
    failurePolicy: Ignore
    sideEffects: None
    admissionReviewVersions: ["v1beta1", "v1"]
Galley使Istio可以与Kubernetes之外的其他环境一起工作，因为它可以将不同的配置数据转换为Istio可以理解的通用格式。比如Consul。

Citadel

负责处理系统上不同服务之间的TLS通信。 Citadel充当证书颁发机构(CA)，并生成证书以允许在数据平面中进行安全的mTLS通信。

istio-sidecar-injector

该功能实际上通过k8s提供的MutatingWebhook功能实现，当我们创建Pod的时候，会拦截创建动作，并进行判断是否执行注入操作，比如我们在部署istio的时候，会部署如下的MutatingWebhook ：

apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: istio-sidecar-injector

  labels:
    istio.io/rev: default
    app: sidecar-injector
    release: istio
webhooks:
  - name: sidecar-injector.istio.io
    clientConfig:
      service:
        name: istiod
        namespace: istio-system
        path: "/inject"
      caBundle: ""
    sideEffects: None
    rules:
      - operations: [ "CREATE" ]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
    failurePolicy: Fail
    admissionReviewVersions: ["v1beta1", "v1"]
    namespaceSelector:
      matchLabels:
        istio-injection: enabled
准入控制器是一段代码，用于在对象持久化之前但请求已经过身份验证和授权之后，拦截对 Kubernetes API 服务器的请求。您可以定义两种类型的 Admission Webhook：Validating 和 Mutating。Validating 类型的 Webhook 可以根据自定义的准入策略决定是否拒绝请求；Mutating 类型的 Webhook 可以根据自定义配置来对请求进行编辑。
遥测
Istio服务网格最流行和最强大的功能之一就是其先进的可观察性。因为所有服务到服务的通信都是通过Envoy代理路由的，并且Istio的控制平面能够从这些代理收集日志和指标，所以服务网格可以为我们提供有关网络状态和服务行为的深刻见解。这为运营商提供了独特的故障排除，管理和优化服务方式，而不会给应用程序开发人员带来任何额外的负担。

早期负责遥测的组件mixer已经不再支持，取而代之的是telemetry v2，具体如下：



由于Istio Telemetry V2缺少可访问K8s元数据的中央组件（Mixer），因此代理服务器本身需要提供丰富指标所需的元数据。此外，必须将Enmixer提供的功能添加到Envoy中，以替代基于Mixer的遥测技术。 Istio Telemetry V2使用自定义的Envoy插件(WASM支持)来实现这一目标。

抛去和谷歌stackdriver无关的插件，主要是metadata_exchange和stats 。

metadata_exchange 会做reqeust和response的上下游标记，记录请求
stats 采集请求相关监控指标，暴露Prometheus 可采集的接口。
