package elfinder

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
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
		DefaultVol: defaultVol,
		Vols:       volsMap,
		Created:    time.Now(),
	}
}

type Connector struct {
	DefaultVol NewVolume
	Vols       map[string]NewVolume
	Created    time.Time
}

func (c *Connector) GetVolId(v NewVolume) string {
	for id, vol := range c.Vols {
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

func (c *Connector) GetVolByTarget(target string) (string, string, error) {
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
	ErrNoFoundVol  = errors.New("no found volume")
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
	id, path, err := connector.GetVolByTarget(target)
	if err != nil {
		log.Print(err)
		if jsonErr := SendJson(rw, NewErr(err)); jsonErr != nil {
			log.Print(jsonErr)
		}
		return
	}
	vol := connector.Vols[id]
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
	id, path, err := connector.GetVolByTarget(target)
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

type InfoResponse struct {
	Files []FileInfo `json:"files"`
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

type ParentsResponse struct {
	Tree []FileInfo `json:"tree"`
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
