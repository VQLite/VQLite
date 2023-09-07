package core

import scann "vqlite/engine/go-scann"

type QueryOpt struct {
	TopK    int `json:"topk"`
	NProbe  int `json:"nprobe"`
	Reorder int `json:"reorder"`
	Timeout int `json:"timeout"`
}

type Metadata struct {
	Vqid string `json:"vqid"`
	Data []byte
}

// Http Request and Response struct

type CreateCollectionRequest struct {
	Name string `json:"name"`
	Dim  int    `json:"dim"`
}

type BatchAddDocumentsRequest struct {
	Documents []AddDocumentRequest `json:"documents"`
}
type AddDocumentRequest struct {
	Vqid       string                 `json:"vqid"`
	Metadata   map[string]interface{} `json:"metadata"`
	Vectors    [][]float32            `json:"vectors"`
	VectorsTag []int64                `json:"vectors_tag"`
}
type DeleteDocumentRequest struct {
	Vqid string `json:"vqid"`
}

type UpdateDocumentMetadataRequest struct {
	Vqid     string                 `json:"vqid"`
	Metadata map[string]interface{} `json:"metadata"`
}

type TrainRequest struct {
	Threads     int  `json:"threads"`
	IgnoreCheck bool `json:"ignore_check"`
}

type SearchRequest struct {
	Vectors [][]float32 `json:"vectors"`
	Opt     QueryOpt    `json:"opt"`
}

type SearchResult struct {
	Vqid     string                 `json:"vqid"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
	Tag      int64                  `json:"tag"`
}

type DocumentMetadataResult struct {
	Vqid      string      `json:"vqid"`
	Data      interface{} `json:"data"`
	SegmentId uint64      `json:"segment_id"`
}

// Statistics

type SegmentStatistics struct {
	SegmentId       uint64                `json:"segment_id"`
	Sealed          bool                  `json:"sealed"`
	Dim             int                   `json:"dim"`
	IndexStatistics scann.IndexStatistics `json:"index_statistics"`
	VectorCount     int64                 `json:"vector_count"`
	DocCount        int64                 `json:"doc_count"`
}

type CollectionStatistics struct {
	CollectionName string              `json:"collection_name"`
	Segments       []SegmentStatistics `json:"segments"`
	SegmentCount   uint64              `json:"segment_count"`
	TotalIndexSize int64               `json:"total_index_size"`
	VectorCount    uint64              `json:"vector_count"`
	DocCount       uint64              `json:"doc_count"`
}

type VQLiteStatistics struct {
	Collections     []CollectionStatistics `json:"collections"`
	CollectionCount uint64                 `json:"collection_count"`
	TotalIndexSize  int64                  `json:"total_index_size"`
	DocCount        uint64                 `json:"total_doc_count"`
}
