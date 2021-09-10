package elfinder

import "net/http"

type CommandFunc func(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter)

func CmdOpen(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdTree(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdLs(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdFile(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdParents(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}
