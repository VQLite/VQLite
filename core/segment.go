package core

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"sync"
	"sync/atomic"
	scann "vqlite/engine/go-scann"
	"vqlite/utils"
)

type Segment struct {
	SegmentId       uint64
	Sealed          bool
	SegmentWorkDir  string
	Dim             int
	SegmentMetadata []*Metadata
	VIndexC         *scann.ScaNNIndex
	rwLock          sync.RWMutex
	HasNewIndex     bool
	VectorCounter   uint64
}

func NewSegment(segmentId uint64, segmentWorkDir string, dim int) (*Segment, error) {
	fmt.Println("NewSegment", segmentId, segmentWorkDir, dim)
	var vIndex *scann.ScaNNIndex
	var err error
	if dim > 0 {
		vIndex, err = scann.NewScaNNIndex(segmentWorkDir, dim, segmentId)
		if err != nil {
			log.Error().Err(err).Msg("create index error")
			return nil, err
		}
	}
	newSegment := &Segment{
		SegmentId:      segmentId,
		Sealed:         false,
		VIndexC:        vIndex,
		SegmentWorkDir: segmentWorkDir,
		Dim:            dim,
		HasNewIndex:    false,
	}
	return newSegment, nil
}

func (s *Segment) Search(queryVecs []float32, opt QueryOpt) ([][]scann.VidScore, error) {
	return s.VIndexC.Search(queryVecs, opt.TopK, opt.NProbe, opt.Reorder)
}

func (s *Segment) BatchAddDocuments(documents *BatchAddDocumentsRequest) {

	s.rwLock.Lock()
	defer s.rwLock.Unlock()

	vectorsIds := make([]int64, 0)
	vectors := make([][]float32, 0)

	for _, document := range documents.Documents {
		documentId := int64(len(s.SegmentMetadata))
		s.SegmentMetadata = append(s.SegmentMetadata, &Metadata{
			Vqid: document.Vqid,
			Data: document.Metadata,
		})
		if len(document.VectorsTag) == 0 {
			for i := 0; i < len(document.Vectors); i++ {
				vectorId, err := utils.EncodeVectorId(documentId, int64(i))
				if err != nil {
					log.Error().Err(err).Msgf("encode vector id error docId:%v, tag:%v", documentId, i)
					continue
				}
				vectorsIds = append(vectorsIds, vectorId)
				vectors = append(vectors, document.Vectors[i])
			}
		} else {
			for i, tag := range document.VectorsTag {
				vectorId, err := utils.EncodeVectorId(documentId, tag)
				if err != nil {
					log.Error().Err(err).Msgf("encode vector id error docId:%v, tag:%v", documentId, tag)
					continue
				}
				vectorsIds = append(vectorsIds, vectorId)
				vectors = append(vectors, document.Vectors[i])
			}
		}

	}
	atomic.AddUint64(&s.VectorCounter, uint64(len(vectors)))
	s.VIndexC.AddWithIDs(vectors, vectorsIds)

}

func (s *Segment) AddDocument(document *AddDocumentRequest) {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	// global increment doc id
	documentId := int64(len(s.SegmentMetadata))
	// add metadata
	s.SegmentMetadata = append(s.SegmentMetadata, &Metadata{
		Vqid: document.Vqid,
		Data: document.Metadata,
	})

	count := len(document.Vectors)
	vectorsIds := make([]int64, 0)
	// when vectors tag is empty, use count as tag
	if len(document.VectorsTag) == 0 {
		for i := 0; i < count; i++ {
			vectorId, err := utils.EncodeVectorId(documentId, int64(i))
			if err != nil {
				log.Error().Err(err).Msgf("encode vector id error docId:%v, tag:%v", documentId, i)
				continue
			}
			vectorsIds = append(vectorsIds, vectorId)
		}
	} else {
		for _, tag := range document.VectorsTag {
			vectorId, err := utils.EncodeVectorId(documentId, tag)
			if err != nil {
				log.Error().Err(err).Msgf("encode vector id error docId:%v, tag:%v", documentId, tag)
				continue
			}
			vectorsIds = append(vectorsIds, vectorId)
		}
	}
	atomic.AddUint64(&s.VectorCounter, uint64(count))
	s.VIndexC.AddWithIDs(document.Vectors, vectorsIds)
}

