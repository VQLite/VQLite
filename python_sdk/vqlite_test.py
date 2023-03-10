import json
import time

import numpy as np
import vqlite_api

api_url = 'http://127.0.0.1:8880/api'
collection_name = 'test_collection'


def gen_vec_list(nb, seed=np.random.RandomState(1234)):
    xb = seed.rand(nb, 128).astype("float32")
    vec_list = xb.tolist()
    return vec_list


api = vqlite_api.Vqlite(url=api_url)
res = api.create_collection("test_collection", 128)
print('create_collection res', res)
stat = api.stat()
print('stat', stat)

for i in range(100):
    res = api.add_document(collection_name, vqlite_api.Document(
        vqid=str(i),
        metadata={
            "title": str(i),
            "artist": str(i),
        },
        vectors=gen_vec_list(1),
        vectors_tag=list(range(1)),

    ))
    print("insert documents", i, res)

train_result = api.train_collection(collection_name)
print('train_result', train_result)
dump_result = api.dump_collection("cover_song_oversea")
print('dump_result', dump_result)
search_result = api.search_collection(collection_name, gen_vec_list(1))
print('search_result', search_result)
