import json
from typing import List

import requests
from tenacity import *


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


class UpdateDocument:
    def __init__(self, vqid, metadata):
        self.vqid = vqid
        self.metadata = metadata

    @staticmethod
    def from_dict(obj):
        vqid = obj.get("vqid")
        metadata = obj.get("metadata")
        return UpdateDocument(vqid, metadata)

    def to_dict(self):
        result = {"vqid": self.vqid,
                  "metadata": self.metadata,
                  }
        return result


class Vqlite:
    def __init__(self, url):
        if url.endswith('/'):
            url = url[:-1]
        self.api_url = url

    @retry(stop=stop_after_attempt(3), reraise=True)
    def request(self, method, url, **kwargs):
        headers = kwargs.pop("headers", {})

        headers = {
            "User-Agent": "VQLite Python SDK",
            **headers,
        }
        response = requests.request(
            method, url, headers=headers, **kwargs
        )
        return response

    def stat(self):
        return self.statistics()

    def statistics(self):
        response = self.request('GET', self.api_url + '/statistics')
        return json.loads(response.text)

    def drop_collection(self, collection_name):
        response = self.request('DELETE', self.api_url + f'/collection/{collection_name}')
        return json.loads(response.text)

    def dump_collection(self, collection_name):
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/dump')
        return json.loads(response.text)

    def dump_collection_index(self, collection_name):
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/dump/index')
        return json.loads(response.text)

    def dump_collection_metadata(self, collection_name):
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/dump/metadata')
        return json.loads(response.text)

    def load_collection(self, collection_name):
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/load')
        return json.loads(response.text)

    def create_collection(self, collection_name, dim):
        payload = {
            'name': collection_name,
            'dim': dim
        }
        payload_json = json.dumps(payload)
        response = self.request('POST', self.api_url + F'/collection/{collection_name}', data=payload_json)
        return json.loads(response.text)

    def train_collection(self, collection_name, threads=0, ignore_check=False):
        payload = {
            'threads': threads,
            'ignore_check': ignore_check,
        }
        payload_json = json.dumps(payload)
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/train', data=payload_json)
        return json.loads(response.text)

    def search_collection(self, collection_name, query_vecs, topk=30, nprobe=128, reorder=128, timeout=60):
        payload = {
            'vectors': query_vecs,
            'opt': {
                'topk': topk,
                'nprobe': nprobe,
                'reorder': reorder,
                'timeout': timeout
            }
        }
        payload_json = json.dumps(payload)
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/search', data=payload_json)
        return json.loads(response.text)

    def add_document(self, collection_name, document: Document):
        payload = document.to_dict()
        payload_json = json.dumps(payload)
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/document', data=payload_json)
        return json.loads(response.text)

    def batch_add_documents(self, collection_name, documents: List[Document]):
        documents_dict = [document.to_dict() for document in documents]
        payload = {
            'documents': documents_dict
        }
        payload_json = json.dumps(payload)
        response = self.request('POST', self.api_url + f'/collection/{collection_name}/document/batch',
                                data=payload_json)
        return json.loads(response.text)

    def update_document(self, collection_name, update_document: UpdateDocument):
        document_dict = update_document.to_dict()
        payload_json = json.dumps(document_dict)
        response = self.request('PUT', self.api_url + f'/collection/{collection_name}/document', data=payload_json)
        return json.loads(response.text)

    def delete_document(self, collection_name, vqid):
        payload = {
            "vqid": vqid
        }
        payload_json = json.dumps(payload)
        response = self.request('DELETE', self.api_url + f'/collection/{collection_name}/document', data=payload_json)
        return json.loads(response.text)
