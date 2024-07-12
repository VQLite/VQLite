package core

import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"vqlite/config"
	scann "vqlite/engine/go-scann"
	"vqlite/utils"
)

type Collection struct {
	Name              string
	Segments          []*Segment
	IndexType         string
	MaxSegmentId      uint64
	CollectionWorkDir string
	Dim               int
	lock              sync.RWMutex
}

// NewCollection creates a new Collection with the specified name and dimension.
//
// Parameters:
// - name: The name of the collection.
// - dim: The dimension of the collection.
//
// Returns:
// - *Collection: The newly created Collection object.
// - error: An error if the collection already exists or if the dimension is less than 0.
func NewCollection(name string, dim int) (*Collection, error) {
	dataPath := config.GlobalConfig.ServiceConfig.DataPath
	collectionPath := utils.Join(dataPath, name)
	// check if collection exists
	if _, ok := VqliteCollectionList.Get(name); ok {
		return nil, fmt.Errorf("collection is exist")
	}

	if dim < 0 {
		return nil, fmt.Errorf("NewCollection dim can not smaller than 0")
	}

	// create collection dir
	if !utils.IsDir(collectionPath) {
		utils.CreateDirPath(collectionPath)
	}

	col := &Collection{
		Name:              name,
		IndexType:         "ScaNN",
		MaxSegmentId:      0,
		Segments:          make([]*Segment, 0),
		CollectionWorkDir: collectionPath,
		Dim:               dim,
	}

	VqliteCollectionList.Add(col) // add to global collection map
	return col, nil
}

// AddNewSegment adds a new segment to the collection.
//
// This function does the following:
// - Locks the collection.
// - Creates a new segment in the collection's work directory.
// - Appends the new segment to the collection.
// - Increments the global segment ID.
//
// Returns the newly created segment or nil if there was an error.
func (c *Collection) AddNewSegment() *Segment {
	segmentWorkDir := utils.Join(c.CollectionWorkDir, fmt.Sprintf("segment_%d", c.MaxSegmentId))
	newSegment, err := NewSegment(c.MaxSegmentId, segmentWorkDir, c.Dim)
	if err != nil {
		log.Error().Err(err).Msg("create new segment error")

		return nil
	}
	c.Segments = append(c.Segments, newSegment)

	// global increment segment id
	atomic.AddUint64(&c.MaxSegmentId, 1)

	return newSegment
}

// GetInsertableSegment returns the insertable segment of the Collection.
//
// It checks if the Collection is empty and if so, it adds a new segment and returns it.
// If the last segment in the Collection has reached the maximum vector size, it seals the index
// and adds a new segment as well.
// Otherwise, it returns the last segment in the Collection.
//
// Returns a pointer to the insertable segment (*Segment).
func (c *Collection) GetInsertableSegment() *Segment {
	if c.Segments == nil || len(c.Segments) == 0 {
		return c.AddNewSegment()
	}
	lastSegment := c.Segments[len(c.Segments)-1]
	lastSegmentStat, err := lastSegment.Statistics()
	if err != nil {
		log.Error().Err(err).Msg("get segment statistics error")
		return nil
	}
	if lastSegmentStat.VectorCount >= config.GlobalConfig.ServiceConfig.SegmentVectorMaxSize {
		lastSegment.SealIndex()
		return c.AddNewSegment()
	}
	return lastSegment
}

// GetSealedSegments returns an array of sealed segments from the Collection.
//
// The function does not take any parameters.
// It returns a slice of pointers to Segment objects.
func (c *Collection) GetSealedSegments() []*Segment {
	segments := make([]*Segment, 0)
	for _, seg := range c.Segments {
		if seg.SegmentIndex.Sealed {
			segments = append(segments, seg)
		}
	}
	return segments
}

// GetSearchableSegments returns a slice of searchable segments from the collection.
//
// It iterates over the segments in the collection and appends the segments that are searchable
// to a new slice. The resulting slice of searchable segments is then returned.
//
// Returns:
// - A slice of *Segment containing the searchable segments.
func (c *Collection) GetSearchableSegments() []*Segment {
	searchableSegments := make([]*Segment, 0)
	for _, seg := range c.Segments {
		if seg.IsSearchable() {
			searchableSegments = append(searchableSegments, seg)
		}
	}
	return searchableSegments
}

