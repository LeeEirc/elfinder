package elfinder

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultSuffix = "_elfinder_"

type UploadRequest struct {
	ReqId       string   `elfinder:"reqid"`
	Target      string   `elfinder:"target"`
	Uploads     []string `elfinder:"upload[]"`
	UploadPaths []string `elfinder:"upload_path[]"`
	MTimes      []string `elfinder:"mtime[]"`
	Names       []string `elfinder:"name[]"`
	Renames     []string `elfinder:"renames[]"`
	Suffix      string   `elfinder:"suffix"`
	Hashes      []string `elfinder:"hashes[hash]"`
	Overwrite   bool     `elfinder:"overwrite"`

	Chunk string `elfinder:"chunk"`
	Cid   string `elfinder:"cid"`
	Range string `elfinder:"range"`
}

type UploadResponse struct {
	Adds     []FileInfo    `json:"added"`
	Warnings []ErrResponse `json:"warning"`

	ChunkMerged string `json:"_chunkmerged"`
	ChunkName   string `json:"_name"`
}

func UploadCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		lsReq UploadRequest
		res   UploadResponse
	)
	if err := UnmarshalElfinderTag(&lsReq, req.MultipartForm.Value); err != nil {
		connector.Logger.Error(err)
		return
	}
	var (
		id   string
		path string
		err  error
		vol  FsVolume
	)
	vol = connector.DefaultVol
	id = connector.GetVolId(connector.DefaultVol)
	path = fmt.Sprintf("/%s", vol.Name())

	if lsReq.Target != "" {
		id, path, err = connector.parseTarget(lsReq.Target)
		if err != nil {
			connector.Logger.Errorf("parse target %s err: %s", lsReq.Target, err)
			if jsonErr := SendJson(rw, NewErr(ERRCmdParams, err)); jsonErr != nil {
				connector.Logger.Errorf("send response json err: %s", err)
			}
			return
		}
		vol = connector.GetFsById(id)
	}
	if vol == nil {
		connector.Logger.Errorf("not found vol by id: %s", id)
		if jsonErr := SendJson(rw, NewErr(ERRCmdParams, ErrNoFoundVol)); jsonErr != nil {
			connector.Logger.Errorf("send response json err: %s", err)
		}
		return
	}

	uploadFiles := req.MultipartForm.File["upload[]"]
	fmt.Printf("upload: %+v\n", lsReq)
	fmt.Printf("path: %+v\n", path)
	var errs []ErrResponse
	if lsReq.Chunk == "" {
		for i := range uploadFiles {
			cwdFile := uploadFiles[i]
			fmt.Println(cwdFile.Filename, cwdFile.Size, cwdFile.Header)
			cwdFd, err := cwdFile.Open()
			if err != nil {
				errs = append(errs, NewErr(ERRUpload, err))
				continue
			}
			currentPath := strings.Join([]string{path, cwdFile.Filename}, Separator)
			relativePath := strings.TrimPrefix(currentPath, fmt.Sprintf("/%s/", vol.Name()))
			if writer, err2 := vol.Create(relativePath); err2 == nil {
				_, err3 := io.Copy(writer, cwdFd)
				if err3 != nil {
					connector.Logger.Errorf("upload file %s err:", cwdFile.Filename, err3)
				} else {
					if info, err := StatFsVolFileByPath(id, vol, currentPath); err == nil {
						res.Adds = append(res.Adds, info)
					}
				}
			}
			_ = cwdFd.Close()
		}
		if len(errs) > 0 {
			res.Warnings = errs
		}
	}
}
