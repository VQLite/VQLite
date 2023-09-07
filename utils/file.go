package utils

import (
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// IsDir determine whether the file is dir
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// CreatNestedFile 给定path创建文件，如果目录不存在就递归创建
func CreatNestedFile(path string) (*os.File, error) {
	basePath := filepath.Dir(path)
	if !Exists(basePath) {
		err := os.MkdirAll(basePath, 0700)
		if err != nil {
			log.Warn().Msgf("cannot create dir，%s", err)
			return nil, err
		}
	}

	return os.Create(path)
}

func CreateDirPath(dirPath string) bool {
	if !Exists(dirPath) {
		err := os.MkdirAll(dirPath, 0700)
		if err != nil {
			log.Warn().Msgf("cannot create dir，%s", err)
			return false
		}
	}
	return true
}

func DeleteDir(path string) error {
	return os.RemoveAll(path)
}

// IsEmpty 返回给定目录是否为空目录
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func ParsePath(path string) string {
	path = strings.TrimRight(path, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func RemoveLastSlash(path string) string {
	if len(path) > 1 {
		return strings.TrimSuffix(path, "/")
	}
	return path
}

func Dir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == 0 {
		return "/"
	}
	if idx == -1 {
		return path
	}
	return path[:idx]
}

func Base(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

func Join(elem ...string) string {
	res := path.Join(elem...)
	if res == "\\" {
		res = "/"
	}
	return res
}

func Split(p string) (string, string) {
	return path.Split(p)
}

func Ext(name string) string {
	return strings.TrimPrefix(path.Ext(name), ".")
}

// DirSize return the size of the directory (KB)
func DirSize(path string) (float64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	sizeKB := float64(size) / 1024.0

	return sizeKB, err
}

func RemoveIndex(s []os.DirEntry, index int) []os.DirEntry {
	return append(s[:index], s[index+1:]...)
}

func FilterValidSegmentDirs(segmentsDirs []os.DirEntry) {
	for i := 0; i < len(segmentsDirs); i++ {
		if !segmentsDirs[i].IsDir() {
			RemoveIndex(segmentsDirs, i)
		}
		sParts := strings.Split(segmentsDirs[i].Name(), "_")
		if len(sParts) != 2 {
			RemoveIndex(segmentsDirs, i)
		}
	}
}
func SortFileNameAscend(segmentsDirs []os.DirEntry) {

	sort.Slice(segmentsDirs, func(i, j int) bool {
		pathA := segmentsDirs[i].Name()
		pathB := segmentsDirs[j].Name()
		a, err1 := strconv.ParseUint(pathA[strings.LastIndex(pathA, "_")+1:], 10, 64)
		b, err2 := strconv.ParseUint(pathB[strings.LastIndex(pathB, "_")+1:], 10, 64)
		if err1 != nil || err2 != nil {
			return pathA < pathB
		}
		return a < b
	})
}
