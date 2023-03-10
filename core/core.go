package core

import (
	"fmt"
	"os"
	"runtime"
	"vqlite/config"
	"vqlite/utils"
)

func init() {
	LoadCollections()
}

func Stat() *VQLiteStat {

	vqliteStat := &VQLiteStat{
		Collections:     make([]CollectionStat, 0),
		CollectionCount: 0,
		TotalIndexSize:  0,
	}
	for _, col := range VqliteCollectionList.Collections {
		collectionStat := col.Stat()
		vqliteStat.Collections = append(vqliteStat.Collections, *collectionStat)
		vqliteStat.CollectionCount += 1
		vqliteStat.TotalIndexSize += collectionStat.TotalIndexSize
	}
	return vqliteStat
}

func SearchCollection(collectionName string, vecs [][]float32, opt QueryOpt) ([][]SearchResult, error) {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return nil, fmt.Errorf("collection [%s] not exists", collectionName)
	}

	flattenedVectors := utils.FlattenFloat32Slice(vecs)
	return collection.Search(flattenedVectors, opt)
}

func CreateCollection(collectionName string, dim int) (*Collection, error) {
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is empty")
	}
	if dim <= 0 {
		return nil, fmt.Errorf("dim must be greater than 0")
	}
	_, ok := VqliteCollectionList.Get(collectionName)
	if ok {
		return nil, fmt.Errorf("collection [%s] already exists", collectionName)
	}
	col, err := NewCollection(collectionName, dim)
	if err != nil {
		return nil, err
	}
	return col, nil
}

func DropCollection(collectionName string) error {
	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	collection.Drop()
	return nil

}

func AddDocument(collectionName string, doc *AddDocumentRequest) error {
	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	if doc.Vqid == "" {
		return fmt.Errorf("vqid is empty")
	}
	if len(doc.Vectors) == 0 {
		return fmt.Errorf("vectors is empty")
	}
	collection.AddDocument(doc)
	return nil
}

func BatchAddDocuments(collectionName string, documents *BatchAddDocumentsRequest) error {
	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	if len(documents.Documents) == 0 {
		return fmt.Errorf("documents is empty")
	}

	collection.BatchAddDocuments(documents)
	return nil
}

func DeleteDocument(collectionName string, vqid string) error {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	if vqid == "" {
		return fmt.Errorf("vqid is empty")
	}
	collection.DeleteDocument(vqid)
	return nil
}
func UpdateDocumentMetadata(collectionName string, doc *UpdateDocumentMetadataRequest) error {
	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	if doc.Vqid == "" {
		return fmt.Errorf("vqid is empty")
	}
	collection.UpdateDocumentMetadata(doc)
	return nil
}

func DumpCollection(collectionName string) error {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	collection.Dump()
	return nil
}

func TrainCollection(collectionName string) error {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	err := collection.Train() // train index and dump to disk
	if err != nil {
		return err
	}
	runtime.GC() // force gc
	return nil
}

func LoadCollection(collectionName string) error {
	col, ok := VqliteCollectionList.Get(collectionName)
	// if collection not exist, create new collection , else load last segment.
	if !ok {
		newCol, err := NewCollection(collectionName, 0)
		if err != nil {
			return err
		}
		newCol.Load()
	} else {
		col.LoadLastSegment()
	}
	return nil
}

func LoadCollections() {

	dataPath := config.GlobalConfig.ServiceConfig.DataPath
	collectionNames, err := os.ReadDir(dataPath)
	for _, collectionName := range collectionNames {
		fmt.Println(collectionName.Name())
	}
	if err != nil {
		return
	}
	for _, collectionName := range collectionNames {
		if collectionName.IsDir() {
			col, err := NewCollection(collectionName.Name(), 0)
			if err != nil {
				continue
			}
			col.Load()
		}
	}
}
