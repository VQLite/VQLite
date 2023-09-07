package go_scann

// #cgo CXXFLAGS: -std=c++17 -I. -Igo-scann
// #cgo LDFLAGS: -L. -ltensorflow_framework -lvqindex_api -lpthread
// #include "vqindex_api.h"
// #include <stdlib.h>
// #include <stdio.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"github.com/rs/zerolog/log"
	"sync"
	"unsafe"
	"vqlite/utils"
	"vqlite/utils/conc"
)

type IndexStatistics struct {
	DatasetSize    int64  `json:"dataset_size"`
	VidSize        int64  `json:"vid_size"`
	IndexSize      int64  `json:"index_size"`
	Nlist          int32  `json:"nlist"`
	VecDim         int32  `json:"vec_dim"`
	BruteThreshold int64  `json:"brute_threshold"`
	IsBrute        bool   `json:"is_brute"`
	Status         string `json:"status"`
}

type VidScore struct {
	Vid   int64
	Score float32
	From  uint64
}

type ScaNNIndex struct {
	vdbC         unsafe.Pointer
	Dim          int
	IndexWorkDir string
	IndexId      uint64
	vdbCRwLock   sync.RWMutex
}

const (
	IndexStateNone    = "INDEX_STATE_NONE"
	IndexStateNoInit  = "INDEX_STATE_NOINIT"
	IndexStateNoIndex = "INDEX_STATE_NOINDEX"
	IndexStateReady   = "INDEX_STATE_READY"
	IndexStateAdd     = "INDEX_STATE_ADD"
	IndexStateTrain   = "INDEX_STATE_TRAIN"
	IndexStateDump    = "INDEX_STATE_DUMP"
	IndexStateUnknown = "INDEX_STATE_UNKNOWN"
)

var SearchableStateSlice = []string{IndexStateReady, IndexStateAdd, IndexStateDump}

var CIndexStateMap = map[C.index_state_t]string{
	C.index_state_t(C.INDEX_STATE_NONE):    IndexStateNone,
	C.index_state_t(C.INDEX_STATE_NOINIT):  IndexStateNoInit,
	C.index_state_t(C.INDEX_STATE_NOINDEX): IndexStateNoIndex,
	C.index_state_t(C.INDEX_STATE_READY):   IndexStateReady,
	C.index_state_t(C.INDEX_STATE_ADD):     IndexStateAdd,
	C.index_state_t(C.INDEX_STATE_TRAIN):   IndexStateTrain,
	C.index_state_t(C.INDEX_STATE_DUMP):    IndexStateDump,
}

const (
	RetCodeOk           = "RET_CODE_OK"
	RetCodeErr          = "RET_CODE_ERR"
	RetCodeNoReady      = "RET_CODE_NOREADY"
	RetCodeMemoryErr    = "RET_CODE_MEMORYERR"
	RetCodeNoPermission = "RET_CODE_NOPERMISSION"
	RetCodeDataErr      = "RET_CODE_DATAERR"
	RetCodeIndexErr     = "RET_CODE_INDEXERR"
	RetCodeAdd2IndexErr = "RET_CODE_ADD2INDEXERR"
	RetCodeNoInit       = "RET_CODE_NOINIT"
	RetCodeUnknown      = "RET_CODE_UNKNOWN"
)

var CRetCodeMap = map[C.ret_code_t]string{
	C.ret_code_t(C.RET_CODE_OK):           RetCodeOk,
	C.ret_code_t(C.RET_CODE_ERR):          RetCodeErr,
	C.ret_code_t(C.RET_CODE_NOREADY):      RetCodeNoReady,
	C.ret_code_t(C.RET_CODE_MEMORYERR):    RetCodeMemoryErr,
	C.ret_code_t(C.RET_CODE_NOPERMISSION): RetCodeNoPermission,
	C.ret_code_t(C.RET_CODE_DATAERR):      RetCodeDataErr,
	C.ret_code_t(C.RET_CODE_INDEXERR):     RetCodeIndexErr,
	C.ret_code_t(C.RET_CODE_ADD2INDEXERR): RetCodeAdd2IndexErr,
	C.ret_code_t(C.RET_CODE_NOINIT):       RetCodeNoInit,
}

