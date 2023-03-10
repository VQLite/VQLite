package core

import scann "vqlite/engine/go-scann"

type QueryOpt struct {
	TopK    int `json:"topk"`
	NProbe  int `json:"nprobe"`
	Reorder int `json:"reorder"`
}

type Metadata struct {
	Vqid string                 `json:"vqid"`
	Data map[string]interface{} `json:"data"`
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

// Stat

type SegmentStat struct {
	SegmentId   uint64          `json:"segment_id"`
	Sealed      bool            `json:"sealed"`
	Dim         int             `json:"dim"`
	IndexStat   scann.IndexStat `json:"index_stat"`
	VectorCount uint64          `json:"vector_count"`
}

type CollectionStat struct {
	CollectionName string        `json:"collection_name"`
	Segments       []SegmentStat `json:"segments"`
	SegmentCount   uint64        `json:"segment_count"`
	TotalIndexSize int64         `json:"total_index_size"`
	VectorCount    uint64        `json:"vector_count"`
}

type VQLiteStat struct {
	Collections     []CollectionStat `json:"collections"`
	CollectionCount uint64           `json:"collection_count"`
	TotalIndexSize  int64            `json:"total_index_size"`
}
