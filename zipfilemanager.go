package elfinder

import (
	"os"
	"sync"
)

var (
	zipTmpFiles = make(map[string]string)
	zipLocker   = new(sync.Mutex)
)

func getTmpFilePath(key string) (string, bool) {
	zipLocker.Lock()
	defer zipLocker.Unlock()
	path, ok := zipTmpFiles[key]
	return path, ok
}

func setTmpFilePath(key, path string) {
	zipLocker.Lock()
	defer zipLocker.Unlock()
	zipTmpFiles[key] = path
}

func delTmpFilePath(key string) {
	zipLocker.Lock()
	defer zipLocker.Unlock()
	if path, ok := zipTmpFiles[key]; ok{
		_ = os.RemoveAll(path)
	}
	delete(zipTmpFiles,key)
}
