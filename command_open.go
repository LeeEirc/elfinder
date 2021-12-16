package elfinder

import (
	"fmt"
	"io/fs"
	"log"
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
	var param OpenRequest

	if err := UnmarshalElfinderTag(&param, req.URL.Query()); err != nil {
		log.Print(err)
		return
	}

	var res OpenResponse

	if param.Init {
		res.Api = APIVERSION
	}
	res.UplMaxSize = "32M"
	var (
		id   string
		path string
		err  error
		vol  NewVolume
	)
	if param.Target == "" {
		vol = connector.DefaultVol
		id = connector.GetVolId(connector.DefaultVol)
		path = fmt.Sprintf("/%s", vol.Name())
	} else {
		id, path, err = connector.GetVolByTarget(param.Target)
		if err != nil {
			log.Print(err)
			if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
				log.Print(jsonErr)
			}
			return
		}
		vol = connector.Vols[id]
	}
	if vol == nil || id == "" {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(ErrNoFoundVol)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}

	cwd, err2 := CreateFileInfoByPath(id, vol, path)
	if err2 != nil {
		log.Print("CreateFileInfoByPath", err2)
		if jsonErr := SendJson(rw, NewErr(err2)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	res.Cwd = cwd
	resFiles, err := ReadFilesByPath(id, vol, path)
	if err != nil {
		log.Print("ReadFilesByPath", err)
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	res.Files = append(res.Files, cwd)
	res.Files = append(res.Files, resFiles...)

	if param.Tree {
		var otherTopVols []NewVolume
		var vids []string
		for vid := range connector.Vols {
			if vid != connector.GetVolId(vol) {
				otherTopVols = append(otherTopVols, connector.Vols[vid])
				vids = append(vids, vid)
			}
		}
		for i := range otherTopVols {
			vid := vids[i]
			cvol := otherTopVols[i]
			cvolS, _ := fs.Stat(cvol, "")
			cwdItem, err3 := CreateFileInfo(vid, cvol, fmt.Sprintf("/%s", vol.Name()), cvolS)
			if err3 != nil {
				log.Print(err3)
				if jsonErr := SendJson(rw, NewErr(err2)); jsonErr != nil {
					log.Print(jsonErr)
				}
				return
			}
			res.Files = append(res.Files, cwdItem)
		}
	}
	if path == "" {
		opt := NewDefaultOption()
		opt.Path = res.Cwd.Name
		res.Options = opt
		res.Cwd.Options = &opt
	}
	if err := SendJson(rw, &res); err != nil {
		log.Print(err)
	}
}
