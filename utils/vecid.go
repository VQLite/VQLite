package utils

import (
	"fmt"
	"sync/atomic"
	"time"
)

const MaxUint32 = int64(^uint32(0))

// EncodeVectorId encode vqid
func EncodeVectorId(docId, extra int64) (int64, error) {
	if docId > MaxUint32 || extra > MaxUint32 {
		return 0, fmt.Errorf("docId or extra is too large, %d, %d", docId, extra)
	}
	return (docId << 32) + extra, nil
}

// DecodeVectorId decode vqid
func DecodeVectorId(docId int64) (int64, int64) {
	return docId >> 32, docId & 0xFFFFFFFF
}

var reqid = func() func() uint64 {
	var id = uint64(time.Now().UnixNano())
	return func() uint64 {
		return atomic.AddUint64(&id, 1)
	}
}()
