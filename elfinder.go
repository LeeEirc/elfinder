package elfinder

import (
	"archive/zip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/form"
)

const (
	APIVERSION    = 2.1050
	UPLOADMAXSIZE = "10M"
)

const (
	defaultZipMaxSize = 1024 * 1024 * 1024 // 1G
	defaultTmpPath    = "/tmp"
)

type Volumes []Volume

func NewElFinderConnector(vs Volumes) *ElFinderConnector {
	var volumeMap = make(map[string]Volume)
	for _, vol := range vs {
		volumeMap[vol.ID()] = vol
	}
	return &ElFinderConnector{Volumes: volumeMap, defaultV: vs[0], req: &ELFRequest{}, res: &ElfResponse{}}
}

func NewElFinderConnectorWithOption(vs Volumes, option map[string]string) *ElFinderConnector {
	var volumeMap = make(map[string]Volume)
	for _, vol := range vs {
		volumeMap[vol.ID()] = vol
	}
	var zipMaxSize int64
	var zipTmpPath string
	for k, v := range option {
		switch strings.ToLower(k) {
		case "zipmaxsize":
			if size, err := strconv.Atoi(v); err == nil && size > 0 {
				zipMaxSize = int64(size)
			}
		case "ziptmppath":
			if _, err := os.Stat(v); err != nil && os.IsNotExist(err) {
				err = os.MkdirAll(v, 0600)
				log.Fatal(err)
			}
			zipTmpPath = v
		}
	}
	if zipMaxSize == 0 {
		zipMaxSize = int64(defaultZipMaxSize)
	}

	if zipTmpPath == "" {
		zipTmpPath = defaultTmpPath
	}
	return &ElFinderConnector{Volumes: volumeMap, defaultV: vs[0], req: &ELFRequest{}, res: &ElfResponse{},
		zipTmpPath: zipTmpPath, zipMaxSize: zipMaxSize}
}

type ElFinderConnector struct {
	Volumes  map[string]Volume
	defaultV Volume
	req      *ELFRequest
	res      *ElfResponse

	zipMaxSize int64
	zipTmpPath string
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

	case "POST":
		err = req.ParseMultipartForm(32 << 20) // ToDo check 8Mb
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

	default:
		http.Error(rw, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	err = decoder.Decode(elf.req, req.Form)
	if err != nil {
		log.Println(err)
	}
	elf.dispatch(rw, req)
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
		path, err = elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
		if err != nil {
			elf.res.Error = []string{"errFolderNotFound"}
			return
		}
	}
	if path == "" || path == "/" {
		v = elf.defaultV
		ret.Cwd = v.RootFileDir()
		ret.Files = v.List(path)
	} else {
		v = elf.getVolume(IDAndTarget[0])
		ret.Cwd, err = v.Info(path)
		if err != nil {
			elf.res.Error = []string{"errFolderNotFound"}
			return
		}
		ret.Files = v.List(path)
	}
	ret.Files = append(ret.Files, ret.Cwd)
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
		for _, item := range v.Parents(path, 0) {
			ret.Files = append(ret.Files, item)
		}
	}
	elf.res = &ret
}

func (elf *ElFinderConnector) file() (read io.ReadCloser, filename string, err error) {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	if err != nil {
		return
	}
	filename = filepath.Base(path)
	reader, err := v.GetFile(path)
	return reader, filename, err
}

