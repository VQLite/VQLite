# VQLite - Simple and Lightweight Vector Search Engine

[中文点这里](README_zh-CN.md)

VQLite is a lightweight and simple vector search engine. VQLite provides a simple RESTful API for building vector search services.

We developed this project because we felt that the existing vector retrieval engines on the market were too heavy and not suitable for our internal needs.

Commonly used vector retrieval engines (such as Milvus, Qdrant, Vearch) are managed by vector dimensions and operate on vectors. In contrast, VQLite processes data based on documents as dimensions. We found that in many cases, there is not a one-to-one relationship between documents and vectors but rather a one-to-many relationship where a document may have multiple vectors. Managing data based solely on vectors can be cumbersome and result in wasted resources when storing multiple metadata copies. Therefore, VQLite's design allows for multiple vectors per document while also allowing storage of metadata without relying on additional storage solutions such as MySQL or Redis.

Of course, the most important aspect is retrieval speed; therefore, we use Google's [ScaNN](https://github.com/google-research/google-research/tree/master/scann) at the core of our system since it may currently be the fastest public retrieval engine available.
~~ In fact, we simply encapsulate ScaNN ~~

The following image comes from ScaNN.

![bench](https://github.com/google-research/google-research/raw/master/scann/docs/glove_bench.png)

# Features

- [x] RESTful API
- [x] Simple and Lightweight
- [x] Fast

# Benchmark


We used 13 million vectors to build the database on AWS machines. Below are the test results.

| machine     | QPS  | params(nprobe, reorder, topK) |
|-------------|------|-------------------------------|
| c5.9xlarge  | 7600 | 256,256,30                    |
| c7g.8xlarge | 2900 | 256,256,30                    |
| r5.8xlarge  | 6900 | 256,256,30                    |

# Requirements

- Linux Or MacOS.
- Golang

# Get Started

We provide two ways of usage: compiling from source code and using Docker.

## Compile from source code

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
First, copy the vqlite.yaml to your machine and make appropriate modifications.

docker pull vqlite/vqlite

docker run --restart=always -d --name vqlite -p 8880:8880 \ 
    -v $(pwd)/vqlite.yaml:/app/vqlite.yaml \
    -v $(pwd)/vqlite_data:/app/vqlite_data \
    vqlite
```

## Example of use

You can check the python_sdk directory for sample code.

# Tips

Try to make each segment as large as possible, set it close to the memory limit, and minimize the number of segments. This can improve search speed.

The more segments there are, the slower the speed will be.