// Search searches for query vectors in the collection.
//
// Parameters:
// - queryVecs: An array of query vectors to search for.
// - opt: The query options.
//
// Returns:
// - [][]SearchResult: A 2D array of search results, where each inner array represents the search results for a query vector.
// - error: An error if any occurred during the search.
func (c *Collection) Search(queryVecs []float32, opt QueryOpt) ([][]SearchResult, error) {
	searchableSegments := c.GetSearchableSegments()

	resultsCh := make(chan [][]scann.VidScore, len(searchableSegments))

	tempResults := make([][][]scann.VidScore, 0, len(searchableSegments))

	eg, ctx := errgroup.WithContext(context.Background())
	// get topK vectors from each segment
	// and merge them to one result

	for i, seg := range searchableSegments {
		_, seg := i, seg // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(opt.Timeout))
			defer cancel()
			searchResults, err := seg.Search(queryVecs, opt)
			if err != nil {
				return err
			}
			select {
			case <-timeoutCtx.Done():
				return fmt.Errorf("search index timeout")
			//case resultsCh <- searchResults.([][]scann.VidScore):
			case resultsCh <- searchResults:
				return nil
			}
		})
	}

	// wait for all search tasks done
	if err := eg.Wait(); err != nil {
		log.Warn().Err(err).Msg("search error")
		return nil, err
	}

	close(resultsCh)

	for result := range resultsCh {
		tempResults = append(tempResults, result)
	}

	if len(searchableSegments) < 1 || len(tempResults) == 0 || len(tempResults[0]) == 0 {
		return nil, fmt.Errorf("index current unavailable")
	}
	// merge search results from all segments
	vecScoreResults := make([][]scann.VidScore, len(queryVecs)/c.Dim)
	for i := 0; i < len(tempResults[0]); i++ {
		for j := 0; j < len(tempResults); j++ {
			vecScoreResults[i] = append(vecScoreResults[i], tempResults[j][i]...)
		}
		// sort every results by score desc
		sort.Slice(vecScoreResults[i], func(a, b int) bool {
			return vecScoreResults[i][a].Score > vecScoreResults[i][b].Score
		})

		// keep topK results
		if opt.TopK > 0 && opt.TopK < len(vecScoreResults[i]) {
			vecScoreResults[i] = vecScoreResults[i][:opt.TopK]
		}
	}
	// convert vecScoreResults to SearchResultItem
	results := make([][]SearchResult, len(queryVecs)/c.Dim)

	for i, vecScoreResult := range vecScoreResults {
		// get vqid from db
		for _, vecScore := range vecScoreResult {
			segmentId := vecScore.From
			seg := c.GetSegmentBySegmentId(segmentId)
			vectorId := vecScore.Vid
			docId, extra := utils.DecodeVectorId(vectorId)
			document := seg.SegmentMetadata.GetByid(int(docId))
			// document == nil means this doc is deleted
			if document == nil {
				continue
			}
			var resultMetadata map[string]interface{}
			err := json.Unmarshal(document.Data, &resultMetadata)
			if err != nil {
				continue
			}

			searchResultItem := &SearchResult{
				Vqid:     document.Vqid,
				Score:    vecScore.Score,
				Metadata: resultMetadata,
				Tag:      extra,
			}
			results[i] = append(results[i], *searchResultItem)
		}
	}
	return results, nil
}

func (c *Collection) GetSegmentBySegmentId(segmentId uint64) *Segment {
	// segment id  == len(segments) means the collection is complete
	// else means  only partial segmen,  need find the segment
	lastSegment := c.Segments[len(c.Segments)-1]
	if lastSegment.SegmentConfig.SegmentId == uint64(len(c.Segments)+1) {
		return c.Segments[segmentId]

	} else {
		// find the segment
		for _, s := range c.Segments {
			if s.SegmentConfig.SegmentId == segmentId {
				return s
			}
		}
		return nil
	}

}

// DeleteDocument deletes a document from the collection.
//
// It takes a string parameter vqid, which represents the unique identifier of the document to be deleted.
// It returns an integer representing the number of documents deleted.
func (c *Collection) DeleteDocument(vqid string) int {
	deletedCount := 0
	for _, seg := range c.Segments {
		deleted := seg.DeleteDocument(vqid)
		if deleted {
			deletedCount += 1
		}
	}
	return deletedCount
}

