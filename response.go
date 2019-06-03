package elfinder

// source code from https://github.com/Supme/goElFinder/blob/master/types.go

var defaultOptions = options{
	Separator:     "/",
	Archivers:     archivers{Create: []string{}, Extract: []string{}},
	CopyOverwrite: 1}

type ElfResponse struct {
	Api        string    `json:"api,omitempty"`        // The version number of the protocol, must be >= 2.1, ATTENTION - return api ONLY for init request!
	Cwd        FileDir   `json:"cwd,omitempty"`        // Current Working Directory - information about the current directory. Information about File/Directory
	Files      []FileDir `json:"files"`                // array of objects - files and directories in current directory. If parameter tree == true, then added to the folder of the directory tree to a given depth. The order of files is not important. Note you must include the top-level volume objects here as well (i.e. cwd is repeated here, in addition to other volumes)
	NetDrivers []string  `json:"netDrivers,omitempty"` // Network protocols list which can be mounted on the fly (using netmount command). Now only ftp supported.
	Options    options   `json:"options,omitempty"`
	UplMaxFile string    `json:"uplMaxFile,omitempty"` // Allowed upload max number of file per request. For example 20
	UplMaxSize string    `json:"uplMaxSize,omitempty"` // Allowed upload max size per request. For example "32M"

	Tree []FileDir `json:"tree"` // for tree

	Dim string `json:"dim,omitempty"` // for images

	Added   []FileDir         `json:"added"`             // for upload, mkdir, rename
	Warning []string          `json:"warning,omitempty"` // for upload
	Changed []FileDir         `json:"changed,omitempty"` // for mkdir
	Hashes  map[string]string `json:"hashes,omitempty"`  // for mkdir
	List    []string          `json:"list,omitempty"`    // for ls

	Name        string `json:"_name,omitempty"`
	Chunkmerged string `json:"_chunkmerged,omitempty"`

	Removed []string `json:"removed,omitempty"` // for remove, rename

	Images map[string]string `json:"images,omitempty"` // for tmb

	Content string `json:"content,omitempty"` // for get

	Url string `json:"url,omitempty"` // for url

	Error interface{} `json:"error,omitempty"`
}

type options struct {
	Path          string    `json:"path,omitempty"`      // Current folder path
	Url           string    `json:"url,omitempty"`       // Current folder URL
	TmbUrl        string    `json:"tmbURL,omitempty"`    // Thumbnails folder URL
	Separator     string    `json:"separator,omitempty"` // Path separator for the current volume
	Disabled      []string  `json:"disabled,omitempty"`  // List of commands not allowed (disabled) on this volume
	Archivers     archivers `json:"archivers,omitempty"`
	CopyOverwrite int64     `json:"copyOverwrite,omitempty"` // (Number) Whether or not to overwrite files with the same name on the current volume when copy
	// ToDo https://github.com/Studio-42/elFinder/wiki/Client-Server-API-2.1#open

}

type archivers struct {
	Create    []string          `json:"create,omitempty"`    // List of the mime type of archives which can be created
	Extract   []string          `json:"extract,omitempty"`   // List of the mime types that can be extracted / unpacked
	Createext map[string]string `json:"createext,omitempty"` // Map list of { MimeType: FileExtention }
}

type FileDir struct {
	Name     string  `json:"name,omitempty"`  // name of file/dir. Required
	Hash     string  `json:"hash,omitempty"`  //  hash of current file/dir path, first symbol must be letter, symbols before _underline_ - volume id, Required.
	Phash    string  `json:"phash,omitempty"` // hash of parent directory. Required except roots dirs.
	Mime     string  `json:"mime,omitempty"`  // mime type. Required.
	Ts       int64   `json:"ts,omitempty"`    // file modification time in unix timestamp. Required.
	Size     int64   `json:"size,omitempty"`  // file size in bytes
	Dirs     byte    `json:"dirs,omitempty"`  // Only for directories. Marks if directory has child directories inside it. 0 (or not set) - no, 1 - yes. Do not need to calculate amount.
	Read     byte    `json:"read,omitempty"`  // is readable
	Write    byte    `json:"write,omitempty"` // is writable
	Isroot   byte    `json:"isroot,omitempty"`
	Locked   byte    `json:"locked,omitempty"`   // is file locked. If locked that object cannot be deleted, renamed or moved
	Tmb      string  `json:"tmb,omitempty"`      // Only for images. Thumbnail file name, if file do not have thumbnail yet, but it can be generated than it must have value "1"
	Alias    string  `json:"alias,omitempty"`    // For symlinks only. Symlink target path.
	Thash    string  `json:"thash,omitempty"`    // For symlinks only. Symlink target hash.
	Dim      string  `json:"dim,omitempty"`      // For images - file dimensions. Optionally.
	Isowner  bool    `json:"isowner,omitempty"`  // has ownership. Optionally.
	Cssclr   string  `json:"cssclr,omitempty"`   // CSS class name for holder icon. Optionally. It can include to options.
	Volumeid string  `json:"volumeid,omitempty"` // Volume id. For directory only. It can include to options.
	Netkey   string  `json:"netkey,omitempty"`   // Netmount volume unique key, Required for netmount volume. It can include to options.
	Options  options `json:"options,omitempty"`  // For volume root only. This value is same to cwd.options.
}
