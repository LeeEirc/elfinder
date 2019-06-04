package elfinder

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var rootPath, _ = os.Getwd()

var DefaultVolume = LocalFileVolume{basePath: rootPath, Id: GenerateID(rootPath)}

type Volume interface {
	ID() string
	Info(path string) (FileDir, error)
	List(path string) []FileDir
	Parents(path string, dep int) []FileDir
	GetFile(path string) (reader io.ReadCloser, err error)
	UploadFile(dir, filename string, reader io.Reader) (FileDir, error)
	UploadChunk(cid int, dirPath, chunkName string,reader io.Reader) error
	MergeChunk(cid, total int,dirPath,filename string)  (FileDir, error)
	CompleteChunk(cid, total int,dirPath,filename string) bool
	MakeDir(dir, newDirname string)(FileDir,error)
	MakeFile(dir, newFilename string)(FileDir,error)
	Rename(oldNamePath, newname string)(FileDir,error)
	Remove(path string) error
	Paste(dir, filename, suffix string, reader io.ReadCloser)(FileDir,error)
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

func (f *LocalFileVolume) Info(path string) (FileDir, error) {
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
		return resFDir, err
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
	return resFDir, nil
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
		fileD, err := f.Info(filepath.Join(path, item.Name()))
		if err != nil{
			continue
		}
		fileDir = append(fileDir, fileD)
	}

	return fileDir
}

func (f *LocalFileVolume) Parents(path string, dep int) []FileDir {
	relativepath := strings.TrimPrefix(strings.TrimPrefix(path, f.basePath), "/")
	relativePaths := strings.Split(relativepath, "/")
	dirs := make([]FileDir, 0, len(relativePaths))
	for i, _ := range relativePaths {
		realDirPath := filepath.Join(f.basePath, filepath.Join(relativePaths[:i]...))
		result, err := f.Info(realDirPath)
		if err != nil{
			continue
		}
		dirs = append(dirs, result)
		tmpDir := f.List(realDirPath)
		for j, item := range tmpDir{
			if item.Dirs == 1{
				dirs = append(dirs, tmpDir[j])
			}
		}
	}
	return dirs
}

func (f *LocalFileVolume) GetFile(path string) (reader io.ReadCloser, err error) {
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
	return f.Info(realPath)
}

func (f *LocalFileVolume) UploadChunk(cid int, dirPath, chunkName string,reader io.Reader) error{
	// chunkName_cid chunkName format "filename.[NUMBER]_[TOTAL].part"
	chunkpath := filepath.Join(dirPath,chunkName)
	chunkRealpath := fmt.Sprintf("%s_%d",chunkpath,cid)
	fd, err := os.OpenFile(chunkRealpath, os.O_WRONLY | os.O_CREATE, 0666)
	defer fd.Close()
	if err != nil{
		return err
	}
	_, err = io.Copy(fd,reader)
	if err != nil{
		return  err
	}
	return nil
}

func (f *LocalFileVolume) MergeChunk(cid, total int, dirPath,filename string) (FileDir, error){
	realPath := filepath.Join(dirPath,filename)
	fd, err := os.OpenFile(realPath, os.O_WRONLY | os.O_CREATE, 0666)
	defer fd.Close()
	if err != nil{
		return FileDir{},err
	}
	for i:=0;i<=total;i++{
		partPath:= fmt.Sprintf("%s.%d_%d.part_%d", realPath,i,total,cid)
		partFD, err := os.Open(partPath)
		if err !=nil{
			return FileDir{},err
		}
		_, err = io.Copy(fd,partFD)
		if err != nil{
			return FileDir{},err
		}
		_ = os.RemoveAll(partPath)
	}
	return f.Info(realPath)
}

func (f *LocalFileVolume) CompleteChunk(cid, total int,dirPath,filename string) bool{
	realPath := filepath.Join(dirPath,filename)
	for i:=0;i<=total;i++{
		partPath:= fmt.Sprintf("%s.%d_%d.part_%d", realPath,i,total,cid)
		_, err :=  f.Info(partPath)
		if err != nil{
			return false
		}
	}
	return true
}

func (f *LocalFileVolume) hash(path string) string {
	return CreateHash(f.Id, path)
}

func (f *LocalFileVolume) MakeDir(dir, newDirname string)(FileDir,error)  {
	realPath := filepath.Join(dir,newDirname)
	err := os.Mkdir(realPath, os.ModePerm)
	if err != nil{
		return FileDir{}, err
	}
	return f.Info(realPath)
}

func (f *LocalFileVolume) MakeFile (dir, newFilename string)(FileDir,error){
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

func (f *LocalFileVolume) Rename (oldNamePath, newName string)(FileDir,error){
	var res FileDir
	dirname := filepath.Dir(oldNamePath)
	realNewNamePath := filepath.Join(dirname,newName)
	err := os.Rename(oldNamePath,realNewNamePath)
	if err!= nil{
		return res,err
	}
	return f.Info(realNewNamePath)
}

func (f *LocalFileVolume) Remove(path string) error{
	return os.RemoveAll(path)
}

func (f *LocalFileVolume) Paste(dir, filename, suffix string, reader io.ReadCloser)(FileDir,error){
	defer reader.Close()
	res := FileDir{}
	realpath := filepath.Join(dir,filename)
	_, err := f.Info(realpath)
	if err == nil {
		realpath += suffix
	}
	dstFd, err := os.Create(realpath)
	if err != nil{
		return res,err
	}
	_, err = io.Copy(dstFd,reader)
	if err != nil{
		return res, err
	}
	return f.Info(realpath)
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
	resFDir.Locked = 1
	return resFDir
}


