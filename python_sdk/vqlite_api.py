import requests
from tenacity import *
import json
from typing import List


class Document:
    def __init__(self, vqid, metadata, vectors, vectors_tag=None):
        self.vqid = vqid
        self.metadata = metadata
        self.vectors = vectors
        self.vectors_tag = vectors_tag

    @staticmethod
    def from_dict(obj):
        vqid = obj.get("vqid")
        metadata = obj.get("metadata")
        vectors = obj.get("vectors")
        vectors_tag = obj.get("vectors_tag")
        return Document(vqid, metadata, vectors, vectors_tag)

    def to_dict(self):
        result = {"vqid": self.vqid,
                  "metadata": self.metadata,
                  "vectors": self.vectors,
                  "vectors_tag": self.vectors_tag
                  }
        return result


class Vqlite:
    def __init__(self, url):
        if url.endswith('/'):
            url = url[:-1]
        self.api_url = url
        self.session = requests.Session()

    @retry(stop=stop_after_attempt(3), reraise=True)
    def request(self, method, url, **kwargs):
        headers = kwargs.pop("headers", {})

        headers = {
            "Content-Type": kwargs.pop("content_type", "application/json"),
            "User-Agent": "VQLite Python SDK",
            **headers,
        }
        # response = self.session.request(
        #     method, url, headers=headers, **kwargs
        # )
        response = requests.request(
            method, url, headers=headers, **kwargs
        )
        return response

    def create_collection(self, collection_name, dim):
        payload = json.dumps({'name': collection_name, 'dim': dim})
        response = self.request('POST', self.api_url + F'/collection/{collection_name}',
                                data=payload)
        return json.loads(response.text)

    def train_collection(self, collection_name):
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/train')
        return json.loads(response.text)

    def dump_collection(self, collection_name):
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/dump')
        return json.loads(response.text)

    def search_collection(self, collection_name, query_vecs, topk=30, nprobe=128, reorder=128):
        payload = {
            'vectors': query_vecs,
            'opt': {
                'topk': topk,
                'nprobe': nprobe,
                'reorder': reorder
            }
        }
        payload = json.dumps(payload)
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/search', data=payload)
        return json.loads(response.text)

    def stat(self):
        response = self.request('GET', self.api_url + '/stat')
        return json.loads(response.text)

    def add_document(self, collection_name, document: Document):
        document_dict = document.to_dict()
        payload = json.dumps(document_dict)
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/document', data=payload)
        return json.loads(response.text)

    def drop_collection(self, collection_name):
        response = self.request('DELETE', self.api_url + f'/collection/{collection_name}')
        return json.loads(response.text)

    def batch_add_documents(self, collection_name, documents: List[Document]):
        documents_dict = [document.to_dict() for document in documents]
        payload = json.dumps({'documents': documents_dict})
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/document/batch', data=payload)
        return json.loads(response.text)

    def load_collection(self, collection_name):
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/load')
        return json.loads(response.text)
