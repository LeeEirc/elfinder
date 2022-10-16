package volumes

import (
	"io"
	"io/fs"
)

/*
	Create 接口的 path 是相对路径格式

*/

type FsVolume interface {
	Name() string
	fs.ReadDirFS
	Create(path string) (io.ReadWriteCloser, error)
	Mkdir(path string) error
	Remove(path string) error
	Rename(old, new string) error
}
