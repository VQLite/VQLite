# VQLite - Simple and Lightweight Vector Search Engine

VQLite 是一个轻量简单的向量搜索引擎。VQLite 提供了一套简单的 RESTful API 用来构建向量搜索服务。

我们写这个项目的原因是我们觉得目前市面上的向量检索引擎都太重了，而且都不太适合我们内部的需求。

目前常见的向量检索引擎（比如 Milvus, Qdrant, Vearch），都是以 vector 维度来管理的，操作的对象都是 vector。与之不同的是，VQLite
在处理数据上是按照 document 为维度处理的。因为在我们的使用中发现很多时候 document 和 vector 不是一对一的关系而是一对多，
也就是说一个 document 可能会有很多的向量，如果按照 vector 来管理数据的话就会很麻烦，而且在存储多份 metadata 上也会造成资源的浪费。所以
VQLite 的设计是一个 document 可以有很多的 vector，同时 document 还可以存储 metadata，基础信息都可以存进来不需要借助另外的存储（mysql
redis 之类的）。

当然，最重要的是检索速度，所以我们的底层用了 Google
的 [ScaNN](https://github.com/google-research/google-research/tree/master/scann)  ，因为它截止到目前为止可能是公开的**最快的
**检索引擎。~~其实我们就是对 ScaNN 的一个的封装~~

下图来自 ScaNN

![bench](https://github.com/google-research/google-research/raw/master/scann/docs/glove_bench.png)

# Features

- [x] RESTful API
- [x] Simple and Lightweight
- [x] Fast

# Benchmark

我们使用 1300 万的向量在 AWS 的机器上建库，下面是测试的结果

| machine     | QPS  | params(nprobe, reorder, topK) |
|-------------|------|-------------------------------|
| c5.9xlarge  | 7600 | 256,256,30                    |
| c7g.8xlarge | 2900 | 256,256,30                    |
| r5.8xlarge  | 6900 | 256,256,30                    |

# Requirements

- Linux Or MacOS.
- Golang

# Get Started

我们提供两种使用方式从源码编译和Docker

## 从源码编译

```bash
git clone --recurse-submodules https://github.com/VQLite/VQLite.git
# git submodule update --remote --merge
cd vqindex
bash build.sh vqindex_api
cp bazel-bin/scann/scann_ops/cc/libvqlite_api.so /usr/local/lib/
cp bazel-bin//external/local_config_tf/libtensorflow_framework.* /usr/local/lib/
cd ..
go build cmd/vqlite.go
```

## Docker

```bash 
首先复制一下 vqlite.yaml 到你的机器上并做合适的修改

docker pull vqlite/vqlite

docker run --restart=always -d --name vqlite -p 8880:8880 \ 
    -v $(pwd)/vqlite.yaml:/app/vqlite.yaml \
    -v $(pwd)/vqlite_data:/app/vqlite_data \
    vqlite
```

## 调用示例

可以查看 python_sdk 目录，有示例代码。

# Tips

尽量让一个 Segment 尽可能的大，贴着内存的上限设置，尽量减少 Segment 的数量，这样可以提高检索速度。
Segment 的数量越多，速度就会越慢。