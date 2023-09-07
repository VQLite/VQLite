package core

import (
	"encoding/json"
	"sync"
)

type SegmentMetadata struct {
	metadata       []*Metadata
	metadataRwLock sync.RWMutex
}

func NewSegmentMetadata() SegmentMetadata {
	return SegmentMetadata{
		metadata: make([]*Metadata, 0),
	}
}

func (sm *SegmentMetadata) Add(metadata *Metadata) {
	sm.metadataRwLock.Lock()
	defer sm.metadataRwLock.Unlock()
	sm.metadata = append(sm.metadata, metadata)

}

func (sm *SegmentMetadata) GetByid(id int) *Metadata {
	sm.metadataRwLock.RLock()
	defer sm.metadataRwLock.RUnlock()
	if id >= len(sm.metadata) {
		return nil
	}
	return sm.metadata[id]
}

func (sm *SegmentMetadata) DeleteByVqid(vqid string) bool {
	sm.metadataRwLock.Lock()
	defer sm.metadataRwLock.Unlock()

	for i := 0; i < len(sm.metadata); i++ {
		if sm.metadata[i] == nil {
			continue
		}
		if sm.metadata[i].Vqid == vqid {
			sm.metadata[i] = nil
			return true
		}
	}
	return false

}

func (sm *SegmentMetadata) Update(vqid string, metadata map[string]interface{}) int {
	sm.metadataRwLock.Lock()
	defer sm.metadataRwLock.Unlock()
	count := 0
	for i := 0; i < len(sm.metadata); i++ {
		if sm.metadata[i] == nil {
			continue
		}
		if sm.metadata[i].Vqid == vqid {
			serializedMetadata, _ := json.Marshal(metadata)
			sm.metadata[i].Data = serializedMetadata
			count += 1
		}
	}
	return count
}

func (sm *SegmentMetadata) GetByVqid(vqid string) *Metadata {
	sm.metadataRwLock.RLock()
	defer sm.metadataRwLock.RUnlock()
	for i := 0; i < len(sm.metadata); i++ {
		if sm.metadata[i] == nil {
			continue
		}
		if sm.metadata[i].Vqid == vqid {
			return sm.metadata[i]
		}
	}
	return nil
}

func (sm *SegmentMetadata) Size() int {
	sm.metadataRwLock.RLock()
	defer sm.metadataRwLock.RUnlock()
	return len(sm.metadata)
}
