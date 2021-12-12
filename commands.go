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
	"time"
)

const defaultMaxMemory = 32 << 20

const (
	cmdOpen    = "open"
	cmdInfo    = "info"
	cmdParents = "parents"
	cmdTree    = "tree"
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
		cmdOpen:    OpenCommand,
		cmdInfo:    InfoCommand,
		cmdParents: ParentsCommand,
		cmdTree:    TreeCommand,
	}
)

type NewVolume interface {
	Name() string
	fs.FS
}

func NewConnector(vols ...NewVolume) *Connector {
	volsMap := make(map[string]NewVolume, len(vols))
	for i := range vols {
		vid := MD5ID(vols[i].Name())
		volsMap[vid] = vols[i]
	}
	var defaultVol NewVolume
	if len(vols) > 0 {
		defaultVol = vols[0]
	}
	return &Connector{
		defaultVol: defaultVol,
		vols:       volsMap,
		Created:    time.Now(),
	}
}

type Connector struct {
	defaultVol NewVolume
	vols       map[string]NewVolume
	Created    time.Time
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
	fmt.Println(r.URL.Query())
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
	return id + "_" + EncodePath(strings.TrimPrefix(path, Separator))
}

func EncodePath(path string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(path))
}

func DecodePath(hashPath string) (string, error) {
	path, err := base64.RawURLEncoding.DecodeString(hashPath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/%s", string(path)), nil
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

func CmdLs(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func CmdFile(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func ParentsCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	var param struct {
		Target string `param:"target"`
	}
	err := BindData(&param, req.URL.Query(), "param")
	if err != nil {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	target := param.Target
	id, path, err := connector.getVolByTarget(target)
	if err != nil {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	vol := connector.vols[id]
	var res ParentsResponse
	cwdinfo, err := CreateFileInfoByPath(id, vol, path)
	if err != nil {
		log.Panicln(err)
		return
	}
	res.Tree = append(res.Tree, cwdinfo)
	for path != "/" {
		path = filepath.Dir(path)
		cwdinfo, err = CreateFileInfoByPath(id, vol, path)
		if err != nil {
			log.Panicln(err)
			return
		}
		res.Tree = append(res.Tree, cwdinfo)

		cwdDirs, err := ReadFilesByPath(id, vol, path)
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

func CmdDir(elf *ElFinderConnector, req *http.Request, rw http.ResponseWriter) {

}

func TreeCommand(connector *Connector, req *http.Request, rw http.ResponseWriter) {
	target := req.URL.Query().Get("target")
	id, path, err := connector.getVolByTarget(target)
	if err != nil {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	fmt.Println(id, path)
	vol := connector.vols[id]
	var res ParentsResponse
	cwdinfo, err := ReadFilesByPath(id, vol, path)
	if err != nil {
		log.Panicln(err)
		return
	}
	res.Tree = append(res.Tree, cwdinfo...)
	if err := SendJson(rw, &res); err != nil {
		log.Print(err)
	}
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

	log.Println(req.URL.Query())

	var res OpenResponse

	if init == "1" {
		res.Api = APIVERSION
	}
	res.UplMaxSize = "32M"
	var (
		id   string
		path string
		err  error
		vol  NewVolume
	)
	if target == "" {
		vol = connector.defaultVol
		id = connector.GetVolId(connector.defaultVol)
		path = fmt.Sprintf("/%s", vol.Name())
	} else {
		id, path, err = connector.getVolByTarget(target)
		if err != nil {
			log.Print(err)
			if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
				log.Print(jsonErr)
			}
			return
		}
		vol = connector.vols[id]
	}
	if vol == nil || id == "" {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(errNoFoundVol)); jsonErr != nil {
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

func SendJson(w http.ResponseWriter, data interface{}) error {
	w.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	return json.NewEncoder(w).Encode(data)
}

func CreateFileInfo(id string, vol NewVolume, path string, fsInfo fs.FileInfo) (FileInfo, error) {
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
	if fsInfo.IsDir() {
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
		Name:       fsInfo.Name(),
		PathHash:   pathHash,
		ParentHash: parentHash,
		MimeType:   MimeType,
		Timestamp:  fsInfo.ModTime().Unix(),
		Size:       fsInfo.Size(),
		HasDirs:    HasDirs,
		ReadAble:   1,
		WriteAble:  1,
		Locked:     0,
		Volumeid:   id + "_",
		Isroot:     isRoot,
	}, nil
}

func CreateFileInfoByPath(id string, vol NewVolume, path string) (FileInfo, error) {
	pathHash := hashPath(id, path)
	parentPath := filepath.Dir(path)
	parentPathHash := hashPath(id, parentPath)
	isRoot := 0
	volRootPath := fmt.Sprintf("/%s", vol.Name())
	if path == volRootPath {
		isRoot = 1
		parentPathHash = ""
	}
	relativePath := strings.TrimPrefix(strings.TrimPrefix(path, volRootPath), "/")

	var name string
	if relativePath == "" {
		relativePath = "."
		name = vol.Name()
	}

	info, err := fs.Stat(vol, relativePath)
	if err != nil {
		return FileInfo{}, err
	}
	if name == "" {
		name = info.Name()
	}

	MimeType := "file"
	HasDirs := 0
	if info.IsDir() {
		MimeType = "directory"
		HasDirs = 1
	}
	Volumeid := ""
	if HasDirs == 1 {
		Volumeid = id + "_"
	}

	return FileInfo{
		Name:       name,
		PathHash:   pathHash,
		ParentHash: parentPathHash,
		MimeType:   MimeType,
		Timestamp:  info.ModTime().Unix(),
		Size:       info.Size(),
		HasDirs:    HasDirs,
		ReadAble:   1,
		WriteAble:  1,
		Locked:     0,
		Volumeid:   Volumeid,
		Isroot:     isRoot,
	}, nil
}

func ReadFilesByPath(id string, vol NewVolume, path string) ([]FileInfo, error) {
	volRootPath := fmt.Sprintf("/%s", vol.Name())
	dirPath := strings.TrimPrefix(strings.TrimPrefix(path, volRootPath), "/")
	if dirPath == "" {
		dirPath = "."
	}
	files, err := fs.ReadDir(vol, dirPath)
	if err != nil {
		log.Println("fs.ReadDir ", err)
		return nil, err
	}

	var res []FileInfo

	for i := range files {
		subPath := filepath.Join(path, files[i].Name())
		info, err := CreateFileInfoByPath(id, vol, subPath)
		if err != nil {
			log.Println("CreateFileInfoByPath ", err, subPath)
			return nil, err
		}
		res = append(res, info)
	}

	return res, nil

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

type ParentsResponse struct {
	Tree []FileInfo `json:"tree"`
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
