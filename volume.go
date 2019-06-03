package elfinder

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"log"
)

var rootPath, _ = os.Getwd()

var DefaultVolume = LocalFileVolume{basePath: rootPath, Id: GenerateID(rootPath)}

type Volume interface {
	ID() string
	Info(path string) FileDir
	List(path string) []FileDir
	Parents(path string, dep int) []FileDir
	GetFile(path string) (reader io.Reader, err error)
	UploadFile(dir, filename string, reader io.Reader) (FileDir, error)
	MakeDir(dir, newDirname string)(FileDir,error)
	MakeFile(dir, newFilename string)(FileDir,error)
	RootFileDir() FileDir
}


func NewLocalVolume(path string)*LocalFileVolume{
	return &LocalFileVolume{
		basePath:path,
		Id:GenerateID(path),
	}
}

type LocalFileVolume struct {
	Id       string
	basePath string
}

func (f *LocalFileVolume) ID() string {
	return f.Id
}

func (f *LocalFileVolume) Info(path string) FileDir {
	var resFDir FileDir
	if path == "" || path == "/" {
		path = f.basePath
	}
	dirPath := filepath.Dir(path)
	if path != f.basePath {
		resFDir.Phash = f.hash(dirPath)
	}

	pathInfo, err := os.Stat(path)
	if err != nil {
		log.Println("open file err", err.Error())
		return resFDir
	}

	resFDir.Name = pathInfo.Name()
	resFDir.Hash = f.hash(path)
	resFDir.Ts = pathInfo.ModTime().Unix()
	resFDir.Size = pathInfo.Size()
	resFDir.Read, resFDir.Write = ReadWritePem(pathInfo.Mode())

	if pathInfo.IsDir() {
		resFDir.Mime = "directory"
		resFDir.Dirs = 1
	} else {
		resFDir.Mime = "file"
		resFDir.Dirs = 0
	}
	return resFDir
}

func (f *LocalFileVolume) List(path string) []FileDir {
	if path == "" || path == "/" {
		path = f.basePath
	}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return []FileDir{}
	}
	fileDir := make([]FileDir, 0, len(files))

	for _, item := range files {
		fileD := f.Info(filepath.Join(path, item.Name()))
		fileDir = append(fileDir, fileD)
	}

	return fileDir
}

func (f *LocalFileVolume) Parents(path string, dep int) []FileDir {
	relativepath := strings.TrimPrefix(strings.TrimPrefix(path, f.basePath), "/")
	relativePaths := strings.Split(relativepath, "/")
	dirs := make([]FileDir, 0, len(relativePaths))
	for i, _ := range relativePaths {
		realDirPath := filepath.Join(f.basePath, filepath.Join(relativePaths[:i+1]...))
		dirs = append(dirs, f.Info(realDirPath))
	}
	return dirs
}

func (f *LocalFileVolume) GetFile(path string) (reader io.Reader, err error) {
	freader, err := os.Open(path)
	return freader, err
}

func (f *LocalFileVolume) UploadFile(dirname, filename string, reader io.Reader) (FileDir,error){
	realPath := filepath.Join(dirname, filename)
	fwriter, err := os.OpenFile(realPath,os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil{
		return FileDir{}, err
	}
	_, err = io.Copy(fwriter,reader)
	if err != nil{
		return FileDir{}, err
	}
	fileDirInfo := f.Info(realPath)
	return fileDirInfo, nil
}

func (f *LocalFileVolume) hash(path string) string {
	return CreateHash(f.Id, path)
}

func (f *LocalFileVolume)MakeDir(dir, newDirname string)(FileDir,error)  {
	realPath := filepath.Join(dir,newDirname)
	err := os.Mkdir(realPath, os.ModePerm)
	if err != nil{
		return FileDir{}, err
	}
	return f.Info(realPath),nil
}

func (f *LocalFileVolume)MakeFile(dir, newFilename string)(FileDir,error){
	var res FileDir
	realPath := filepath.Join(dir,newFilename)
	fd, err := os.Create(realPath)
	if err != nil{
		return res, err
	}
	fdInfo ,err := fd.Stat()
	if err != nil{
		return res, err
	}
	res.Name = fdInfo.Name()
	res.Hash = f.hash(realPath)
	res.Phash = f.hash(dir)
	res.Ts = fdInfo.ModTime().Unix()
	res.Size = fdInfo.Size()
	res.Mime = "file"
	res.Dirs = 0
	res.Read, res.Write = ReadWritePem(fdInfo.Mode())
	return res,nil

}

func (f *LocalFileVolume) RootFileDir() FileDir {
	var resFDir= FileDir{}
	info, _ := os.Stat(f.basePath)
	resFDir.Name = info.Name()
	resFDir.Hash = f.hash(f.basePath)
	resFDir.Mime = "directory"
	resFDir.Volumeid = f.Id
	resFDir.Dirs = 1
	resFDir.Read, resFDir.Write = ReadWritePem(info.Mode())
	resFDir.Size = info.Size()
	return resFDir
}


