package elfinder

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-playground/form"
)

const (
	APIVERSION    = "2.1"
	UPLOADMAXSIZE = "10M"
)

type Volumes []Volume

func NewElFinderConnector(vs Volumes) *ElFinderConnector {
	var volumeMap = make(map[string]Volume)
	for _, vol := range vs {
		volumeMap[vol.ID()] = vol
	}
	return &ElFinderConnector{Volumes: volumeMap, defaultV: vs[0], req: &ELFRequest{}, res: &ElfResponse{}}
}

type ElFinderConnector struct {
	Volumes  map[string]Volume
	defaultV Volume
	req      *ELFRequest
	res      *ElfResponse
}

func (elf *ElFinderConnector) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var err error
	decoder := form.NewDecoder()
	switch req.Method {
	case "GET":
		if err := req.ParseForm(); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		err = decoder.Decode(elf.req, req.Form)
		if err != nil {
			log.Println(err)
		}

		elf.dispatch(rw, req)

	case "POST":
		err = req.ParseMultipartForm(32 << 20) // ToDo check 8Mb
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		err = decoder.Decode(elf.req, req.Form)
		if err != nil {
			log.Println(err)
		}
		elf.dispatch(rw, req)
	default:
		http.Error(rw, "Method Not Allowed", http.StatusMethodNotAllowed)
	}

}

func (elf *ElFinderConnector) open() {
	// client: reload, back, forward, home , open
	// open dir
	var ret ElfResponse
	var path string
	var v Volume
	var err error

	IDAndTarget := strings.Split(elf.req.Target, "_")
	if len(IDAndTarget) == 1 {
		path = "/"
	} else {
		path, err = elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
		if err != nil {
			elf.res.Error = []string{"errFolderNotFound"}
			return
		}
	}
	log.Println("open parseTarget path: ", path)

	if path == "" || path == "/" {
		v = elf.defaultV
		ret.Cwd = v.RootFileDir()
		ret.Files = v.List(path)
	} else {
		v = elf.getVolume(IDAndTarget[0])
		ret.Cwd = v.Info(path)
		ret.Files = v.List(path)
		ret.Files = append(ret.Files, ret.Cwd)
	}

	if elf.req.Init {
		ret.Api = APIVERSION
		ret.UplMaxSize = UPLOADMAXSIZE
		ret.Options = defaultOptions
	}

	if elf.req.Tree {
		ret.Tree = make([]FileDir, 0, len(elf.Volumes))
		for _, item := range elf.Volumes {
			ret.Files = append(ret.Files, item.RootFileDir())
		}
	}
	elf.res = &ret
}

func (elf *ElFinderConnector) file() (read io.ReadCloser, filename string, err error) {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	if err != nil {
		return
	}
	filename = filepath.Base(path)
	reader, err := v.GetFile(path)
	return reader, filename, err
}

func (elf *ElFinderConnector) get() {

}

func (elf *ElFinderConnector) ls() {
	var path string
	elf.res.List = make([]string, 0)
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	if len(IDAndTarget) == 1 {
		path = "/"
	} else {
		path, _ = elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	}

	dirs := v.List(path)
	resultFiles := make([]string, 0, len(dirs))
	if elf.req.Intersect != nil {

		for _, item := range dirs {
			for _, jitem := range elf.req.Intersect {
				if item.Name == jitem {
					resultFiles = append(resultFiles,
						fmt.Sprintf(`"%s";"%s"`, item.Hash, item.Name))
				}
			}
		}
	} else {
		for _, item := range dirs {
			resultFiles = append(resultFiles, fmt.Sprintf(`"%s";"%s"`, item.Hash, item.Name))
		}
	}

	elf.res.List = resultFiles

}

func (elf *ElFinderConnector) parents() {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	if err != nil {
		elf.res.Error = err
		return
	}

	log.Println(" parents parseTarget path: ", path)
	elf.res.Tree = v.Parents(path, 0)
}

func (elf *ElFinderConnector) mkDir() {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	if err != nil {
		elf.res.Error = []string{"errMkdir",elf.req.Name }
		return
	}
	fileDir,err := v.MakeDir(path,elf.req.Name)
	if err != nil{
		elf.res.Error = []string{"errMkdir",elf.req.Name}
		return
	}
	elf.res.Added = []FileDir{fileDir}
}

func (elf *ElFinderConnector) mkFile(){
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	if err != nil {
		elf.res.Error = []string{"errMkfile",elf.req.Name }
		return
	}
	fileDir, err := v.MakeFile(path ,elf.req.Name)
	if err != nil{
		elf.res.Error = []string{"errMkfile",elf.req.Name}
		return
	}
	elf.res.Added = []FileDir{fileDir}
}

func (elf *ElFinderConnector) paste() {
	//cut, copy, paste
	added := make([]FileDir,0,len(elf.req.Targets))
	removed := make([]string,0,len(elf.req.Targets))

	dstIDAndTarget := strings.Split(elf.req.Dst, "_")
	dstPath, err := elf.parseTarget(strings.Join(dstIDAndTarget[1:],"_"))
	if err != nil{
		elf.res.Error = []string{"errFolderNotFound"}
		return
	}
	dstVol := elf.getVolume(dstIDAndTarget[0])
	for i, target := range elf.req.Targets{
		srcIDAndTarget := strings.Split(target, "_")
		srcVol := elf.getVolume(srcIDAndTarget[0])
		srcPath, err := elf.parseTarget(strings.Join(srcIDAndTarget[1:],"_"))
		if err != nil{
			log.Println("parse path err: ", err)
			continue
		}
		srcFileDir := srcVol.Info(srcPath)
		srcFd ,err := srcVol.GetFile(srcPath)
		if err !=nil{
			log.Println("Get File err: ", err)
			continue
		}
		newFileDir,err := dstVol.Paste(dstPath,srcFileDir.Name,elf.req.Suffix,srcFd)
		if err != nil{
			log.Println("parse path err: ", err)
			continue
		}
		if elf.req.Cut{
			err = srcVol.Remove(srcPath)
			if err == nil{
				removed = append(removed,elf.req.Targets[i])
			}else {
				log.Println("cut file failed")
			}
		}
		added = append(added,newFileDir)
	}

	elf.res.Added = added
	elf.res.Removed = removed
}