func (s *Segment) DeleteDocument(vqid string) bool {
	for i := 0; i < len(s.SegmentMetadata); i++ {
		if s.SegmentMetadata[i].Vqid == vqid {
			s.SegmentMetadata[i] = nil
			return true
		}
	}
	return false
}

func (s *Segment) UpdateDocumentMetadata(document *UpdateDocumentMetadataRequest) bool {
	for i := 0; i < len(s.SegmentMetadata); i++ {
		if s.SegmentMetadata[i].Vqid == document.Vqid {
			s.SegmentMetadata[i].Data = document.Metadata
			return true
		}
	}
	return false
}

func (s *Segment) Seal() {
	s.Sealed = true
}

func (s *Segment) IsSearchable() bool {
	indexStat := s.VIndexC.Stat()
	// INDEX >0
	// Status in [IndexStateReady, IndexStateAdd, IndexStateDump]
	if indexStat.IndexSize > 0 {
		return utils.SliceContains(scann.SearchableStatusSlice, indexStat.Status)
	}
	return false
}

func (s *Segment) Stat() SegmentStat {
	indexStat := s.VIndexC.Stat()
	return SegmentStat{
		SegmentId:   s.SegmentId,
		Sealed:      s.Sealed,
		Dim:         s.Dim,
		IndexStat:   indexStat,
		VectorCount: s.VectorCounter,
	}
}

func (s *Segment) Train() {
	s.DumpMetadata() // dump segment metadata and raw data
	s.VIndexC.Train()
	s.DumpIndex() // dump index
}

func (s *Segment) DropIndex() error {
	err := s.VIndexC.Destroy()
	if err != nil {
		log.Error().Err(err).Msg("drop index error")
		return err
	}
	return nil
}

func (s *Segment) Drop() error {
	// delete all metadata
	err := utils.DeleteDir(s.SegmentWorkDir)
	if err != nil {
		return err
	}
	// delete index
	err = s.DropIndex()
	if err != nil {
		return err
	}
	return nil
}

func (s *Segment) DumpMetadata() error {
	// create segment dir
	if !utils.IsDir(s.SegmentWorkDir) {
		utils.CreateDirPath(s.SegmentWorkDir)
	}
	segmentSerializeFilename := utils.Join(s.SegmentWorkDir, "metadata.gob")
	err := utils.Dump(s, segmentSerializeFilename)
	if err != nil {
		log.Error().Err(err).Msg("dump segment metadata error")
	}
	return err
}
func (s *Segment) DumpIndex() error {
	// create segment dir
	if !utils.IsDir(s.SegmentWorkDir) {
		utils.CreateDirPath(s.SegmentWorkDir)
	}
	err := s.VIndexC.Dump()
	return err
}

func (s *Segment) Dump() {
	err := s.DumpMetadata()
	if err != nil {
		return
	}
	log.Info().Msg("dump segment metadata success")
	s.DumpIndex()
}

func (s *Segment) LoadIndex() {
	s.VIndexC = nil
	var vIndex *scann.ScaNNIndex
	vIndex, err := scann.NewScaNNIndex(s.SegmentWorkDir, s.Dim, s.SegmentId)
	if err != nil {
		log.Error().Err(err).Msg("create index error")
	}
	s.VIndexC = vIndex
	if err != nil {
		log.Error().Err(err).Msg("load segment error")
	}
}
func (s *Segment) LoadMetadata() {
	segmentSerializeFilename := utils.Join(s.SegmentWorkDir, "metadata.gob")
	isExist := utils.Exists(segmentSerializeFilename)
	segmentWorkDirTemp := s.SegmentWorkDir
	if !isExist {
		log.Error().Msgf("segment metadata file not exist:%v", segmentSerializeFilename)
		return
	}
	err := utils.Load(s, segmentSerializeFilename)

	if err != nil {
		log.Error().Err(err).Msg("load segment error")
	}
	// serialize will load SegmentWorkDir, but it may be not real dir, so we need to reset it
	s.SegmentWorkDir = segmentWorkDirTemp
}

func (s *Segment) Load() {
	s.LoadMetadata()
	s.LoadIndex()
}
