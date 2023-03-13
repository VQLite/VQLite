# Design

其实没啥设计，非常简单，一看就会，基本都按照目录分好了。


## Document

因为 VQLite 的操作的最小单元是 `Document`，所以我们先来看看 `Document` 是什么。

Document 由以下四个部分组成

|     字段      |         用途         |
|:-----------:|:------------------:|
|    vqid     | document 的 id， 不为空 |
|  metadata   |    存储的元数据，可以为空     |
|   vectors   |       向量，不为空       |
| vectors_tag |     向量的标记，可以为空     |

* vqid 需要用户自己提前定义好，这个就用来代表 document 的 id，不能为空。
* metadata 可以随便塞任何类型的数据（但是注意尽量保证插有用的信息，尽可能的少，因为这些数据都存在内存里），可以为空。
* vectors 是一个数组，即使只有一个 vector 也要写成数组的形式，不能为空。
* vectors_tag 与上面的 vector 对应，用来标记 vector, 这里的值是一个数组 如 [1,2,3]
  ，里面的值不要大于uint32的最大值（4294967295），可以为空，空的时候 VQLite 会按照向量的顺序给他们生成 tag。

下面举个 document 实际的 🌰 栗子。

假设我们有一首歌曲想入库：

```
{
    "vqid": "f8f78d25-23b0-4941-a7f4-7be66e6d8eea",
    "metadata": {"title":"Yellow", "artist":"coldplay"}
    "vectors": [[1,2,3], [4,5,6], [7,8,9]]
    "vectors_tag": [0, 60, 120]  
}
```

- vqid 为我们为这首歌生成的唯一 id, 用户自己生成；
- metadata 存放歌曲的歌名歌手信息；
- vectors 存放音频的 embedding；
- vectors_tag 则用来存放对应 vectors 的标记，比如这里就是第一个 vector 是第 0 秒的内容 第二个 vector 就是第 60 秒的内容,第三个是
  120 秒的内容。这里如果你不需要用你不写也没事，会自动生成序列号。

# 结构

基本上所有的操作都是一层一层由外向内调用的，顺序基本上都是 `API` -> `CollectionList` -> `Collection` -> `Segment`。

## 文件结构

理解数据存放的文件结构之后就知道 VQLite 的设计了。

```
vqlite_data/
`-- collection
    `-- segment_0
        |-- datasets.vql
        |-- index/
        |-- metadata.gob
        `-- vids.vql
```

这里我们借鉴~~抄袭~~了 Milvus 的做法，引入了 collection 的概念，你可以把它理解为数据库中的表的概念。

- vqlite_data 是一个文件夹，里面存储了所有的 collection
- collection 是一个文件夹，里面存储了所有的 segment
- Segment_{序号} 是一个文件夹，里面存储了所有的数据
- metadata.gob 是一个文件， 是 golang 的对象，存着当前 segment 的基础信息，也包括 向量对应的 metadata
- datasets.vql 是一个文件，存储了所有的向量
- vids.vql 是一个文件，存储了所有的向量对应的 vqid
- index 是一个文件夹，里面是 ScaNN 的索引文件


## Segment

Segment 是 VQLite 的最小存储单元，一个 Segment 里面存储了一定数量的 Document，这个数量是可以配置的。

## Collection

Collection 管理 Segment, 一个 Collection 可以有很多个 Segment。
Collection 接受到的所有操作都会转发给其管理的所有的 Segment。


## CollectionList

CollectionList 管理 Collection. VQLite 只有一个 CollectionList，它管理着所有的 Collection。

每次接收到操作的时候，都会先去 CollectionList 中找到对应的 Collection，再去进行对应的操作。

