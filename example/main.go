package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

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
		name: info.Name(),
		FS:   lfs,
	}
}

type LocalV struct {
	name string
	fs.FS
}

func (l LocalV) Name() string {
	return l.name
}

func (l LocalV) Create(path string) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("no support %s", l.name)
}

func (l LocalV) Mkdir(path string) error {
	return nil
}

func (l LocalV) Remove(path string) error {
	return nil
}

func (l LocalV) Rename(old, new string) error {
	return nil
}

func (l LocalV) ReadDir(path string) ([]fs.DirEntry, error) {
	return nil, nil
}