func NewScaNNIndex(indexWorkDir string, dimIn int, indexId uint64) (vdb *ScaNNIndex, err error) {
	if !utils.Exists(indexWorkDir) {
		utils.CreateDirPath(indexWorkDir)
	}
	workDirC := C.CString(indexWorkDir)
	defer C.free(unsafe.Pointer(workDirC))
	// config
	var vqliteConfig C.index_config_t
	vqliteConfig.dim_ = C.uint32_t(dimIn)
	vqliteConfig.brute_threshold_ = C.uint64_t(0) // 0 means default value 4096
	vqliteConfig.index_type_ = C.index_type_t(C.INDEX_TYPE_SCANN)
	vqliteConfig.storage_type_ = C.storage_type_t(C.STORAGE_FILE) // storage type: memory or file
	vqliteConfig.partitioning_train_sample_rate_ = C.float(0.2)
	vqliteConfig.hash_train_sample_rate_ = C.float(0.1)
	// init index
	vdbC := C.vqindex_init(workDirC, vqliteConfig)
	if vdbC == nil {
		err = fmt.Errorf("failed to create index")
		return
	} else {
		log.Info().Msgf("created index success indexWorkDir:%s, Dim: %v, indexId, %v", indexWorkDir, dimIn, indexId)
	}
	vdb = &ScaNNIndex{
		vdbC:         vdbC,
		Dim:          dimIn,
		IndexWorkDir: indexWorkDir,
		IndexId:      indexId,
	}

	return
}

func (vdb *ScaNNIndex) Destroy() {
	vdb.vdbCRwLock.Lock()
	tempPtr := vdb.vdbC
	vdb.vdbC = nil
	vdb.vdbCRwLock.Unlock()

	if tempPtr == nil {
		return
	}
	C.vqindex_release(tempPtr)
	log.Info().Msgf("Destroy VectoDB %+v", vdb)

}

func (vdb *ScaNNIndex) Search(xq []float32, k int, nprobe int, reorder int) ([][]VidScore, error) {
	vdb.vdbCRwLock.RLock()
	defer vdb.vdbCRwLock.RUnlock()

	if vdb.vdbC == nil {
		return nil, fmt.Errorf("index not initialized")
	}

	nq := len(xq) / vdb.Dim
	if nq < 1 {
		return nil, fmt.Errorf("invalid xq size")
	}

	var searchParams C.params_search_t
	searchParams.topk_ = C.uint32_t(k)
	searchParams.reorder_topk_ = C.uint32_t(reorder)
	searchParams.nprobe_ = C.uint32_t(nprobe)
	searchResult := make([]C.result_search_t, nq*k)

	exeCodeC, _ := conc.GetSQPool().Submit(func() (any, error) {
		exeCode := C.vqindex_search(vdb.vdbC, (*C.float)(&xq[0]), C.int(len(xq)), (*C.result_search_t)(&searchResult[0]), searchParams)
		return exeCode, nil
	}).Await()
	exeCode := exeCodeC.(C.ret_code_t)
	//exeCode := C.vqindex_search(vdb.vdbC, (*C.float)(&xq[0]), C.int(len(xq)), (*C.result_search_t)(&searchResult[0]), searchParams)
	if exeCode != 0 {
		//log.Errorf("search failed")
		log.Error().Msgf("search failed, exeCode: %v, msg: %s", exeCode, CRetCodeMap[exeCode])
		return nil, fmt.Errorf("search failed")
	}
	res := make([][]VidScore, nq)

	for i := 0; i < nq; i++ {
		for j := 0; j < k; j++ {
			score := float32(searchResult[i*k+j].score_)
			if score < 0 {
				break
			}
			vidScore := VidScore{Vid: int64(searchResult[i*k+j].vid_), Score: float32(searchResult[i*k+j].score_), From: vdb.IndexId}
			res[i] = append(res[i], vidScore)

		}
	}

	return res, nil
}

