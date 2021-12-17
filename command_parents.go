package elfinder

import (
	"log"
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
		param  ParentsRequest
		res    ParentsResponse
		errRes ElfinderErr
	)

	err := UnmarshalElfinderTag(&param, req.URL.Query())
	if err != nil {
		errRes.Errs = []string{errCmdParams, err.Error()}
		if jsonErr := SendJson(rw, &errRes); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	target := param.Target
	id, path, err := connector.parseTarget(target)
	if err != nil {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	vol := connector.Vols[id]
	cwdinfo, err := StatFsVolFileByPath(id, vol, path)
	if err != nil {
		log.Panicln(err)
		return
	}
	res.Tree = append(res.Tree, cwdinfo)
	for path != "/" {
		path = filepath.Dir(path)
		cwdinfo, err = StatFsVolFileByPath(id, vol, path)
		if err != nil {
			log.Panicln(err)
			return
		}
		res.Tree = append(res.Tree, cwdinfo)

		cwdDirs, err := ReadFsVolDir(id, vol, path)
		if err != nil {
			log.Panicln(err)
			return
		}
		res.Tree = append(res.Tree, cwdDirs...)
	}

	if err := SendJson(rw, &res); err != nil {
		log.Print(err)
	}

}
