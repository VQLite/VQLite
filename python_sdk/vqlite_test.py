import numpy as np

import vqlite_api

api_url = 'http://127.0.0.1:8880/api'
collection_name = 'test'


def gen_vec_list(nb, seed=np.random.RandomState(1234)):
    xb = seed.rand(nb, 128).astype("float32")
    vec_list = xb.tolist()
    return vec_list


api = vqlite_api.Vqlite(url=api_url)

create_result = api.create_collection(collection_name, 128)
print('create_collection', create_result)

statistics_result = api.statistics()
print('statistics_result', statistics_result)

for i in range(10):
    res = api.add_document(collection_name, vqlite_api.Document(
        vqid=str(i),
        metadata={
            "title": str(i),
            "artist": str(i),
        },
        vectors=gen_vec_list(1000),
        vectors_tag=list(range(1000)),
    ))
    print("insert documents", i, res)

train_result = api.train_collection(collection_name)
print('train_result', train_result)

search_result = api.search_collection(collection_name, gen_vec_list(1))
print('search_result', search_result)
