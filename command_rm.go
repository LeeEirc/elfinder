package elfinder

import (
	"fmt"
	"net/http"
	"strings"
)

type RmRequest struct {
	ReqId   string   `elfinder:"reqid"`
	Targets []string `elfinder:"targets[]"`
}

type RmResponse struct {
	Removed []string `json:"removed"`
}

func RmCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var (
		cmdReq      RmRequest
		cmdResponse RmResponse
	)
	err := UnmarshalElfinderTag(&cmdReq, req.URL.Query())
	if err != nil {
		connector.Logger.Error(err)
		if jsonErr := SendJson(rw, NewErr(ERRCmdReq, err)); jsonErr != nil {
			connector.Logger.Error(jsonErr)
		}
		return
	}
	for i := range cmdReq.Targets {
		target := cmdReq.Targets[i]
		id, path, err := connector.parseTarget(target)
		if err != nil {
			connector.Logger.Error(err)
			if jsonErr := SendJson(rw, NewErr(ERRCmdReq, err)); jsonErr != nil {
				connector.Logger.Error(jsonErr)
			}
			return
		}
		vol, ok := connector.Vols[id]
		if !ok {
			connector.Logger.Errorf("not found vol by id: %s", id)
			if err = SendJson(rw, NewErr(ERROpen, ErrNoFoundVol)); err != nil {
				connector.Logger.Errorf("send response json err: %s", err)
			}
			return
		}
		cwdInfo, err := StatFsVolFileByPath(id, vol, path)
		if err != nil {
			connector.Logger.Error(err)
			return
		}
		relativePath := strings.TrimPrefix(path, fmt.Sprintf("/%s/", vol.Name()))
		if err := vol.Remove(relativePath); err != nil {
			connector.Logger.Error(err)
			if jsonErr := SendJson(rw, NewErr(ERRRm, err)); jsonErr != nil {
				connector.Logger.Error(jsonErr)
			}
			return
		}
		cmdResponse.Removed = append(cmdResponse.Removed, cwdInfo.PathHash)
	}
	if err := SendJson(rw, &cmdResponse); err != nil {
		connector.Logger.Error(err)
	}

}
