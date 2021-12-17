package elfinder

import (
	"fmt"
	"net/http"
)

type LsRequest struct {
	Target     string   `elfinder:"target"`
	Intersects []string `elfinder:"intersect[]"`
}

type LsResponse struct {
	List map[string]string `json:"list"`
}

func LsCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		lsReq LsRequest
		res   LsResponse
	)

	if err := UnmarshalElfinderTag(&lsReq, req.URL.Query()); err != nil {
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
			if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
				connector.Logger.Errorf("send response json err: %s", err)
			}
			return
		}
		vol = connector.GetFsById(id)
	}
	if vol == nil {
		connector.Logger.Errorf("not found vol by id: %s", id)
		if jsonErr := SendJson(rw, NewErr(ErrNoFoundVol)); jsonErr != nil {
			connector.Logger.Errorf("send response json err: %s", err)
		}
		return
	}

	resFiles, err := ReadFsVolDir(id, vol, path)
	if err != nil {
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	if len(resFiles) > 0 {
		items := make(map[string]string, len(resFiles))
		res.List = make(map[string]string, len(resFiles))
		for i := range resFiles {
			items[resFiles[i].Name] = resFiles[i].PathHash
		}
		for i := range lsReq.Intersects {
			name := lsReq.Intersects[i]
			if hashPath, ok := items[name]; ok {
				res.List[hashPath] = name
			}
		}
	}

	if err = SendJson(rw, NewErr(ErrNoFoundVol)); err != nil {
		connector.Logger.Errorf("send response json err: %s", err)
	}
}
