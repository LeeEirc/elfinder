package connection

import (
	"github.com/LeeEirc/elfinder/codecs"
	"github.com/LeeEirc/elfinder/errs"
	"github.com/LeeEirc/elfinder/model"
	"net/http"
	"path/filepath"
)

type ParentsResponse struct {
	Tree []model.FileInfo `json:"tree"`
}

type ParentsRequest struct {
	Target string `elfinder:"target"`
}

func ParentsCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		param ParentsRequest
		res   ParentsResponse
	)

	err := codecs.UnmarshalElfinderTag(&param, req.URL.Query())
	if err != nil {
		connector.Logger.Error(err)
		if jsonErr := SendJson(rw, NewErr(errs.ERRCmdReq, err)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	target := param.Target
	id, path, err := connector.ParseTarget(target)
	if err != nil {
		connector.Logger.Error(err)
		if jsonErr := SendJson(rw, NewErr(errs.ERRCmdReq, err)); jsonErr != nil {
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
