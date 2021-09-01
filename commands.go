package elfinder

import "net/http"

type CommandFunc func(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter)

func CmdOpen(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdTree(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}


func Cmdls(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter){

}