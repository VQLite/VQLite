package core

import (
	"fmt"
	"os"
	"runtime"
	"vqlite/config"
	"vqlite/utils"
)

//func init() {
//	LoadCollections()
//}

func Statistics() *VQLiteStatistics {

	vqliteStatistics := &VQLiteStatistics{
		Collections:     make([]CollectionStatistics, 0),
		CollectionCount: 0,
		TotalIndexSize:  0,
		DocCount:        0,
	}
	for _, col := range VqliteCollectionList.Collections {
		collectionStatistics := col.Statistics()
		vqliteStatistics.Collections = append(vqliteStatistics.Collections, *collectionStatistics)
		vqliteStatistics.CollectionCount += 1
		vqliteStatistics.TotalIndexSize += collectionStatistics.TotalIndexSize
		vqliteStatistics.DocCount += collectionStatistics.DocCount
	}
	return vqliteStatistics
}

func SearchCollection(collectionName string, vecs [][]float32, opt QueryOpt) ([][]SearchResult, error) {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		err := CheckCollection(collectionName)
		if err != nil {
			return nil, err
		}
		go func() {
			err := LoadCollection(collectionName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "load collection [%s] failed, err: %s", collectionName, err.Error())
				return
			}
		}()
		return nil, fmt.Errorf("collection [%s] is loding", collectionName)
	}

	flattenedVectors := utils.FlattenFloat32Slice(vecs)
	CheckSearchOpt(&opt)
	return collection.Search(flattenedVectors, opt)
}

func CheckSearchOpt(opt *QueryOpt) {
	if opt.TopK == 0 {
		opt.TopK = 30
	}
	if opt.Timeout == 0 {
		opt.Timeout = 60
	}
	if opt.NProbe == 0 {
		opt.NProbe = 128
	}
	if opt.Reorder == 0 {
		opt.Reorder = 128
	}
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
	err := collection.Drop()
	if err != nil {
		return err
	}
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

func DeleteDocument(collectionName string, vqid string) (int, error) {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return 0, fmt.Errorf("collection [%s] not exists", collectionName)
	}
	if vqid == "" {
		return 0, fmt.Errorf("vqid is empty")
	}
	deletedCount := collection.DeleteDocument(vqid)
	return deletedCount, nil
}

func UpdateDocumentMetadata(collectionName string, doc *UpdateDocumentMetadataRequest) (int, error) {
	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return 0, fmt.Errorf("collection [%s] not exists", collectionName)
	}
	if doc.Vqid == "" {
		return 0, fmt.Errorf("vqid is empty")
	}
	updatedCount := collection.UpdateDocumentMetadata(doc)
	return updatedCount, nil
}

func GetDocumentMetadata(collectionName string, vqid string, checkDuplicate bool) ([]DocumentMetadataResult, error) {
	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return nil, fmt.Errorf("collection [%s] not exists", collectionName)
	}
	return collection.GetDocumentMetadata(vqid, checkDuplicate), nil
}

func DumpCollection(collectionName string) error {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	collection.Dump()
	return nil
}
func DumpCollectionMetadata(collectionName string) error {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	collection.DumpMetadata()
	return nil
}

func DumpCollectionIndex(collectionName string) error {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	collection.DumpIndex()
	return nil
}

func TrainCollection(collectionName string, numThreads int, ignoreCheck bool) error {

	collection, ok := VqliteCollectionList.Get(collectionName)
	if !ok {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	err := collection.Train(numThreads, ignoreCheck) // train index and dump to disk
	if err != nil {
		return err
	}
	runtime.GC() // force gc
	return nil
}

func CheckCollection(collectionName string) error {

	dataPath := config.GlobalConfig.ServiceConfig.DataPath
	collectionPath := utils.Join(dataPath, collectionName)
	if !utils.Exists(collectionPath) {
		return fmt.Errorf("collection [%s] not exists", collectionName)
	}
	return nil
}
func LoadCollection(collectionName string) error {
	col, ok := VqliteCollectionList.Get(collectionName)
	// if collection not exist, create new collection , else load last segment.
	fmt.Println("load collection", collectionName, ok)
	if !ok {
		newCol, err := NewCollection(collectionName, 0)
		fmt.Println("load collection NewCollection", collectionName, err)
		if err != nil {
			return err
		}
		newCol.Load()
	} else {
		col.CheckAndLoadNewIndexSegments()
	}
	return nil
}

func LoadAllCollections() {
	dataPath := config.GlobalConfig.ServiceConfig.DataPath
	collectionNames, err := os.ReadDir(dataPath)
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