func (elf *ElFinderConnector) ping() {

}

func (elf *ElFinderConnector) put() {
	// POST
}

func (elf *ElFinderConnector) rename() {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	if err != nil {
		elf.res.Error = []string{"errRename",elf.req.Name}
		return
	}
	fileDir, err := v.Rename(path, elf.req.Name)
	if err != nil {
		elf.res.Error = []string{"errRename",elf.req.Name}
		return
	}
	elf.res.Added = []FileDir{fileDir}
	elf.res.Removed = []string{elf.req.Target}

}

func (elf *ElFinderConnector) resize() {

}

func (elf *ElFinderConnector) rm() {
	removed := make([]string, 0, len(elf.req.Targets))
	for _, target := range elf.req.Targets{
		IDAndTarget := strings.Split(target, "_")
		v := elf.getVolume(IDAndTarget[0])
		path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
		if err !=nil{
			log.Println(err)
			continue
		}
		err = v.Remove(path)
		if err != nil{
			log.Println(err)
			continue
		}
		removed = append(removed,target)
	}
	elf.res.Removed = removed
}

func (elf *ElFinderConnector) search() {

}

func (elf *ElFinderConnector) size() {
	var totalSize int64
	for _, target := range elf.req.Targets{
		IDAndTarget := strings.Split(target, "_")
		v := elf.getVolume(IDAndTarget[0])
		path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
		if err != nil{
			log.Println(err)
			continue
		}
		tmpInfo := v.Info(path)
		totalSize += tmpInfo.Size
	}
	log.Println(totalSize)
	elf.res.Size = totalSize
}

func (elf *ElFinderConnector) tree() {
	var ret = ElfResponse{Tree: []FileDir{}}
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	if err != nil {
		elf.res.Error = err
		return
	}

	log.Println("tree parseTarget path: ", path)
	dirs := v.List(path)
	ret.Cwd = v.Info(path)
	for i, item := range v.List(path) {
		if item.Dirs == 1 {
			ret.Tree = append(ret.Tree, dirs[i])
		}
	}
	elf.res = &ret
}

func (elf *ElFinderConnector) upload() (Volume, string) {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, _ := elf.parseTarget(strings.Join(IDAndTarget[1:],"_"))
	return v, path
}

func (elf *ElFinderConnector) url() {

}

func (elf *ElFinderConnector) dispatch(rw http.ResponseWriter, req *http.Request) {

	switch elf.req.Cmd {
	case "open":
		elf.open()
	case "tree":
		elf.tree()
	case "file":
		readFile, filename, err := elf.file()
		defer readFile.Close()
		if err != nil {
			elf.res.Error = err.Error()
		} else {
			mimeType := mime.TypeByExtension(filepath.Ext(filename))
			rw.Header().Set("Content-Type", mimeType)
			if req.Form["download"] != nil {
				rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
			} else {
				rw.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename==%s", filename))
			}
			_, err := io.Copy(rw, readFile)
			if err == nil {
				log.Printf("download file %s successful", filename)
				return
			} else {
				elf.res.Error = err.Error()
				log.Printf("download file %s err: %s", filename, err.Error())
			}
		}
	case "get":

	case "info":

	case "ls":
		elf.ls()
	case "parents":
		elf.parents()
	case "mkdir":
		elf.mkDir()
	case "mkfile":
		elf.mkFile()
	case "paste":
		elf.paste()
	case "rename":
		elf.rename()
	case "rm":
		elf.rm()
	case "size":
		if len(elf.req.Targets)== 0{
			targets := make([]string,0, 5)
			for i:= 0; i<100;i++{
				value := req.Form.Get(fmt.Sprintf("targets[%d]", i))
				if value == ""{
					break
				}
				targets = append(targets,value)
			}
			elf.req.Targets = targets
		}
		elf.size()
	case "upload":
		v, dirpath := elf.upload()
		files := req.MultipartForm.File["upload[]"]
		added := make([]FileDir, 0, len(files))
		errs := make([]string, 0, len(files))
		for _, uploadFile := range files {
			f, err := uploadFile.Open()
			result, err := v.UploadFile(dirpath, uploadFile.Filename, f)
			if err != nil {
				errs = append(errs, "errUpload")
				continue
			}
			added = append(added, result)
		}
		if len(errs) >= 1 {
			elf.res.Warning = errs
		}
		elf.res.Added = added
	case "put":
		log.Println("=====put")
	}

	rw.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(elf.res)
	if err != nil {
		log.Println("elf Marshal err:", err.Error())
	}
	_, err = rw.Write(data)
	if err != nil {
		log.Println("ResponseWriter Write err:", err.Error())
	}

}

func (elf *ElFinderConnector) getVolume(vid string) Volume {
	if vid == "" {
		return elf.defaultV
	}
	log.Println("getVolume ", vid)
	if v, ok := elf.Volumes[vid]; ok {
		return v
	} else {
		return elf.defaultV
	}

}

func (elf *ElFinderConnector) parseTarget(target string) (path string, err error) {
	if target == "" || target == "/" {
		return "/", nil
	}
	path, err = Decode64(target)
	if err != nil {
		return "", err
	}
	return path, nil
}
