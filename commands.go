package elfinder

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

const defaultMaxMemory = 32 << 20

const (
	cmdOpen    = "open"
	cmdInfo    = "info"
	cmdParents = "parents"
	cmdTree    = "tree"
	cmdLs      = "ls"
	cmdUpload  = "upload"
	cmdRm      = "rm"
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
		cmdParents: ParentsCommand,
		cmdTree:    TreeCommand,
		cmdLs:      LsCommand,
		cmdUpload:  UploadCommand,
		cmdRm:      RmCommand,
	}
)

func DecodeTarget(target string) (id, path string, err error) {
	ret := strings.SplitN(target, "_", 2)
	if len(ret) != 2 {
		return "", "", ErrValidTarget
	}
	path, err = base64Decode(ret[1])
	return ret[0], path, err
}

func EncodeTarget(id, path string) string {
	return strings.Join([]string{id, base64Encode(path)}, "_")
}

func base64Encode(path string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(path))
}

func base64Decode(hashPath string) (string, error) {
	path, err := base64.RawURLEncoding.DecodeString(hashPath)
	return string(path), err
}

var (
	errNoFoundCmd  = errors.New("no found cmd")
	ErrNoFoundVol  = errors.New("no found volume")
	ErrValidTarget = errors.New("no valid target")
)

func parseCommand(req *http.Request) (string, error) {
	var cmd string
	switch req.Method {
	case http.MethodGet:
		cmd = req.URL.Query().Get("cmd")
	case http.MethodPost:
		cmd = req.Form.Get("cmd")
	}
	if cmd == "" {
		return "", fmt.Errorf("%w: %s", errNoFoundCmd, req.URL)
	}
	return cmd, nil
}

type CommandHandler func(connector *Connector, req *http.Request, rw http.ResponseWriter)

type RequestFormParseFunc func(req *http.Request) error

func SendJson(w http.ResponseWriter, data interface{}) error {
	w.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	return json.NewEncoder(w).Encode(data)
}

func StatFsVolFileByPath(id string, vol FsVolume, path string) (FileInfo, error) {
	pathHash := EncodeTarget(id, path)
	parentPath := filepath.Dir(path)
	parentPathHash := EncodeTarget(id, parentPath)
	isRoot := 0
	volRootPath := fmt.Sprintf("/%s", vol.Name())
	if path == volRootPath {
		isRoot = 1
		parentPathHash = ""
	}
	relativePath := strings.TrimPrefix(strings.TrimPrefix(path, volRootPath), Separator)

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

	var locked int
	r, w := ParseFileMode(info.Mode())
	if w == 0 {
		locked = 1
	}
	return FileInfo{
		Name:       name,
		PathHash:   pathHash,
		ParentHash: parentPathHash,
		MimeType:   MimeType,
		Timestamp:  info.ModTime().Unix(),
		Size:       info.Size(),
		HasDirs:    HasDirs,
		ReadAble:   r,
		WriteAble:  w,
		Locked:     locked,
		Volumeid:   Volumeid,
		Isroot:     isRoot,
	}, nil
}

func ReadFsVolDir(id string, vol FsVolume, path string) ([]FileInfo, error) {
	volRootPath := fmt.Sprintf("/%s", vol.Name())
	dirPath := strings.TrimPrefix(strings.TrimPrefix(path, volRootPath), "/")
	if dirPath == "" {
		dirPath = "."
	}
	files, err := fs.ReadDir(vol, dirPath)
	if err != nil {
		return nil, err
	}
	var res []FileInfo

	for i := range files {
		subPath := strings.Join([]string{path, files[i].Name()}, Separator)
		info, err2 := StatFsVolFileByPath(id, vol, subPath)
		if err2 != nil {
			return nil, err2
		}
		res = append(res, info)
	}

	return res, nil
}

func NewErr(errType ErrType, errs ...error) (respErr ErrResponse) {
	respErr.Type = errType
	respErr.Errs = errs
	return
}

type ErrResponse struct {
	Type ErrType
	Errs []error
}

func (e ErrResponse) MarshalJSON() ([]byte, error) {
	errs := make([]string, 0, len(e.Errs)+1)
	errs = append(errs, string(e.Type))
	for i := range e.Errs {
		errs = append(errs, e.Errs[i].Error())
	}
	data := map[string]interface{}{
		"error": errs,
	}
	return json.Marshal(data)
}
