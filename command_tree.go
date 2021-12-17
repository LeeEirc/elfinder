package elfinder

import (
	"fmt"
	"log"
	"net/http"
)

type TreeRequest struct {
	Target string `elfinder:"target"`
	ReqId  string `elfinder:"reqid"`
}

func TreeCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var param TreeRequest

	if err := UnmarshalElfinderTag(&param, req.URL.Query()); err != nil {
		log.Print(err)
		return
	}
	id, path, err := connector.parseTarget(param.Target)
	if err != nil {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	fmt.Println(id, path)
	vol := connector.Vols[id]
	var res ParentsResponse
	cwdinfo, err := ReadFsVolDir(id, vol, path)
	if err != nil {
		log.Panicln(err)
		return
	}
	res.Tree = append(res.Tree, cwdinfo...)
	if err := SendJson(rw, &res); err != nil {
		log.Print(err)
	}
}
