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
		cmdParents: ParentsCommand,
		cmdTree:    TreeCommand,

		cmdInfo: InfoCommand,
	}
)

type NewVolume interface {
	Name() string
	fs.FS
}

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
	if cmd = req.URL.Query().Get("cmd"); cmd == "" {
		return "", errNoFoundCmd
	}
	return cmd, nil
}

type CommandHandler func(connector *Connector, req *http.Request, rw http.ResponseWriter)

type RequestFormParseFunc func(req *http.Request) error

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
	pathHash = EncodeTarget(id, path)
	if path != "" && path != "/" {
		parentHash = EncodeTarget(id, parentPath)
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
	pathHash := EncodeTarget(id, path)
	parentPath := filepath.Dir(path)
	parentPathHash := EncodeTarget(id, parentPath)
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

type ElfinderErr struct {
	Errs interface{} `json:"error"`
}
