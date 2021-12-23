package elfinder

import (
	"net/http"
	"path/filepath"
)

type ParentsResponse struct {
	Tree []FileInfo `json:"tree"`
}

type ParentsRequest struct {
	Target string `elfinder:"target"`
}

func ParentsCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		param ParentsRequest
		res   ParentsResponse
	)

	err := UnmarshalElfinderTag(&param, req.URL.Query())
	if err != nil {
		connector.Logger.Error(err)
		if jsonErr := SendJson(rw, NewErr(ERRCmdReq, err)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	target := param.Target
	id, path, err := connector.parseTarget(target)
	if err != nil {
		connector.Logger.Error(err)
		if jsonErr := SendJson(rw, NewErr(ERRCmdReq, err)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	vol := connector.Vols[id]
	cwdInfo, err := StatFsVolFileByPath(id, vol, path)
	if err != nil {
		connector.Logger.Error(err)
		return
	}
	res.Tree = append(res.Tree, cwdInfo)
	for {
		path = filepath.Dir(path)
		if path == "/" {
			break
		}
		cwdInfo, err = StatFsVolFileByPath(id, vol, path)
		if err != nil {
			connector.Logger.Error(err)
			return
		}
		res.Tree = append(res.Tree, cwdInfo)

		cwdDirs, err := ReadFsVolDir(id, vol, path)
		if err != nil {
			connector.Logger.Error(err)
			return
		}
		res.Tree = append(res.Tree, cwdDirs...)
	}

	if err := SendJson(rw, &res); err != nil {
		connector.Logger.Error(err)
	}

}