// UpdateDocumentMetadata updates the metadata of a document in the collection.
//
// It takes a pointer to an UpdateDocumentMetadataRequest struct as a parameter.
// The function returns an integer representing the number of documents whose metadata was updated.
func (c *Collection) UpdateDocumentMetadata(document *UpdateDocumentMetadataRequest) int {
	updatedCount := 0
	for _, seg := range c.Segments {
		segmentUpdatedCount := seg.UpdateDocumentMetadata(document)
		updatedCount += segmentUpdatedCount
	}
	return updatedCount
}

// GetDocumentMetadata retrieves the metadata of a document from the collection.
//
// Parameters:
//   - vqid: The unique identifier of the document.
//   - checkDuplicate: Indicates whether to check for duplicate documents.
//
// Returns:
//   - docMetadataList: A list of DocumentMetadataResult containing the metadata of the documents.
func (c *Collection) GetDocumentMetadata(vqid string, checkDuplicate bool) []DocumentMetadataResult {
	var docMetadataList []DocumentMetadataResult

	for _, seg := range c.Segments {
		segmentDocMetadataList := seg.GetDocumentMetadata(vqid, checkDuplicate)
		if len(segmentDocMetadataList) == 0 {
			continue
		}

		for _, docMetadata := range segmentDocMetadataList {
			var documentResult DocumentMetadataResult
			documentResult.Vqid = docMetadata.Vqid
			documentResult.SegmentId = seg.SegmentConfig.SegmentId
			_ = json.Unmarshal(docMetadata.Data, &documentResult.Data)
			docMetadataList = append(docMetadataList, documentResult)
		}

		if !checkDuplicate {
			break
		}
	}
	return docMetadataList
}

// AddDocument adds a document to the Collection.
//
// It checks if there is an insertable segment available. If not, it creates a new segment.
// Then, it adds the document to the segment.
func (c *Collection) AddDocument(document *AddDocumentRequest) {
	c.lock.Lock()
	defer c.lock.Unlock()
	seg := c.GetInsertableSegment()
	seg.AddDocument(document)
}

// BatchAddDocuments adds a batch of documents to the collection.
//
// It takes a pointer to a BatchAddDocumentsRequest struct as a parameter.
// There is no return value.
func (c *Collection) BatchAddDocuments(documents *BatchAddDocumentsRequest) {
	c.lock.Lock()
	defer c.lock.Unlock()
	seg := c.GetInsertableSegment()
	seg.BatchAddDocuments(documents)
}

// Statistics calculates and returns the statistics of the collection.
//
// It iterates over each segment in the collection and calculates the statistics
// for each segment. It then aggregates the segment statistics to calculate the
// statistics for the entire collection.
//
// Returns a pointer to a CollectionStatistics struct that contains the name of
// the collection, the list of segment statistics, the total number of segments,
// the total index size, and the total number of documents in the collection.
func (c *Collection) Statistics() *CollectionStatistics {

	collectionStatistics := &CollectionStatistics{
		CollectionName: c.Name,
		Segments:       make([]SegmentStatistics, 0),
		SegmentCount:   0,
		TotalIndexSize: 0,
		DocCount:       0,
	}
	for _, seg := range c.Segments {
		segmentStatistics, err := seg.Statistics()
		if err != nil {
			log.Error().Err(err).Msg("get segment statistics error")
		}
		collectionStatistics.Segments = append(collectionStatistics.Segments, *segmentStatistics)
		collectionStatistics.SegmentCount += 1
		collectionStatistics.TotalIndexSize += segmentStatistics.IndexStatistics.IndexSize
		collectionStatistics.VectorCount += uint64(segmentStatistics.VectorCount)
		collectionStatistics.DocCount += uint64(segmentStatistics.DocCount)
	}
	return collectionStatistics
}

