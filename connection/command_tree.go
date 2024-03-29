package connection

import (
	"fmt"
	"log"
	"net/http"

	"github.com/LeeEirc/elfinder/codecs"
	"github.com/LeeEirc/elfinder/errs"
)

type TreeRequest struct {
	Target string `elfinder:"target"`
	ReqId  string `elfinder:"reqid"`
}

func TreeCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var param TreeRequest

	if err := codecs.UnmarshalElfinderTag(&param, req.URL.Query()); err != nil {
		log.Print(err)
		return
	}
	id, path, err := connector.ParseTarget(param.Target)
	if err != nil {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(errs.ERRCmdParams, err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	fmt.Println(id, path)
	vol := connector.Vols[id]
	var res ParentsResponse
	cwdInfo, err := ReadFsVolDir(id, vol, path)
	if err != nil {
		log.Panicln(err)
		return
	}
	res.Tree = append(res.Tree, cwdInfo...)
	if err := SendJson(rw, &res); err != nil {
		log.Print(err)
	}
}
