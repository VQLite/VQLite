package core

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

func NewCollection(name string, dim int) (*Collection, error) {
	dataPath := config.GlobalConfig.ServiceConfig.DataPath
	collectionPath := utils.Join(dataPath, name)
	// check if collection exists
	if _, ok := VqliteCollectionList.Get(name); ok {
		return nil, fmt.Errorf("collection is exist")
	}

	// create collection dir
	if dim > 0 {
		if !utils.IsDir(collectionPath) {
			utils.CreateDirPath(collectionPath)
		} else {
			return nil, fmt.Errorf("collection dir is exist")
		}
	}

	col := &Collection{
		Name:              name,
		IndexType:         "scann",
		MaxSegmentId:      0,
		Segments:          make([]*Segment, 0),
		CollectionWorkDir: collectionPath,
		Dim:               dim,
	}

	if dim > 0 {
		// create default segment when create a new collection
		col.AddNewSegment()
	}
	VqliteCollectionList.Add(col) // add to global collection map

	return col, nil
}

func (c *Collection) AddNewSegment() *Segment {
	c.lock.Lock()
	defer c.lock.Unlock()

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

func (c *Collection) GetInsertableSegment() *Segment {
	lastSegment := c.Segments[len(c.Segments)-1]
	if lastSegment.VectorCounter >= config.GlobalConfig.ServiceConfig.SegmentVectorMaxSize {
		lastSegment.Seal()
		return c.AddNewSegment()
	}
	return lastSegment
}

func (c *Collection) GetSealedSegments() []*Segment {
	segments := make([]*Segment, 0)
	for _, seg := range c.Segments {
		if seg.Sealed {
			segments = append(segments, seg)
		}
	}
	return segments
}

func (c *Collection) GetSearchableSegments() []*Segment {
	searchableSegments := make([]*Segment, 0)
	for _, seg := range c.Segments {
		if seg.IsSearchable() {
			searchableSegments = append(searchableSegments, seg)
		}
	}
	return searchableSegments
}

func (c *Collection) Search(queryVecs []float32, opt QueryOpt) ([][]SearchResult, error) {
	searchableSegments := c.GetSearchableSegments()
	tempResults := make([][][]scann.VidScore, len(searchableSegments))

	eg := &errgroup.Group{}
	// get topK vectors from each segment
	// and merge them to one result
	for i, seg := range searchableSegments {
		i, seg := i, seg // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			searchResults, err := seg.Search(queryVecs, opt)
			if err == nil {
				tempResults[i] = searchResults
			}
			return nil
		})
	}
	// wait for all search tasks done
	if err := eg.Wait(); err != nil {
		log.Warn().Err(err).Msg("search error")
		if len(searchableSegments) < 1 || len(tempResults[0]) == 0 {
			return nil, err
		}
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
			seg := c.Segments[segmentId]
			vectorId := vecScore.Vid
			docId, extra := utils.DecodeVectorId(vectorId)
			document := seg.SegmentMetadata[docId]
			// document == nil means this doc is deleted
			if document == nil {
				continue
			}
			searchResultItem := &SearchResult{
				Vqid:     document.Vqid,
				Score:    vecScore.Score,
				Metadata: document.Data,
				Tag:      extra,
			}
			results[i] = append(results[i], *searchResultItem)
		}
	}
	return results, nil
}

func (c *Collection) DeleteDocument(vqid string) {
	for _, seg := range c.Segments {
		deleted := seg.DeleteDocument(vqid)
		if deleted {
			break
		}
	}
}

func (c *Collection) UpdateDocumentMetadata(document *UpdateDocumentMetadataRequest) {
	for _, seg := range c.Segments {
		updated := seg.UpdateDocumentMetadata(document)
		if updated {
			break
		}
	}
}

func (c *Collection) AddDocument(document *AddDocumentRequest) {
	seg := c.GetInsertableSegment()
	if seg == nil {
		seg = c.AddNewSegment()
	}
	seg.AddDocument(document)
}

