package core

import (
	"sync"
)

var VqliteCollectionList CollectionList

type CollectionList struct {
	Collections map[string]*Collection
	lock        sync.RWMutex
}

func init() {
	VqliteCollectionList.Collections = make(map[string]*Collection)
}

func (t *CollectionList) Add(collection *Collection) {
	t.lock.Lock()
	t.Collections[collection.Name] = collection
	t.lock.Unlock()
}

func (t *CollectionList) Get(name string) (*Collection, bool) {
	t.lock.RLock()
	collection, ok := t.Collections[name]
	t.lock.RUnlock()
	return collection, ok
}

func (t *CollectionList) Delete(name string) {
	t.lock.Lock()
	delete(t.Collections, name)
	t.lock.Unlock()
}

func (t *CollectionList) Len() int {
	t.lock.RLock()
	n := len(t.Collections)
	t.lock.RUnlock()
	return n
}

func (t *CollectionList) List() []*Collection {
	t.lock.RLock()
	indexes := make([]*Collection, 0, len(t.Collections))
	for _, index := range t.Collections {
		indexes = append(indexes, index)
	}
	t.lock.RUnlock()
	return indexes
}
