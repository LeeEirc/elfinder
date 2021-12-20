package elfinder

import (
	"io"
	"io/fs"
)


/*
	Create 接口的 path 是相对路径格式

*/


type FsVolume interface {
	Name() string
	fs.FS
	Create(path string) (io.ReadWriteCloser, error)
}