// Train trains the Collection by iterating through its segments,
// checking if memory is enough, and training the index if necessary.
//
// Parameters:
// - numThreads: The number of threads to use for training.
// - ignoreCheck: Whether to ignore the memory check and train the index regardless.
//
// Return:
// - An error if there was an issue during training, otherwise nil.
func (c *Collection) Train(numThreads int, ignoreCheck bool) error {

	if utils.GetCpuCount() < numThreads {
		numThreads = 0
	}
	for _, seg := range c.Segments {
		// check if memory is enough
		segStatistics, err := seg.Statistics()
		if err != nil {
			return err
		}

		indexStatistics := &segStatistics.IndexStatistics
		// check if index need to be trained
		if indexStatistics.VidSize > 0 && indexStatistics.IndexSize < indexStatistics.VidSize {
			if !ignoreCheck {
				requiredMemorySize := uint64(float64(indexStatistics.VidSize*int64(indexStatistics.VecDim)*4) * 1.5) // estimated memory size
				availableMemory := utils.GetAvailableMemory()                                                        // available memory size
				if availableMemory < requiredMemorySize {                                                            // if available memory is not enough, skip training
					err := fmt.Errorf("no enough memory to train, require %d, availableMemory %d, segmentId %d", requiredMemorySize, availableMemory, seg.SegmentConfig.SegmentId)
					log.Error().Err(err).Msg("train error")
					return err
				}
			}
			err := seg.Train(numThreads) // train index
			if err != nil {
				return err
			}
			seg.SetHasNewIndex() // mark segment has new index
		}
	}
	return nil
}

func (c *Collection) DropIndex() error {
	for _, seg := range c.Segments {
		err := seg.DropIndex()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Collection) Drop() error {
	for _, seg := range c.Segments {
		err := seg.Drop()
		if err != nil {
			return err
		}
	}
	err := utils.DeleteDir(c.CollectionWorkDir)
	if err != nil {
		return err
	}
	VqliteCollectionList.Delete(c.Name)
	return nil
}

func (c *Collection) Dump() {
	for _, col := range c.Segments {
		col.Dump()
	}
}

func (c *Collection) DumpMetadata() {
	for _, col := range c.Segments {
		col.DumpMetadata()
	}
}

func (c *Collection) DumpIndex() {
	for _, col := range c.Segments {
		col.DumpIndex()
	}
}

func (c *Collection) LoadIndex() {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, seg := range c.Segments {
		if seg.HasNewIndex() {
			seg.LoadIndex()
		}
	}
}

func (c *Collection) Load() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.Segments != nil && len(c.Segments) > 0 {
		fmt.Println("collection segments is not nil", c.Segments)
		return
	}
	// load segments
	segmentsDirs, err := os.ReadDir(c.CollectionWorkDir)
	//remove useless files or dir
	utils.FilterValidSegmentDirs(segmentsDirs)
	// sort with numbers
	utils.SortFileNameAscend(segmentsDirs)

	if err != nil {
		log.Error().Err(err).Msg("load collection error")
		return
	}

	c.LoadSegments(segmentsDirs)

	if len(c.Segments) > 0 {
		c.MaxSegmentId = c.Segments[len(c.Segments)-1].SegmentConfig.SegmentId + 1
		c.Dim = c.Segments[0].SegmentConfig.Dim
	}

}

func (c *Collection) LoadSegments(segmentsDirs []os.DirEntry) {
	tempSegments := make([]*Segment, len(segmentsDirs))

	eg := &errgroup.Group{}
	//eg, ctx := errgroup.WithContext(context.Background())
	eg.SetLimit(4)
	for i, segmentDir := range segmentsDirs {
		i, segmentDir := i, segmentDir // https://golang.org/doc/faq#closures_and_goroutines

		eg.Go(func() error {

			segmentDirParts := strings.Split(segmentDir.Name(), "_")
			segmentId, err := strconv.ParseUint(segmentDirParts[1], 10, 64)
			if err != nil {
				return err
			}

			segmentWorkDir := utils.Join(c.CollectionWorkDir, segmentDir.Name())
			fmt.Println("load segment", segmentId, segmentWorkDir)
			seg, err := NewSegment(segmentId, segmentWorkDir, 0)
			seg.Load()
			if err == nil {
				tempSegments[i] = seg
			}
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		log.Error().Err(err).Msg("load collection error")
		return
	}
	c.Segments = tempSegments
}

func (c *Collection) CheckAndLoadNewIndexSegments() {

	c.lock.Lock()
	defer c.lock.Unlock()

	for _, seg := range c.Segments {
		if seg.HasNewIndex() {
			seg.LoadIndex()
		}
	}
}