func (elf *ElFinderConnector) ls() {
	var path string
	elf.res.List = make([]string, 0)
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	if len(IDAndTarget) == 1 {
		path = "/"
	} else {
		path, _ = elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
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
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	if err != nil {
		elf.res.Error = err
		return
	}
	elf.res.Tree = v.Parents(path, 0)
}

func (elf *ElFinderConnector) mkDir() {
	added := make([]FileDir, 0)
	hashs := make(map[string]string)
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	if err != nil {
		elf.res.Error = []string{errMkdir, elf.req.Name}
		return
	}
	if elf.req.Name != "" {
		fileDir, err := v.MakeDir(path, elf.req.Name)
		if err != nil {
			elf.res.Error = []string{errMkdir, elf.req.Name}
			return
		}
		added = append(added, fileDir)
	}
	if len(elf.req.Dirs) != 0 {
		for _, name := range elf.req.Dirs {
			fileDir, err := v.MakeDir(path, name)
			if err != nil {
				elf.res.Error = []string{errMkdir, elf.req.Name}
				break
			}
			added = append(added, fileDir)
			hashs[name] = name
		}
	}
	elf.res.Added = added
	elf.res.Hashes = hashs
}

func (elf *ElFinderConnector) mkFile() {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	if err != nil {
		elf.res.Error = []string{"errMkfile", elf.req.Name}
		return
	}
	fileDir, err := v.MakeFile(path, elf.req.Name)
	if err != nil {
		elf.res.Error = []string{"errMkfile", elf.req.Name}
		return
	}
	elf.res.Added = []FileDir{fileDir}
}

func (elf *ElFinderConnector) paste() {
	//cut, copy, paste
	added := make([]FileDir, 0, len(elf.req.Targets))
	removed := make([]string, 0, len(elf.req.Targets))

	dstIDAndTarget := strings.Split(elf.req.Dst, "_")
	dstPath, err := elf.parseTarget(strings.Join(dstIDAndTarget[1:], "_"))
	if err != nil {
		elf.res.Error = errNotFound
		return
	}
	dstVol := elf.getVolume(dstIDAndTarget[0])
	for i, target := range elf.req.Targets {
		srcIDAndTarget := strings.Split(target, "_")
		srcVol := elf.getVolume(srcIDAndTarget[0])
		srcPath, err := elf.parseTarget(strings.Join(srcIDAndTarget[1:], "_"))
		if err != nil {
			log.Println("parse path err: ", err)
			continue
		}
		srcFileDir, err := srcVol.Info(srcPath)
		if err != nil {
			log.Println("Get File err: ", err)
			continue
		}
		if srcFileDir.Dirs == 1 {
			newDirName := srcFileDir.Name
			dstFolderFiles := dstVol.List(dstPath)
			for _, item := range dstFolderFiles {
				if item.Dirs == 1 && item.Name == srcFileDir.Name {
					newDirName = newDirName + elf.req.Suffix
				}
			}
			newDstDirFile, err := dstVol.MakeDir(dstPath, newDirName)
			if err != nil {
				log.Printf("Make Dir err: %s", err.Error())
				elf.res.Error = []string{errMsg, err.Error()}
				break
			}
			added = append(added, newDstDirFile)
			newAddFiles := elf.copyFolder(filepath.Join(dstPath, newDstDirFile.Name), srcPath, dstVol, srcVol)
			added = append(added, newAddFiles...)
		} else {
			srcFd, err := srcVol.GetFile(srcPath)
			if err != nil {
				log.Println("Get File err: ", err.Error())
				elf.res.Error = []string{errMsg, err.Error()}
				break
			}
			newFileDir, err := dstVol.Paste(dstPath, srcFileDir.Name, elf.req.Suffix, srcFd)
			if err != nil {
				log.Println("parse path err: ", err)
				elf.res.Error = []string{errMsg, err.Error()}
				break
			}
			added = append(added, newFileDir)
		}
		if elf.req.Cut {
			err = srcVol.Remove(srcPath)
			if err == nil {
				removed = append(removed, elf.req.Targets[i])
			} else {
				log.Println("cut file failed")
				elf.res.Error = []string{errMsg, err.Error()}
			}
		}
	}
	elf.res.Added = added
	elf.res.Removed = removed
}

func (elf *ElFinderConnector) copyFolder(dstPath, srcDir string, dstVol, srcVol Volume) (added []FileDir) {
	srcFiles := srcVol.List(srcDir)
	added = make([]FileDir, 0, len(srcFiles))
	for i := 0; i < len(srcFiles); i++ {
		srcPath := filepath.Join(srcDir, srcFiles[i].Name)
		if srcFiles[i].Dirs == 1 {
			subDirFile, err := dstVol.MakeDir(dstPath, srcFiles[i].Name)
			if err != nil {
				log.Printf("Make dir err: %s", err.Error())
				break
			}
			added = append(added, subDirFile)
			newDstPath := filepath.Join(dstPath, subDirFile.Name)
			subAdded := elf.copyFolder(newDstPath, srcPath, dstVol, srcVol)
			added = append(added, subAdded...)
		} else {
			srcFd, err := srcVol.GetFile(srcPath)
			if err != nil {
				log.Println("Get File err: ", err)
				continue
			}
			newFileDir, err := dstVol.Paste(dstPath, srcFiles[i].Name, elf.req.Suffix, srcFd)
			if err != nil {
				log.Println("parse path err: ", err)
				continue
			}
			added = append(added, newFileDir)
		}
	}
	return
}

func (elf *ElFinderConnector) ping() {

}

func (elf *ElFinderConnector) rename() {
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	if err != nil {
		elf.res.Error = []string{"errRename", elf.req.Name}
		return
	}
	fileDir, err := v.Rename(path, elf.req.Name)
	if err != nil {
		elf.res.Error = []string{"errRename", elf.req.Name}
		return
	}
	elf.res.Added = []FileDir{fileDir}
	elf.res.Removed = []string{elf.req.Target}

}

func (elf *ElFinderConnector) resize() {

}

func (elf *ElFinderConnector) rm() {
	removed := make([]string, 0, len(elf.req.Targets))
	for _, target := range elf.req.Targets {
		IDAndTarget := strings.Split(target, "_")
		v := elf.getVolume(IDAndTarget[0])
		path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
		if err != nil {
			log.Println(err)
			continue
		}
		err = v.Remove(path)
		if err != nil {
			log.Println(err)
			continue
		}
		removed = append(removed, target)
	}
	elf.res.Removed = removed
}

func (elf *ElFinderConnector) search() {
	var ret = ElfResponse{Files: []FileDir{}}
	var err error
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, _ := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	ret.Files, err = v.Search(path, elf.req.QueryKey, elf.req.Mimes...)
	if err != nil || len(ret.Files) == 0 {
		ret.Error = errNotFound
	}
	elf.res = &ret
}

func (elf *ElFinderConnector) size() {
	var totalSize int64
	for _, target := range elf.req.Targets {
		IDAndTarget := strings.Split(target, "_")
		v := elf.getVolume(IDAndTarget[0])
		path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
		if err != nil {
			log.Println(err)
			continue
		}
		tmpInfo, err := v.Info(path)

		if err != nil {
			log.Println(err)
			continue
		}
		if tmpInfo.Dirs == 1 {
			totalSize += calculateFolderSize(v, path)
		} else {
			totalSize += tmpInfo.Size
		}
	}
	elf.res.Size = totalSize
}

func (elf *ElFinderConnector) tree() {
	var ret = ElfResponse{Tree: []FileDir{}}
	IDAndTarget := strings.Split(elf.req.Target, "_")
	v := elf.getVolume(IDAndTarget[0])
	path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	if err != nil {
		elf.res.Error = err
		return
	}
	dirs := v.List(path)
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
	path, _ := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
	return v, path
}

func (elf *ElFinderConnector) dispatch(rw http.ResponseWriter, req *http.Request) {

	switch elf.req.Cmd {
	case "open":
		elf.open()
	case "tree":
		elf.tree()
	case "file":
		readFile, filename, err := elf.file()
		if err != nil {
			elf.res.Error = err.Error()
		} else {
			mimeType := mime.TypeByExtension(filepath.Ext(filename))
			rw.Header().Set("Content-Type", mimeType)
			if req.Form["download"] != nil {
				rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			} else {
				rw.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename=="%s"`, filename))
			}
			if req.Form.Get("cpath") != "" {
				http.SetCookie(rw, &http.Cookie{Path: req.Form.Get("cpath"), Name: "elfdl" + req.Form.Get("reqid"), Value: "1"})
			}
			_, err := io.Copy(rw, readFile)
			if err == nil {
				_ = readFile.Close()
				log.Printf("download file %s successful", filename)
				return
			} else {
				elf.res.Error = err.Error()
				log.Printf("download file %s err: %s", filename, err.Error())
			}
		}
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
		if len(elf.req.Targets) == 0 {
			targets := make([]string, 0, 5)
			for i := 0; i < 100; i++ {
				value := req.Form.Get(fmt.Sprintf("targets[%d]", i))
				if value == "" {
					break
				}
				targets = append(targets, value)
			}
			elf.req.Targets = targets
		}
		elf.size()
	case "upload":
		v, dirpath := elf.upload()
		files := req.MultipartForm.File["upload[]"]
		added := make([]FileDir, 0, len(files))
		errs := make([]string, 0, len(files))
		if elf.req.Cid != 0 && elf.req.Chunk != "" {
			re, err := regexp.Compile(`(.*?)\.([0-9][0-9]*?_[0-9][0-9]*?)(\.part)`)
			if err != nil {
				elf.res.Error = errFolderUpload
				break
			}
			ch := re.FindStringSubmatch(elf.req.Chunk)
			if len(ch) != 4 {
				elf.res.Error = errUploadFile
				break
			}
			t := strings.Split(ch[2], "_")
			currentPart, err := strconv.Atoi(t[0])
			if err != nil {
				elf.res.Error = errUploadFile
				break
			}
			totalPart, err := strconv.Atoi(t[1])
			if err != nil {
				elf.res.Error = errUploadFile
				break
			}
			rangeData := strings.Split(elf.req.Range, ",")
			if len(rangeData) != 3 {
				errs = append(errs, "err range data")
				break
			}
			offSet, err := strconv.Atoi(rangeData[0])
			if err != nil {
				elf.res.Error = errUploadFile
				break
			}
			chunkLength, err := strconv.Atoi(rangeData[1])
			if err != nil {
				elf.res.Error = errUploadFile
				break
			}
			totalSize, err := strconv.Atoi(rangeData[2])
			if err != nil {
				elf.res.Error = errUploadFile
				break
			}
			filename := ch[1]
			for i, uploadFile := range files {
				f, err := uploadFile.Open()
				if err != nil {
					errs = append(errs, err.Error())
					continue
				}
				data := ChunkRange{Offset: int64(offSet), Length: int64(chunkLength), TotalSize: int64(totalSize)}
				uploadPath := ""
				if len(elf.req.UploadPath) == len(files) && elf.req.UploadPath[i] != elf.req.Target {
					uploadPath = elf.req.UploadPath[i]
				}
				err = v.UploadChunk(elf.req.Cid, dirpath, uploadPath, filename, data, f)
				if err != nil {
					errs = append(errs, err.Error())
				}
				_ = f.Close()
			}
			if currentPart == totalPart {
				elf.res.Chunkmerged = fmt.Sprintf("%d_%d_%s", elf.req.Cid, totalPart, filename)
				elf.res.Name = filename
			}
		} else if elf.req.Chunk != "" {
			// Chunk merge request
			re, err := regexp.Compile(`([0-9]*)_([0-9]*)_(.*)`)
			if err != nil {
				elf.res.Error = errFolderUpload
				break
			}
			ch := re.FindStringSubmatch(elf.req.Chunk)
			if len(ch) != 4 {
				elf.res.Error = errFolderUpload
				break
			}
			var uploadPath string
			if len(elf.req.UploadPath) == 1 && elf.req.UploadPath[0] != elf.req.Target {
				uploadPath = elf.req.UploadPath[0]
			}

			cid, _ := strconv.Atoi(ch[1])
			total, _ := strconv.Atoi(ch[2])
			result, err := v.MergeChunk(cid, total, dirpath, uploadPath, ch[3])
			if err != nil {
				errs = append(errs, err.Error())
				break
			}
			added = append(added, result)
		} else {
			for i, uploadFile := range files {
				f, err := uploadFile.Open()
				uploadPath := ""
				if len(elf.req.UploadPath) == len(files) && elf.req.UploadPath[i] != elf.req.Target {
					uploadPath = elf.req.UploadPath[i]
				}
				result, err := v.UploadFile(dirpath, uploadPath, uploadFile.Filename, f)
				if err != nil {
					errs = append(errs, "errUpload")
					continue
				}
				added = append(added, result)
			}

		}
		elf.res.Warning = errs
		elf.res.Added = added
	case "zipdl":
		switch elf.req.Download {
		case "1":
			var fileKey string
			var filename string
			var mimetype string
			if len(elf.req.Targets) == 4 {
				fileKey = elf.req.Targets[1]
				filename = elf.req.Targets[2]
				mimetype = elf.req.Targets[3]
			}
			var ret ElfResponse
			if zipTmpPath, ok := getTmpFilePath(fileKey); ok {
				zipFd, err := os.Open(zipTmpPath)
				if err == nil {
					rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
					rw.Header().Set("Content-Type", mimetype)
					if _, err = io.Copy(rw, zipFd); err == nil {
						_ = zipFd.Close()
						delTmpFilePath(fileKey)
						return
					}
					rw.Header().Del("Content-Disposition")
					rw.Header().Del("Content-Type")
					ret.Error = err
					log.Println("zip download send err: ", err.Error())
				}
				log.Println("zip download err: ", err.Error())
				ret.Error = err
			}
			elf.res = &ret
		default:
			elf.zipdl()
		}
	case "abort":
		rw.WriteHeader(http.StatusNoContent)
		return
	case "search":
		elf.search()
	default:
		elf.res.Error = errUnknownCmd
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

func (elf *ElFinderConnector) zipdl() {
	var ret ElfResponse
	var zipWriter *zip.Writer
	var totalZipSize int64

	zipVs := make([]Volume, 0, len(elf.req.Targets))
	zipPaths := make([]string, 0, len(elf.req.Targets))
	for _, target := range elf.req.Targets {
		IDAndTarget := strings.Split(target, "_")
		v := elf.getVolume(IDAndTarget[0])
		path, err := elf.parseTarget(strings.Join(IDAndTarget[1:], "_"))
		if err != nil {
			log.Println(err)
			continue
		}
		zipVs = append(zipVs, v)
		zipPaths = append(zipPaths, path)
	}

	// check maxsize
	for i := 0; i < len(zipVs); i++ {
		info, err := zipVs[i].Info(zipPaths[i])
		if err != nil {
			continue
		}
		if info.Dirs == 1 {
			totalZipSize += calculateFolderSize(zipVs[i], zipPaths[i])
		} else {
			totalZipSize += info.Size
		}
	}

	if elf.zipMaxSize == 0 {
		elf.zipMaxSize = int64(defaultZipMaxSize)
	}
	if totalZipSize >= elf.zipMaxSize {
		ret.Error = errArcMaxSize
		elf.res = &ret
		return
	}

	zipRes := make(map[string]string)
	zipFileKey := GenerateTargetsMD5Key(elf.req.Targets...)
	if elf.zipTmpPath == "" {
		elf.zipTmpPath = defaultTmpPath
	}
	filename := fmt.Sprintf("%s%s.zip",
		time.Now().UTC().Format("20060102150405"), zipFileKey)
	zipTmpPath := filepath.Join(elf.zipTmpPath, filename)
	dstFd, err := os.Create(zipTmpPath)
	if err != nil {
		log.Println("create tmp zip file err: ", err)
		ret.Error = err.Error()
		elf.res = &ret
		return
	}

	zipWriter = zip.NewWriter(dstFd)
	for i := 0; i < len(zipVs); i++ {
		v := zipVs[i]
		path := zipPaths[i]
		info, err := v.Info(path)
		if err != nil {
			log.Println("Could not get info: ", path)
			ret.Error = err.Error()
			goto endErr
		}
		if info.Dirs == 0 {
			fheader := zip.FileHeader{
				Name:     info.Name,
				Modified: time.Now().UTC(),
				Method:   zip.Deflate,
			}
			zipFile, err := zipWriter.CreateHeader(&fheader)
			if err != nil {
				log.Println("Create zip err: ", err.Error())
				ret.Error = err.Error()
				goto endErr
			}
			reader, err := v.GetFile(path)
			if err != nil {
				log.Println("Get file err:", err.Error())
				ret.Error = err.Error()
				goto endErr
			}
			_, err = io.Copy(zipFile, reader)
			if err != nil {
				log.Println("Get file err:", err.Error())
				ret.Error = err.Error()
				goto endErr
			}
			_ = reader.Close()
		} else {
			if err := zipFolder(v, filepath.Dir(path), path, zipWriter); err != nil {
				log.Println("create tmp zip file err: ", err)
				ret.Error = err.Error()
				goto endErr
			}
		}
	}
	err = zipWriter.Close()
	if err != nil {
		log.Println("Zip file finish err: ", err)
		ret.Error = err.Error()
		goto endErr
	}
	setTmpFilePath(zipFileKey, zipTmpPath)
	zipRes["mime"] = "application/zip"
	zipRes["file"] = zipFileKey
	zipRes["name"] = filename
	ret.Zipdl = zipRes
endErr:
	elf.res = &ret

}

func GenerateTargetsMD5Key(targets ...string) string {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%d", time.Now().Nanosecond())))
	for _, target := range targets {
		h.Write([]byte(target))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func zipFolder(v Volume, baseFolder, folderPath string, zipW *zip.Writer) error {
	if !strings.HasSuffix(baseFolder, "/") {
		baseFolder += "/"
	}
	res := v.List(folderPath)
	for i := 0; i < len(res); i++ {
		currentPath := filepath.Join(folderPath, res[i].Name)
		if res[i].Dirs == 1 {
			err := zipFolder(v, baseFolder, currentPath, zipW)
			if err != nil {
				return err
			}
			continue
		}
		relPath := strings.TrimPrefix(currentPath, baseFolder)
		fheader := zip.FileHeader{
			Name:     relPath,
			Modified: time.Now().UTC(),
			Method:   zip.Deflate,
		}

		zipFile, err := zipW.CreateHeader(&fheader)
		if err != nil {
			return err
		}
		reader, err := v.GetFile(currentPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(zipFile, reader)
		if err != nil {
			return err
		}
		_ = reader.Close()
	}
	return nil
}

func calculateFolderSize(v Volume, folderPath string) int64 {
	var totalSize int64
	resInfos := v.List(folderPath)
	for i := 0; i < len(resInfos); i++ {
		currentPath := filepath.Join(folderPath, resInfos[i].Name)
		if resInfos[i].Dirs == 1 {
			totalSize += calculateFolderSize(v, currentPath)
		}
		totalSize += resInfos[i].Size
	}
	return totalSize
}
