package elfinder

import (
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
)

type InfoResponse struct {
	Files []FileInfo `json:"files"`
}

func InfoCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	targets := req.Form["targets[]"]
	var resp InfoResponse
	for _, target := range targets {
		id, path, err := connector.parseTarget(target)
		if err != nil {
			log.Print(err)
			if err != nil {
				if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
					log.Print(jsonErr)
				}
				return
			}
		}
		if vol := connector.Vols[id]; vol != nil {
			dirs, err2 := fs.ReadDir(vol, path)
			if err2 != nil {
				log.Print(err2)
				if jsonErr := SendJson(rw, NewErr(err2)); jsonErr != nil {
					log.Print(jsonErr)
				}
				return
			}

			for i := range dirs {
				info, err := dirs[i].Info()
				if err != nil {
					log.Print(err2)
					if jsonErr := SendJson(rw, NewErr(err2)); jsonErr != nil {
						log.Print(jsonErr)
					}
					return
				}
				subpath := filepath.Join(path, info.Name())
				cwd, err := CreateFileInfo(id, vol, subpath, info)
				if err != nil {
					log.Print(err)
					if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
						log.Print(jsonErr)
					}
					return
				}
				resp.Files = append(resp.Files, cwd)

			}

		}
	}
	if jsonErr := SendJson(rw, &resp); jsonErr != nil {
		log.Print(jsonErr)
	}
	return

}


func CreateFileInfo(id string, vol FsVolume, path string, fsInfo fs.FileInfo) (FileInfo, error) {
	var (
		pathHash   string
		parentHash string
		MimeType   string
		HasDirs    int
		isRoot     int
	)
	parentPath := filepath.Dir(path)
	pathHash = EncodeTarget(id, path)
	if path != "" && path != "/" {
		parentHash = EncodeTarget(id, parentPath)
	} else {
		isRoot = 1
	}
	MimeType = "file"
	if fsInfo.IsDir() {
		MimeType = "directory"
		dirItems, err2 := fs.ReadDir(vol, path)
		if err2 != nil {
			return FileInfo{}, err2
		}
		for i := range dirItems {
			if dirItems[i].IsDir() {
				HasDirs = 1
				break
			}
		}
	}
	return FileInfo{
		Name:       fsInfo.Name(),
		PathHash:   pathHash,
		ParentHash: parentHash,
		MimeType:   MimeType,
		Timestamp:  fsInfo.ModTime().Unix(),
		Size:       fsInfo.Size(),
		HasDirs:    HasDirs,
		ReadAble:   1,
		WriteAble:  1,
		Locked:     0,
		Volumeid:   id + "_",
		Isroot:     isRoot,
	}, nil
}