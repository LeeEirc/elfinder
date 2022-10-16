package connection

import (
	"fmt"
	"net/http"

	"github.com/LeeEirc/elfinder"
	"github.com/LeeEirc/elfinder/codecs"
	"github.com/LeeEirc/elfinder/errs"
	"github.com/LeeEirc/elfinder/model"
	"github.com/LeeEirc/elfinder/volumes"
)

type OpenRequest struct {
	Init   bool   `elfinder:"init"` //  (true|false|not set)
	Target string `elfinder:"target"`
	Tree   bool   `elfinder:"tree"`
}

type OpenResponse struct {
	Api        float32            `json:"api,omitempty"`
	Cwd        model.FileInfo     `json:"cwd"`
	Files      []model.FileInfo   `json:"files,omitempty"`
	NetDrivers []string           `json:"netDrivers,omitempty"`
	UplMaxFile int                `json:"uplMaxFile"`
	UplMaxSize string             `json:"uplMaxSize"`        //  "32M"
	Options    model.Option       `json:"options,omitempty"` // Further information about the folder and its volume
	Debug      *model.DebugOption `json:"debug,omitempty"`   // Debug information, if you specify the corresponding connection option.
}

func OpenCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		param OpenRequest
		res   OpenResponse
	)

	if err := codecs.UnmarshalElfinderTag(&param, req.URL.Query()); err != nil {
		connector.Logger.Error(err)
		return
	}
	res.UplMaxSize = "32M"
	var (
		id   string
		path string
		err  error
		vol  volumes.FsVolume
	)
	vol = connector.DefaultVol
	id = connector.GetVolId(connector.DefaultVol)
	path = fmt.Sprintf("/%s", vol.Name())
	if param.Target != "" {
		id, path, err = connector.ParseTarget(param.Target)
		if err != nil {
			connector.Logger.Errorf("parse target %s errs: %s", param.Target, err)
			if jsonErr := SendJson(rw, NewErr(errs.ERROpen, err)); jsonErr != nil {
				connector.Logger.Errorf("send response json errs: %s", err)
			}
			return
		}
		vol = connector.GetFsById(id)
	}
	if vol == nil {
		connector.Logger.Errorf("not found vol by id: %s", id)
		if err = SendJson(rw, NewErr(errs.ERROpen, ErrNoFoundVol)); err != nil {
			connector.Logger.Errorf("send response json errs: %s", err)
		}
		return
	}

	cwd, err2 := StatFsVolFileByPath(id, vol, path)
	if err2 != nil {
		if jsonErr := SendJson(rw, NewErr(errs.ERROpen, err2)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	res.Cwd = cwd
	resFiles, err := ReadFsVolDir(id, vol, path)
	if err != nil {
		if jsonErr := SendJson(rw, NewErr(errs.ERROpen, err)); jsonErr != nil {
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
					if jsonErr := SendJson(rw, NewErr(errs.ERROpen, err3)); jsonErr != nil {
						connector.Logger.Error(jsonErr)
					}
					return
				}
				res.Files = append(res.Files, vItem)

			}
		}
	}
	if param.Init {
		res.Api = elfinder.APIVERSION
		opt := model.NewDefaultOption()
		opt.Path = res.Cwd.Name
		res.Options = opt
		res.Cwd.Options = &opt
	}
	if err := SendJson(rw, &res); err != nil {
		connector.Logger.Error(err)
	}
}
