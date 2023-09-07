package core

import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	scann "vqlite/engine/go-scann"
	"vqlite/utils"
)

// SegmentConfig segment config
type SegmentConfig struct {
	SegmentId      uint64
	SegmentWorkDir string
	Dim            int
}

// SegmentIndex segment index
type SegmentIndex struct {
	VIndexC     *scann.ScaNNIndex
	Sealed      bool
	hasNewIndex atomic.Bool
	isTraining  atomic.Bool
}

type Segment struct {
	SegmentConfig   SegmentConfig
	SegmentIndex    SegmentIndex
	SegmentMetadata SegmentMetadata
}

const (
	TrainSuccess               = 0
	SegmentConfigFileNotExists = -1
	NewSegmentErr              = -2
	SegmentConfigFileLoadErr   = -3
	TrainSegmentErr            = -4
	DumpSegmentErr             = -5
)

//var RetMsg map[int]string = map[int]string{
//	TrainSuccess:               "TrainSuccess",
//	SegmentConfigFileNotExists: "SegmentConfigFileNotExists",
//	NewSegmentErr:              "NewSegmentErr",
//	SegmentConfigFileLoadErr:   "SegmentConfigFileLoadErr",
//	TrainSegmentErr:            "TrainSegmentErr",
//	DumpSegmentErr:             "DumpSegmentErr",
//}

func TrainSegmentByCmd(segmentWorkDir string, numThreads int) int {
	// new Segment
	fmt.Println("segmentWorkDir:", segmentWorkDir)
	fmt.Println("numThreads:", numThreads)
	segmentConfigSerializeFilename := utils.Join(segmentWorkDir, "config.gob")
	isExist := utils.Exists(segmentConfigSerializeFilename)
	if !isExist {
		log.Error().Msgf("segment config file not exist:%v", segmentConfigSerializeFilename)
		return SegmentConfigFileNotExists
	}

	seg, err := NewSegment(0, segmentWorkDir, 0)

	if err != nil {
		log.Error().Err(err).Msg("new segment error")
		return NewSegmentErr
	}

	err = utils.Load(&seg.SegmentConfig, segmentConfigSerializeFilename)
	if err != nil {
		log.Error().Err(err).Msg("load segment config error")
		return SegmentConfigFileLoadErr
	}
	seg.SegmentConfig.SegmentWorkDir = segmentWorkDir

	seg.LoadIndex()

	err = seg.SegmentIndex.VIndexC.Train(numThreads) // train index and dump index

	if err != nil {
		log.Error().Err(err).Msg("train segment index error")
		return TrainSegmentErr
	}
	err = seg.DumpIndex()
	if err != nil {
		log.Error().Err(err).Msg("dump segment index error")
		return DumpSegmentErr
	}

	log.Info().Msg("train segment success")
	return TrainSuccess

}

func NewSegment(segmentId uint64, segmentWorkDir string, dim int) (*Segment, error) {
	var vIndex *scann.ScaNNIndex
	var err error
	if dim > 0 {
		vIndex, err = scann.NewScaNNIndex(segmentWorkDir, dim, segmentId)
		if err != nil {
			log.Error().Err(err).Msg("create index error")
			return nil, err
		}
	}
	segment := &Segment{
		SegmentConfig: SegmentConfig{
			SegmentId:      segmentId,
			SegmentWorkDir: segmentWorkDir,
			Dim:            dim,
		},
		SegmentIndex: SegmentIndex{
			VIndexC:     vIndex,
			Sealed:      false,
			hasNewIndex: *atomic.NewBool(false),
		},
		SegmentMetadata: NewSegmentMetadata(),
	}
	log.Info().Msgf("NewSegment segmentId: %v, segmentWorkDir: %v, dim: %v", segmentId, segmentWorkDir, dim)
	return segment, nil
}

func (s *Segment) Search(queryVecs []float32, opt QueryOpt) ([][]scann.VidScore, error) {
	return s.SegmentIndex.VIndexC.Search(queryVecs, opt.TopK, opt.NProbe, opt.Reorder)
}

