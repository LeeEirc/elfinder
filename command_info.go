package elfinder

import (
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
)

func InfoCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	targets := req.Form["targets[]"]
	log.Print(targets)
	var resp InfoResponse
	for _, target := range targets {
		id, path, err := connector.GetVolByTarget(target)
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
