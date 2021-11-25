package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeeEirc/elfinder"
)

func main() {
	mux := http.NewServeMux()
	dir, _ := os.Getwd()
	info, _ := os.Stat(dir)
	lfs := localFs{dir, info}
	connector := elfinder.NewConnector(&lfs)
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./elf/"))))
	mux.Handle("/connector", connector)
	fmt.Println("Listen on :8000")
	log.Fatal(http.ListenAndServe(":8000", mux))
}

var (
	_ elfinder.NewVolume = (*localFs)(nil)
)

type localFs struct {
	RootPath string
	fs.FileInfo
}

func (l *localFs) Open(name string) (fs.File, error) {
	path := strings.TrimPrefix(name, l.Name())
	return os.Open(filepath.Join(l.RootPath, path))
}
