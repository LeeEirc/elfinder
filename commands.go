package elfinder

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
)

const defaultMaxMemory = 32 << 20

const (
	cmdOpen = "open"
	cmdInfo = "info"
)

var (
	GetFormParse = func(req *http.Request) error {
		return req.ParseForm()
	}
	PostFormParse = func(req *http.Request) error {
		return req.ParseMultipartForm(defaultMaxMemory)
	}
)
var (
	supportedMethods = map[string]RequestFormParseFunc{
		http.MethodGet:  GetFormParse,
		http.MethodPost: PostFormParse,
	}

	supportedCommands = map[string]CommandHandler{
		cmdOpen: OpenCommand,
		cmdInfo: InfoCommand,
	}
)

type NewVolume interface {
	fs.FS
	fs.FileInfo
}

func NewConnector(vols ...NewVolume) *Connector {
	letter := "l"
	volsMap := make(map[string]NewVolume, len(vols))
	for i := range vols {
		vid := fmt.Sprintf("%s%d", letter, i)
		volsMap[vid] = vols[i]
	}
	var defaultVol NewVolume
	if len(vols) > 0 {
		defaultVol = vols[0]
	}
	return &Connector{
		defaultVol: defaultVol,
		vols:       volsMap,
	}
}

type Connector struct {
	defaultVol NewVolume
	vols       map[string]NewVolume
}

func (c *Connector) GetVolId(v NewVolume) string {
	for id, vol := range c.vols {
		if vol == v {
			return id
		}
	}
	return ""
}

