---
titile: Jekyll Minimal-Mistakes 搭建博客
excerpt: jekyll 搭建博客可以很方便的发布在GitHub Pages上面，但是由于 GitHub 官方支持的主题插件比较少，我使用了一个非官方的主题插件——Jekyll Minimal Mistake。
sticky: true
---

## 前言

jekyll 搭建博客可以很方便的发布在GitHub Pages上面，但是由于 GitHub 官方支持的主题插件比较少，我使用了一个非官方的主题插件——Jekyll Minimal Mistakes。这个主题还挺好的用的，但是由于缺少一些中文文档，这里把搭建过程整理下来方便，以后有需求的人复用，也方便我万一断更了更容易把这项技能捡起来。

项目地址：https://github.com/mmistakes/minimal-mistakes

## 网站部署

## 发布文章

在 _posts 目录下添加一个文件：
```shell
vim _posts/2022-11-15-test.md

---
layout: single
title:  "test"
date:   2022-11-15 12:00
author_profile: false
---

# Header
## Header 2
### Header 3
#### Header 4
#### Header 5
```

## 发布专题

编辑 navigation.yml 文件
```shell
vim ./_data/navigation.yml
# main links
main:
  - title: "关于"
    url: /about/
  - title: "Ceph专题"
    url: /ceph/
  - title: "Kubernetes专题"
    url: /kubernetes/
  - title: "Interview专题"
    url: /interview/
+ - title: "test"
+   url: /test/
```

在 page 目录下面创建一个 test.md的文件，添加如下内容：
```shell
vim ./_pages/test.md
# test collection
---
layout: collection
title: "test"
permalink: /test/
collection: test
author_profile: false
---
```

编辑 _config.yml 增加一条 test collection
```
vim _config.yml

# ...
# Collections
collections:
  ceph:
    output: true
    permalink: /:collection/:path/
+ test:
+   output: true
+   permalink: /:collection/:path/

# ..

# Defaults
defaults:
  # _posts
  - scope:
      path: ""
      type: posts
    values:
      layout: single
      author_profile: true
      read_time: true
      comments: # true
      share: false
      related: true
  # _ceph
  - scope:
      path: ""
      type: ceph
    values:
      layout: single
      author_profile: true
      share: false
      related: true
+ # _test
+ - sope:
+     pah: ""
+    type: test
+   vaues:
+     layout: sngle
+     authr_profile: true
+     share false
+     related: true
```

## 百度统计



## 更多技术分享浏览我的博客：  
https://thierryzhou.github.io