func (vdb *ScaNNIndex) AddWithIDs(vectors [][]float32, vids []int64) bool {
	vdb.vdbCRwLock.Lock()
	defer vdb.vdbCRwLock.Unlock()

	flattenedVectors := utils.FlattenFloat32Slice(vectors)

	nb := len(vids)
	if len(flattenedVectors) != nb*vdb.Dim {
		log.Error().Msgf("invalid length of vectors, want %v, have %v, nb %v, vdb.Dim %v ", nb*vdb.Dim, len(flattenedVectors), nb, vdb.Dim)
		return false
	}
	exeCodeC, _ := conc.GetDynamicPool().Submit(func() (any, error) {
		exeCode := C.vqindex_add(vdb.vdbC, (*C.float)(&flattenedVectors[0]), C.uint64_t(nb*vdb.Dim), (*C.int64_t)(&vids[0]))
		return exeCode, nil
	}).Await()

	exeCode := exeCodeC.(C.ret_code_t)
	//exeCode := C.vqindex_add(vdb.vdbC, (*C.float)(&flattenedVectors[0]), C.uint64_t(nb*vdb.Dim), (*C.int64_t)(&vids[0]))
	if exeCode != 0 {
		log.Error().Msgf("add failed, exeCode: %v,  msg: %s", exeCode, CRetCodeMap[exeCode])
		return false
	}
	log.Info().Msgf("AddWithIDs success, nb %v", nb)
	return true
}

func (vdb *ScaNNIndex) Statistics() IndexStatistics {
	vdb.vdbCRwLock.RLock()
	defer vdb.vdbCRwLock.RUnlock()
	statisticsC, _ := conc.GetDynamicPool().Submit(func() (any, error) {
		statisticsC := C.vqindex_stats(vdb.vdbC)
		return statisticsC, nil
	}).Await()

	statistics := statisticsC.(C.index_stats_t)

	//statistics := C.vqindex_stats(vdb.vdbC)
	datasetSize := int64(statistics.datasets_size_)
	vidSize := int64(statistics.vid_size_)
	indexSize := int64(statistics.index_size_)
	nlist := int32(statistics.index_nlist_)
	vecDim := int32(statistics.dim_)
	bruteThreshold := int64(statistics.brute_threshold_)
	isBruteIndication := int8(statistics.is_brute_)
	statusIndication := statistics.current_status_

	var isBrute bool

	if isBruteIndication == 1 {
		isBrute = true
	} else {
		isBrute = false
	}

	var status string
	if s, ok := CIndexStateMap[statusIndication]; ok {
		status = s
	} else {
		status = IndexStateUnknown
	}

	stat := IndexStatistics{
		DatasetSize:    datasetSize,
		VidSize:        vidSize,
		IndexSize:      indexSize,
		Nlist:          nlist,
		VecDim:         vecDim,
		BruteThreshold: bruteThreshold,
		IsBrute:        isBrute,
		Status:         status,
	}
	return stat
}

func (vdb *ScaNNIndex) Train(numThreads int) error {
	statistics := vdb.Statistics()
	trainType := C.train_type_t(C.TRAIN_TYPE_DEFAULT) // 0 TRAIN_TYPE_DEFAULT, 1 TRAIN_TYPE_NEW, 2 TRAIN_TYPE_ADD
	trainNlist := C.uint32_t(0)                       // 0 means use default value upNearestPower2(sqrt(n))
	trainNthreads := C.int32_t(numThreads)            // 0 means enable all cores

	if statistics.DatasetSize == 0 {
		errMsg := fmt.Errorf("train failed, dataset size is 0")
		log.Error().Err(errMsg)
		return errMsg
	}

	if statistics.DatasetSize == statistics.IndexSize {
		errMsg := fmt.Errorf("train failed, dataset size is equal to index size")
		log.Error().Err(errMsg)
		return errMsg
	}

	// train and dump index
	exeCode := C.vqindex_train(vdb.vdbC, trainType, trainNlist, trainNthreads)
	if exeCode != 0 {
		errMsg := fmt.Errorf("train failed, exeCode: %v, msg: %s", exeCode, CRetCodeMap[exeCode])
		log.Error().Err(errMsg)
		return errMsg
	} else {
		log.Info().Msgf("train success, exeCode: %v, msg: %s", exeCode, CRetCodeMap[exeCode])
	}
	return nil
}

func (vdb *ScaNNIndex) Dump() error {
	vdb.vdbCRwLock.RLock()
	defer vdb.vdbCRwLock.RUnlock()

	exeCode := C.vqindex_dump(vdb.vdbC)
	if exeCode != 0 {
		errMsg := fmt.Errorf("dump failed, exeCode: %v", exeCode)
		log.Error().Err(errMsg)
		return errMsg
	} else {
		log.Info().Msgf("dump success")
		return nil
	}
}
