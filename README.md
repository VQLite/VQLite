# VQLite - Simple and Lightweight Vector Search Engine

[中文点这里](README_zh-CN.md)

VQLite is a lightweight and simple vector search engine. VQLite provides a simple RESTful API for building vector search services.


The reason we are creating this project is that there is currently no solution that meets our needs (as described in the following paragraph), and we feel that the vector search engines available on the market are too heavy, often requiring a Kubernetes cluster with multiple replicas. This can be a waste of resources and a burden for many projects.

Commonly used vector retrieval engines (such as Milvus, Qdrant, Vearch) are managed by vector dimensions and operate on vectors. In contrast, VQLite processes data based on documents as dimensions. We found that in many cases, there is not a one-to-one relationship between documents and vectors but rather a one-to-many relationship where a document may have multiple vectors. Managing data based solely on vectors can be cumbersome and result in wasted resources when storing multiple metadata copies. Therefore, VQLite's design allows for multiple vectors per document while also allowing storage of metadata without relying on additional storage solutions such as MySQL or Redis.

If you really want to use this project, understand the structure and design of VQLite, or if you want to make some modifications to VQLite to adapt it to your own needs, it is strongly recommended that you read  [Design.md](Design.md).

Of course, the most important aspect is retrieval speed; therefore, we use Google's [ScaNN](https://github.com/google-research/google-research/tree/master/scann) at the core of our system since it may currently be the fastest public retrieval engine available.
~~In fact, we simply encapsulate ScaNN~~

The following image comes from ScaNN.

![bench](https://github.com/google-research/google-research/raw/master/scann/docs/glove_bench.png)

# Features

- [x] RESTful API
- [x] Simple and Lightweight
- [x] Fast

# Benchmark


We used 13 million vectors to build the database on AWS machines. Below are the test results.

| Machine     | QPS  | Params(nprobe, reorder, topK) |
|-------------|------|-------------------------------|
| c5.9xlarge  | 7600 | 256,256,30                    |
| c7g.8xlarge | 2900 | 256,256,30                    |
| r5.8xlarge  | 6900 | 256,256,30                    |

# Requirements

- Linux Or MacOS.
- Golang

# Get Started

We provide two ways to use it, compiling from source code and using Docker. We recommend using the Docker.

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

First, copy the vqlite.yaml to your machine and make appropriate modifications.

```bash 

docker pull ghcr.io/vqlite/vqlite:latest

docker run --restart=always -d --name vqlite -p 8880:8880 \ 
    -v $(pwd)/vqlite.yaml:/app/vqlite.yaml \
    -v $(pwd)/vqlite_data:/app/vqlite_data \
    vqlite
```

## Example of use

You can check the python_sdk directory for sample code.

# Tips

- Try to make each segment as large as possible, set it close to the memory limit, and minimize the number of segments. This can improve search speed. The more segments there are, the slower the speed will be.
- According to our tests, the settings of nprobe and topK have little impact on retrieval speed. Reorder has a significant impact on both speed and recall rate. In simple terms, the larger the reorder value, the slower the retrieval speed but with higher recall rate.
- If you have already built the index, you can even delete all datasets.vql files to save disk space.