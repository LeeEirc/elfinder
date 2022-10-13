package elfinder

import (
	"fmt"
	"net/http"
)

type OpenRequest struct {
	Init   bool   `elfinder:"init"` //  (true|false|not set)
	Target string `elfinder:"target"`
	Tree   bool   `elfinder:"tree"`
}

type OpenResponse struct {
	Api        float32      `json:"api,omitempty"`
	Cwd        FileInfo     `json:"cwd"`
	Files      []FileInfo   `json:"files,omitempty"`
	NetDrivers []string     `json:"netDrivers,omitempty"`
	UplMaxFile int          `json:"uplMaxFile"`
	UplMaxSize string       `json:"uplMaxSize"`        //  "32M"
	Options    Option       `json:"options,omitempty"` // Further information about the folder and its volume
	Debug      *DebugOption `json:"debug,omitempty"`   // Debug information, if you specify the corresponding connector option.
}

func OpenCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		param OpenRequest
		res   OpenResponse
	)

	if err := UnmarshalElfinderTag(&param, req.URL.Query()); err != nil {
		connector.Logger.Error(err)
		return
	}
	res.UplMaxSize = "32M"
	var (
		id   string
		path string
		err  error
		vol  FsVolume
	)
	vol = connector.DefaultVol
	id = connector.GetVolId(connector.DefaultVol)
	path = fmt.Sprintf("/%s", vol.Name())
	if param.Target != "" {
		id, path, err = connector.ParseTarget(param.Target)
		if err != nil {
			connector.Logger.Errorf("parse target %s err: %s", param.Target, err)
			if jsonErr := SendJson(rw, NewErr(ERROpen, err)); jsonErr != nil {
				connector.Logger.Errorf("send response json err: %s", err)
			}
			return
		}
		vol = connector.GetFsById(id)
	}
	if vol == nil {
		connector.Logger.Errorf("not found vol by id: %s", id)
		if err = SendJson(rw, NewErr(ERROpen, ErrNoFoundVol)); err != nil {
			connector.Logger.Errorf("send response json err: %s", err)
		}
		return
	}

	cwd, err2 := StatFsVolFileByPath(id, vol, path)
	if err2 != nil {
		if jsonErr := SendJson(rw, NewErr(ERROpen, err2)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	res.Cwd = cwd
	resFiles, err := ReadFsVolDir(id, vol, path)
	if err != nil {
		if jsonErr := SendJson(rw, NewErr(ERROpen, err)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	res.Files = append(res.Files, cwd)
	res.Files = append(res.Files, resFiles...)

	if param.Tree {
		for vid := range connector.Vols {
			if connector.Vols[vid].Name() != vol.Name() {
				vItem, err3 := StatFsVolFileByPath(vid, connector.Vols[vid], fmt.Sprintf("/%s", vol.Name()))
				if err3 != nil {
					connector.Logger.Error(err3)
					if jsonErr := SendJson(rw, NewErr(ERROpen, err3)); jsonErr != nil {
						connector.Logger.Error(jsonErr)
					}
					return
				}
				res.Files = append(res.Files, vItem)

			}
		}
	}
	if param.Init {
		res.Api = APIVERSION
		opt := NewDefaultOption()
		opt.Path = res.Cwd.Name
		res.Options = opt
		res.Cwd.Options = &opt
	}
	if err := SendJson(rw, &res); err != nil {
		connector.Logger.Error(err)
	}
}
