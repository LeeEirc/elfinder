package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/LeeEirc/elfinder"
)

func main() {
	mux := http.NewServeMux()
	connector := elfinder.NewConnector(elfinder.WithVolumes(NewLocalV()))
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./elf/"))))
	mux.Handle("/connector", connector)
	fmt.Println("Listen on :8000")
	log.Fatal(http.ListenAndServe(":8000", mux))
}

var (
	_ elfinder.FsVolume = (*LocalV)(nil)
)

func NewLocalV() elfinder.FsVolume {
	dir, _ := os.Getwd()
	info, err := os.Stat(dir)
	if err != nil {
		log.Fatal(err)
	}
	lfs := os.DirFS(dir)
	return LocalV{
		rootPath: dir,
		name:     info.Name(),
		FS:       lfs,
	}
}

type LocalV struct {
	rootPath string
	name     string
	fs.FS
}

func (l LocalV) Name() string {
	return l.name
}

func (l LocalV) Create(path string) (io.ReadWriteCloser, error) {
	absPath := l.getAbsPath(path)
	return os.Create(absPath)
}

func (l LocalV) Mkdir(path string) error {
	absPath := l.getAbsPath(path)
	return os.Mkdir(absPath, os.ModePerm)
}

func (l LocalV) Remove(path string) error {
	absPath := l.getAbsPath(path)
	return os.Remove(absPath)
}

func (l LocalV) Rename(old, new string) error {
	oldAbsPath := l.getAbsPath(old)
	newAbsPath := l.getAbsPath(new)
	return os.Rename(oldAbsPath, newAbsPath)
}

func (l LocalV) ReadDir(path string) ([]fs.DirEntry, error) {
	absPath := l.getAbsPath(path)
	return os.ReadDir(absPath)
}
func (l LocalV) getAbsPath(path string) string {
	return filepath.Join(l.rootPath, path)
}
