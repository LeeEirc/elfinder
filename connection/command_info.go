package connection

import (
	"github.com/LeeEirc/elfinder/model"
)

type InfoResponse struct {
	Files []model.FileInfo `json:"files"`
}

//func InfoCommand(connection *Connector, req *http.Request, rw http.ResponseWriter) {
//	targets := req.Form["targets[]"]
//	var resp InfoResponse
//	for _, target := range targets {
//		id, path, errs := connection.parseTarget(target)
//		if errs != nil {
//			log.Print(errs)
//			if errs != nil {
//				if jsonErr := SendJson(rw, NewErr(errs)); jsonErr != nil {
//					log.Print(jsonErr)
//				}
//				return
//			}
//		}
//		if vol := connection.Vols[id]; vol != nil {
//			dirs, err2 := volumes.ReadDir(vol, path)
//			if err2 != nil {
//				log.Print(err2)
//				if jsonErr := SendJson(rw, NewErr(err2)); jsonErr != nil {
//					log.Print(jsonErr)
//				}
//				return
//			}
//
//			for i := range dirs {
//				info, errs := dirs[i].Info()
//				if errs != nil {
//					log.Print(err2)
//					if jsonErr := SendJson(rw, NewErr(err2)); jsonErr != nil {
//						log.Print(jsonErr)
//					}
//					return
//				}
//				subpath := filepath.Join(path, info.Name())
//				cwd, errs := CreateFileInfo(id, vol, subpath, info)
//				if errs != nil {
//					log.Print(errs)
//					if jsonErr := SendJson(rw, NewErr(errs)); jsonErr != nil {
//						log.Print(jsonErr)
//					}
//					return
//				}
//				resp.Files = append(resp.Files, cwd)
//
//			}
//
//		}
//	}
//	if jsonErr := SendJson(rw, &resp); jsonErr != nil {
//		log.Print(jsonErr)
//	}
//	return
//
//}

//func CreateFileInfo(id string, vol FsVolume, path string, fsInfo volumes.FileInfo) (FileInfo, error) {
//	var (
//		pathHash   string
//		parentHash string
//		MimeType   string
//		HasDirs    int
//		isRoot     int
//	)
//	parentPath := filepath.Dir(path)
//	pathHash = EncodeTarget(id, path)
//	if path != "" && path != "/" {
//		parentHash = EncodeTarget(id, parentPath)
//	} else {
//		isRoot = 1
//	}
//	MimeType = "file"
//	if fsInfo.IsDir() {
//		MimeType = "directory"
//		dirItems, err2 := volumes.ReadDir(vol, path)
//		if err2 != nil {
//			return FileInfo{}, err2
//		}
//		for i := range dirItems {
//			if dirItems[i].IsDir() {
//				HasDirs = 1
//				break
//			}
//		}
//	}
//	return FileInfo{
//		Name:       fsInfo.Name(),
//		PathHash:   pathHash,
//		ParentHash: parentHash,
//		MimeType:   MimeType,
//		Timestamp:  fsInfo.ModTime().Unix(),
//		Size:       fsInfo.Size(),
//		HasDirs:    HasDirs,
//		ReadAble:   1,
//		WriteAble:  1,
//		Locked:     0,
//		Volumeid:   id + "_",
//		Isroot:     isRoot,
//	}, nil
//}