func (c *Connector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	formParseFunc, ok := supportedMethods[r.Method]
	if !ok {
		msg := fmt.Sprintf("%s Method not allowed", r.Method)
		http.Error(w, msg, http.StatusMethodNotAllowed)
		return
	}
	if err := formParseFunc(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cmd, err := parseCommand(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Print("cmd: ", cmd)
	handleFunc, ok := supportedCommands[cmd]
	if !ok {
		msg := fmt.Sprintf("not supported cmd %s", cmd)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	handleFunc(c, r, w)
}

func (c *Connector) getVolByTarget(target string) (string, string, error) {
	id, path, err := parseTarget(target)
	if err != nil {
		return "", "", err
	}
	return id, path, err
}

func parseTarget(target string) (id, path string, err error) {
	ret := strings.SplitN(target, "_", 2)
	if len(ret) == 2 {
		id = ret[0]
		hpath := ret[1]
		path, err = DecodePath(hpath)
		return id, path, err
	}
	return "", "", errValidTarget
}

func hashPath(id, path string) string {
	return id + "_" + Encode64(path)
}

func EncodePath(path string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(path))
}

func DecodePath(hashPath string) (string, error) {
	path, err := base64.RawURLEncoding.DecodeString(hashPath)
	if err != nil {
		return "", err
	}
	return string(path), nil
}

var (
	errNoFoundCmd  = errors.New("no found cmd")
	errNoFoundVol  = errors.New("no found volume")
	errValidTarget = errors.New("no valid target")
)

func parseCommand(req *http.Request) (string, error) {
	var cmd string
	if cmd = req.URL.Query().Get("cmd"); cmd == "" {
		return "", errNoFoundCmd
	}
	return cmd, nil
}

type CommandHandler func(connector *Connector, req *http.Request, rw http.ResponseWriter)

type RequestFormParseFunc func(req *http.Request) error

func CmdTree(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdLs(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdFile(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdParents(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdDir(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func InfoCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	targets := req.Form["targets[]"]
	log.Print(targets)
	var resp InfoResponse
	for _, target := range targets {
		id, path, err := connector.getVolByTarget(target)
		if err != nil {
			log.Print(err)
			if err != nil {
				if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
					log.Print(jsonErr)
				}
				return
			}
		}
		if vol := connector.vols[id]; vol != nil {
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

type InfoResponse struct {
	Files []FileInfo `json:"files"`
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomStr(n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func OpenCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	init := req.FormValue("init")
	target := req.FormValue("target")
	tree := req.FormValue("tree")

	log.Println("  ", init, "  ", tree, "  ", target)

	var res OpenResponse

	if init == "1" {
		res.Api = APIVERSION
	}
	res.UplMaxSize = "32M"
	var (
		id     string
		path   string
		err    error
		vol    NewVolume
		fsinfo fs.FileInfo
	)
	if target == "" {
		vol = connector.defaultVol
		id = connector.GetVolId(connector.defaultVol)
		//path = "/"
		fsinfo = vol
	} else {
		id, path, err = connector.getVolByTarget(target)
		log.Print(err)
		if err != nil {
			if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
				log.Print(jsonErr)
			}
			return
		}
		vol = connector.vols[id]
		path = strings.TrimPrefix(path, "/")
		cmdInfo, err := fs.Stat(vol, path)
		if err != nil {
			log.Print(err)
			if jsonErr := SendJson(rw, NewErr(errNoFoundVol)); jsonErr != nil {
				log.Print(jsonErr)
			}
			return
		}
		fsinfo = cmdInfo
	}
	if vol == nil || id == "" {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(errNoFoundVol)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}



	cwd, err2 := CreateFileInfo(id, vol, path, fsinfo)
	if err2 != nil {
		log.Print(err2)
		if jsonErr := SendJson(rw, NewErr(err2)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	res.Cwd = cwd
	res.Files = append(res.Files, cwd)
	if tree == "1" {
		var otherTopVols []NewVolume
		var vids []string
		for vid := range connector.vols {
			if vid != connector.GetVolId(vol) {
				otherTopVols = append(otherTopVols, connector.vols[vid])
				vids = append(vids, vid)
			}
		}
		for i := range otherTopVols {
			vid := vids[i]
			cvol := otherTopVols[i]
			cwdItem, err3 := CreateFileInfo(vid, cvol, "", cvol)
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
	log.Printf("%+v\n", res)
	if err := SendJson(rw, &res); err != nil {
		log.Print(err)
	}
}

func SendJson(w http.ResponseWriter, data interface{}) error {
	w.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	return json.NewEncoder(w).Encode(data)
}

func CreateFileInfo(id string, vol NewVolume, path string, cmdInfo fs.FileInfo) (FileInfo, error) {
	var (
		pathHash   string
		parentHash string
		MimeType   string
		HasDirs    int
		isRoot     int
	)
	parentPath := filepath.Dir(path)
	pathHash = hashPath(id, path)
	if path != "" && path != "/" {
		parentHash = hashPath(id, parentPath)
	} else {
		isRoot = 1
	}
	MimeType = "file"
	if cmdInfo.IsDir() {
		MimeType = "directory"
		dirItems, err2 := fs.ReadDir(vol, path)
		if err2 != nil {
			return FileInfo{}, err2
		}
		for i := range dirItems {
			if dirItems[i].IsDir() {
				HasDirs = 1
				break
			}
		}
	}
	return FileInfo{
		Name:       cmdInfo.Name(),
		PathHash:   pathHash,
		ParentHash: parentHash,
		MimeType:   MimeType,
		Timestamp:  cmdInfo.ModTime().Unix(),
		Size:       cmdInfo.Size(),
		HasDirs:    HasDirs,
		ReadAble:   1,
		WriteAble:  1,
		Locked:     0,
		Volumeid:   id + "_",
		Isroot:     isRoot,
	}, nil
}

type OpenRequest struct {
	Init   string `query:"init"` //  (true|false|not set)
	Target string `query:"target"`
	Tree   bool   `query:"tree"`
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

type DebugOption struct {
	Connector string        `json:"connector"`
	Time      float64       `json:"time"`
	Memory    string        `json:"memory"`
	Volumes   []interface{} `json:"volumes"`
}

type ErrResponse map[string]interface{}

func NewErr(errs ...error) ErrResponse {
	errResp := make(ErrResponse)
	switch len(errs) {
	case 1:
		errResp["err"] = errs[0]
	default:
		errResp["err"] = errs
	}
	return errResp
}

/*
 options : {
   "path"            : "files/folder42",                        // (String) Current folder path
   "url"             : "http://localhost/elfinder/files/",      // (String) Current folder URL
   "tmbURL"          : "http://localhost/elfinder/files/.tmb/", // (String) Thumbnails folder URL
   "separator"       : "/",                                     // (String) Path separator for the current volume
   "disabled"        : [],                                      // (Array)  List of commands not allowed (disabled) on this volume
   "copyOverwrite"   : 1,                                       // (Number) Whether or not to overwrite files with the same name on the current volume when copy
   "uploadOverwrite" : 1,                                       // (Number) Whether or not to overwrite files with the same name on the current volume when upload
   "uploadMaxSize"   : 1073741824,                              // (Number) Upload max file size per file
   "uploadMaxConn"   : 3,                                       // (Number) Maximum number of chunked upload connection. `-1` to disable chunked upload
   "uploadMime": {                                              // (Object) MIME type checker for upload
       "allow": [ "image", "text/plain" ],                      // (Array) Allowed MIME type
       "deny": [ "all" ],                                       // (Array) Denied MIME type
       "firstOrder": "deny"                                     // (String) First order to check ("deny" or "allow")
   },
   "dispInlineRegex" : "^(?:image|text/plain$)",                // (String) Regular expression of MIME types that can be displayed inline with the `file` command
   "jpgQuality"      : 100,                                     // (Number) JPEG quality to image resize / crop / rotate (1-100)
   "syncChkAsTs"     : 1,                                       // (Number) Whether or not to current volume can detect update by the time stamp of the directory
   "syncMinMs"       : 30000,                                   // (Number) Minimum inteval Milliseconds for auto sync
   "uiCmdMap"        : { "chmod" : "perm" },                    // (Object) Command conversion map for the current volume (e.g. chmod(ui) to perm(connector))
   "i18nFolderName"  : 1,                                       // (Number) Is enabled i18n folder name that convert name to elFinderInstance.messages['folder_'+name]
   "archivers"       : {                                        // (Object) Archive settings
     "create"  : [
       "application/zip",
       "application/x-tar",
       "application/x-gzip"
     ],                                                   // (Array)  List of the mime type of archives which can be created
     "extract" : [
       "application/zip",
       "application/x-tar",
       "application/x-gzip"
     ],                                                   // (Array)  List of the mime types that can be extracted / unpacked
     "createext": {
       "application/zip": "zip",
       "application/x-tar": "tar",
       "application/x-gzip": "tgz"
     }                                                    // (Object)  Map list of { MimeType: FileExtention }
   }
 }
*/

type Option struct {
	Path            string            `json:"path"`
	URL             string            `json:"url"`
	TmbURL          string            `json:"tmbURL"`
	Separator       string            `json:"separator"`
	Disabled        []string          `json:"disabled"`
	CopyOverwrite   int               `json:"copyOverwrite"`
	UploadOverwrite int               `json:"uploadOverwrite"`
	UploadMaxSize   int               `json:"uploadMaxSize"`
	UploadMaxConn   int               `json:"uploadMaxConn"`
	UploadMime      UploadMimeOption  `json:"uploadMime"`
	DispInlineRegex string            `json:"dispInlineRegex"`
	JpgQuality      int               `json:"jpgQuality"`
	SyncChkAsTs     int               `json:"syncChkAsTs"`
	SyncMinMs       int               `json:"syncMinMs"`
	UiCmdMap        map[string]string `json:"uiCmdMap"`
	I18nFolderName  int               `json:"i18nFolderName"`
	Archivers       ArchiverOption    `json:"archivers"`
}

type ArchiverOption struct {
	Create    []string          `json:"create"`
	Extract   []string          `json:"extract"`
	Createext map[string]string `json:"createext"`
}

type UploadMimeOption struct {
	Allow []string `json:"allow"`

	Deny []string `json:"deny"`

	FirstOrder string `json:"firstOrder"`
}

var (
	defaultArchivers = ArchiverOption{
		Create:    createArray,
		Extract:   extractArray,
		Createext: createextMap,
	}
	createextMap = map[string]string{
		"application/zip":    "zip",
		"application/x-tar":  "tar",
		"application/x-gzip": "tgz",
	}
	extractArray = []string{
		"application/zip",
		"application/x-tar",
		"application/x-gzip",
	}
	createArray = []string{
		"application/zip",
		"application/x-tar",
		"application/x-gzip",
	}
)

func NewDefaultOption() Option {
	return Option{
		Separator: Separator,
		Archivers: defaultArchivers,
	}
}

const Separator = "/"
