package elfinder

import (
	"fmt"
	"net/http"
)

const defaultSuffix = "_elfinder_"

type UploadRequest struct {
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
	Warnings []ElfinderErr `json:"warning"`

	ChunkMerged string `json:"_chunkmerged"`
	ChunkName   string `json:"_name"`
}

func UploadCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		lsReq UploadRequest
		//res   UploadResponse
	)
	if err := UnmarshalElfinderTag(&lsReq, req.MultipartForm.Value); err != nil {
		connector.Logger.Error(err)
		return
	}
	fmt.Printf("upload: %+v\n", lsReq)
	if lsReq.Chunk == "" {

	}

}