func (c *Collection) BatchAddDocuments(documents *BatchAddDocumentsRequest) {
	seg := c.GetInsertableSegment()
	if seg == nil {
		seg = c.AddNewSegment()
	}
	seg.BatchAddDocuments(documents)
}

func (c *Collection) Stat() *CollectionStat {

	collectionStat := &CollectionStat{
		CollectionName: c.Name,
		Segments:       make([]SegmentStat, 0),
		SegmentCount:   0,
		TotalIndexSize: 0,
	}
	for _, seg := range c.Segments {
		segmentStat := seg.Stat()
		collectionStat.Segments = append(collectionStat.Segments, segmentStat)
		collectionStat.SegmentCount += 1
		collectionStat.TotalIndexSize += segmentStat.IndexStat.IndexSize
		collectionStat.VectorCount += segmentStat.VectorCount
	}
	return collectionStat
}

func (c *Collection) Train() error {
	for _, seg := range c.Segments {
		// check if memory is enough
		indexStat := seg.Stat().IndexStat
		requiredMemorySize := uint64(float64(indexStat.VidSize*int64(indexStat.VecDim)*4) * 1.5) // estimated memory size
		availableMemory := utils.GetAvailableMemory()                                            // available memory size
		if availableMemory < requiredMemorySize {                                                // if available memory is not enough, skip training
			err := fmt.Errorf("no enough memory to train, require %d, availableMemory %d, segmentId %d", requiredMemorySize, availableMemory, seg.SegmentId)
			log.Error().Err(err).Msg("train error")
			return err
		}

		// check if index need to be trained
		if indexStat.VidSize > 0 && indexStat.IndexSize < indexStat.VidSize {
			seg.Train()            // train index
			seg.HasNewIndex = true // mark index has new index
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

func (c *Collection) LoadIndex() {
	for _, seg := range c.Segments {
		if seg.HasNewIndex {
			seg.LoadIndex()
		}
	}

}
func (c *Collection) Load() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// load segments
	segmentsDirs, err := ioutil.ReadDir(c.CollectionWorkDir)

	//remove useless files or dir
	utils.FilterValidSegmentDirs(segmentsDirs)
	// sort with numbers
	utils.SortFileNameAscend(segmentsDirs)

	if err != nil {
		log.Error().Err(err).Msg("load collection error")
		return
	}

	for _, segmentDir := range segmentsDirs {
		segmentDirParts := strings.Split(segmentDir.Name(), "_")
		segmentId, err := strconv.ParseUint(segmentDirParts[1], 10, 64)

		if err != nil {
			log.Error().Err(err).Msg("load collection error")
			return
		}

		segmentWorkDir := utils.Join(c.CollectionWorkDir, segmentDir.Name())
		seg, err := NewSegment(segmentId, segmentWorkDir, 0)
		if err != nil {
			log.Error().Err(err).Msg("load collection error")
			return
		}

		seg.Load()

		c.Segments = append(c.Segments, seg)

		//atomic.AddUint64(&c.MaxSegmentId, 1)
	}

	if len(c.Segments) > 0 {
		c.MaxSegmentId = c.Segments[len(c.Segments)-1].SegmentId + 1
		c.Dim = c.Segments[0].Dim
	}

}

func (c *Collection) LoadLastSegment() {
	c.lock.Lock()
	defer c.lock.Unlock()
	lastSegmentId := c.MaxSegmentId - 1
	segmentWorkDir := utils.Join(c.CollectionWorkDir, fmt.Sprintf("segment_%d", lastSegmentId))
	seg, err := NewSegment(lastSegmentId, segmentWorkDir, 0)
	if err != nil {
		log.Error().Err(err).Msg("load segment error")
		return
	}
	seg.Load()
	c.Segments[lastSegmentId] = seg
}