func (s *Segment) BatchAddDocuments(documents *BatchAddDocumentsRequest) {

	vectorsIds := make([]int64, 0)
	vectors := make([][]float32, 0)
	for _, document := range documents.Documents {
		documentId := int64(s.SegmentMetadata.Size())
		serializedMetadata, _ := json.Marshal(document.Metadata)

		s.SegmentMetadata.Add(&Metadata{
			Vqid: document.Vqid,
			Data: serializedMetadata,
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
	s.SegmentIndex.VIndexC.AddWithIDs(vectors, vectorsIds)

}

func (s *Segment) AddDocument(document *AddDocumentRequest) {

	// global increment doc id
	documentId := int64(s.SegmentMetadata.Size())

	serializedMetadata, _ := json.Marshal(document.Metadata)
	// add metadata
	s.SegmentMetadata.Add(&Metadata{
		Vqid: document.Vqid,
		Data: serializedMetadata,
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

	s.SegmentIndex.VIndexC.AddWithIDs(document.Vectors, vectorsIds)
}

func (s *Segment) DeleteDocument(vqid string) bool {
	return s.SegmentMetadata.DeleteByVqid(vqid)
}

func (s *Segment) UpdateDocumentMetadata(document *UpdateDocumentMetadataRequest) int {
	return s.SegmentMetadata.Update(document.Vqid, document.Metadata)
}

func (s *Segment) GetDocumentMetadata(vqid string, checkDuplicate bool) []*Metadata {
	var docMetadataList []*Metadata
	for i := 0; i < s.SegmentMetadata.Size(); i++ {
		if s.SegmentMetadata.GetByid(i) == nil {
			continue
		}
		if s.SegmentMetadata.GetByid(i).Vqid == vqid {
			docMetadataList = append(docMetadataList, s.SegmentMetadata.GetByid(i))
			if !checkDuplicate {
				break
			}
		}

	}

	return docMetadataList
}

func (s *Segment) SealIndex() {
	s.SegmentIndex.Sealed = true
}

func (s *Segment) IsSearchable() bool {

	if s.SegmentIndex.VIndexC == nil {
		return false
	}

	indexStatistics := s.SegmentIndex.VIndexC.Statistics()

	// INDEX > 0
	// Status in [IndexStateReady, IndexStateAdd, IndexStateDump]
	if indexStatistics.IndexSize > 0 {
		return utils.SliceContains(scann.SearchableStateSlice, indexStatistics.Status)
	}
	return false
}

func (s *Segment) Statistics() (*SegmentStatistics, error) {
	if s.SegmentIndex.VIndexC == nil {
		return nil, errors.New("index is nil")
	}
	indexStatistics := s.SegmentIndex.VIndexC.Statistics()
	vectorCount := indexStatistics.VidSize
	return &SegmentStatistics{
		SegmentId:       s.SegmentConfig.SegmentId,
		Sealed:          s.SegmentIndex.Sealed,
		Dim:             s.SegmentConfig.Dim,
		IndexStatistics: indexStatistics,
		VectorCount:     vectorCount,
		DocCount:        int64(s.SegmentMetadata.Size()),
	}, nil
}

func (s *Segment) Train(numThreads int) error {
	if !s.SegmentIndex.isTraining.CompareAndSwap(false, true) {
		return errors.New("segment is training")
	}

	s.DumpConfig()   // dump segment config
	s.DumpMetadata() // dump segment metadata and raw data

	cmd := &exec.Cmd{
		Path: "/proc/self/exe",
		Args: []string{os.Args[0] + "_train", "train", "-segmentWorkDir", s.SegmentConfig.SegmentWorkDir, "-numThreads", strconv.Itoa(numThreads)},
		SysProcAttr: &syscall.SysProcAttr{
			Pdeathsig: unix.SIGTERM,
		},
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("failed to start command")
	}
	if err := cmd.Wait(); err != nil {
		log.Error().Err(err).Msg("failed to wait command")

		exitError, ok := err.(*exec.ExitError)
		if !ok || exitError.ExitCode() != TrainSuccess {
			return err
		}
	}
	s.SegmentIndex.isTraining.Store(false)
	s.SetHasNewIndex()
	s.LoadIndex()
	return nil
}

func (s *Segment) DropIndex() error {
	s.SegmentIndex.VIndexC.Destroy()
	log.Info().Msgf("drop index success, segmentId:%v", s.SegmentConfig.SegmentId)
	return nil
}

func (s *Segment) Drop() error {
	// delete all metadata
	err := utils.DeleteDir(s.SegmentConfig.SegmentWorkDir)
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

func (s *Segment) DumpConfig() error {
	log.Info().Msgf("dump segment config, segmentId:%v", s.SegmentConfig.SegmentId)

	// create segment dir
	if !utils.IsDir(s.SegmentConfig.SegmentWorkDir) {
		utils.CreateDirPath(s.SegmentConfig.SegmentWorkDir)
	}

	segmentConfigSerializeFilename := utils.Join(s.SegmentConfig.SegmentWorkDir, "config.gob")
	err := utils.Dump(s.SegmentConfig, segmentConfigSerializeFilename)
	if err != nil {
		log.Error().Err(err).Msg("dump segment config error")
	}
	return err

}

func (s *Segment) DumpMetadata() error {
	log.Info().Msgf("dump segment metadata, segmentId:%v", s.SegmentConfig.SegmentId)

	// create segment dir
	if !utils.IsDir(s.SegmentConfig.SegmentWorkDir) {
		utils.CreateDirPath(s.SegmentConfig.SegmentWorkDir)
	}
	segmentMetadataSerializeFilename := utils.Join(s.SegmentConfig.SegmentWorkDir, "metadata.gob")
	err := utils.Dump(s.SegmentMetadata.metadata, segmentMetadataSerializeFilename)
	if err != nil {
		log.Error().Err(err).Msg("dump segment metadata error")
	}
	return err
}

func (s *Segment) DumpIndex() error {
	log.Info().Msgf("dump segment index, segmentId:%v", s.SegmentConfig.SegmentId)

	// create segment dir
	if !utils.IsDir(s.SegmentConfig.SegmentWorkDir) {
		utils.CreateDirPath(s.SegmentConfig.SegmentWorkDir)
	}
	err := s.SegmentIndex.VIndexC.Dump()
	return err
}

func (s *Segment) Dump() {
	var err error

	// dump segment metadata
	err = s.DumpMetadata()
	if err != nil {
		log.Error().Err(err).Msg("dump segment metadata error")
		return
	}
	log.Info().Msg("dump segment metadata success")

	// dump segment config
	err = s.DumpConfig()
	if err != nil {
		log.Error().Err(err).Msg("dump segment config error")
		return
	}
	log.Info().Msg("dump segment config success")

	// dump segment index
	//err = s.DumpIndex()
	//if err != nil {
	//	log.Error().Err(err).Msg("dump segment index error")
	//	return
	//}
	//log.Info().Msg("dump segment index success")
}

func (s *Segment) LoadConfig() {
	log.Info().Msgf("load segment config, segmentId:%v", s.SegmentConfig.SegmentId)

	segmentConfigSerializeFilename := utils.Join(s.SegmentConfig.SegmentWorkDir, "config.gob")
	isExist := utils.Exists(segmentConfigSerializeFilename)
	if !isExist {
		log.Error().Msgf("segment config file not exist:%v", segmentConfigSerializeFilename)
		return
	}
	err := utils.Load(&s.SegmentConfig, segmentConfigSerializeFilename)
	if err != nil {
		log.Error().Err(err).Msg("load segment config error")
	}
}

func (s *Segment) LoadMetadata() {
	log.Info().Msgf("load segment metadata, segmentId:%v", s.SegmentConfig.SegmentId)
	segmentMetadataSerializeFilename := utils.Join(s.SegmentConfig.SegmentWorkDir, "metadata.gob")
	if !utils.Exists(segmentMetadataSerializeFilename) {
		log.Error().Msgf("segment metadata file not exist:%v", segmentMetadataSerializeFilename)
		return
	}
	segmentWorkDirTemp := s.SegmentConfig.SegmentWorkDir

	err := utils.Load(&s.SegmentMetadata.metadata, segmentMetadataSerializeFilename)

	if err != nil {
		log.Error().Err(err).Msg("load segment metadata error")
	}
	// serialize will load SegmentWorkDir, but it may be not real dir, so we need to reset it
	s.SegmentConfig.SegmentWorkDir = segmentWorkDirTemp
}

func (s *Segment) LoadIndex() {
	if s.SegmentIndex.VIndexC == nil {
		log.Info().Msgf("load segment index, new index ,segmentId:%v", s.SegmentConfig.SegmentId)

		newIndex, err := scann.NewScaNNIndex(s.SegmentConfig.SegmentWorkDir, s.SegmentConfig.Dim, s.SegmentConfig.SegmentId)
		if err != nil {
			log.Error().Err(err).Msg("create index error")
		}
		s.SegmentIndex.VIndexC = newIndex

	} else if s.SegmentIndex.VIndexC != nil && s.HasNewIndex() {
		log.Info().Msgf("load segment index, replace index ,segmentId:%v", s.SegmentConfig.SegmentId)

		// load new index
		newIndex, err := scann.NewScaNNIndex(s.SegmentConfig.SegmentWorkDir, s.SegmentConfig.Dim, s.SegmentConfig.SegmentId)
		if err != nil {
			log.Error().Err(err).Msg("create index error")
		}
		// destroy replace old index and replace it
		oldIndexC := s.SegmentIndex.VIndexC
		s.SegmentIndex.VIndexC = newIndex
		oldIndexC.Destroy()
		s.SetNoNewIndex()
	}
}

func (s *Segment) Load() {
	s.LoadConfig()
	s.LoadIndex()
	s.LoadMetadata()
}

func (s *Segment) SetHasNewIndex() {
	s.SegmentIndex.hasNewIndex.Store(true)
}
func (s *Segment) SetNoNewIndex() {
	s.SegmentIndex.hasNewIndex.Store(false)
}
func (s *Segment) HasNewIndex() bool {
	return s.SegmentIndex.hasNewIndex.Load()
}
