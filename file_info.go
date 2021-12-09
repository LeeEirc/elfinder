package elfinder

import "os"

type FileInfo struct {
	Name       string  `json:"name"`
	PathHash   string  `json:"hash"`
	ParentHash string  `json:"phash"`
	MimeType   string  `json:"mime"` // Required
	Timestamp  int64   `json:"ts"`   // unix timestamp  Required
	Size       int64   `json:"size"` //file size in bytes
	HasDirs    int     `json:"dirs"` // Only for directories. Marks if directory has child directories inside it. 0 (or not set)
	ReadAble   int     `json:"read"`
	WriteAble  int     `json:"write"`
	Locked     int     `json:"locked"`
	TmbImage   string  `json:"tmb"`
	AliasName  string  `json:"alias"` // symlinks only. Symlink target path.
	Thash      string  `json:"thash"` //  For symlinks only. Symlink target hash.
	Dim        string  `json:"dim"`
	IsOwner    bool    `json:"isowner"` // has ownership. Optionally.
	Csscls     string  `json:"csscls"`
	Volumeid   string  `json:"volumeid"` // Volume id. For directory only.
	Options    *Option `json:"options,omitempty"`
	Isroot     int     `json:"isroot"`
}

func NewFileInfo(vid string, vol NewVolume, info os.FileInfo) {

}

/*

{
    "name"   : "Images",             // (String) name of file/dir. Required
    "hash"   : "l0_SW1hZ2Vz",        // (String) hash of current file/dir path, first symbol must be letter, symbols before _underline_ - volume id, Required.
    "phash"  : "l0_Lw",              // (String) hash of parent directory. Required except roots dirs.
    "mime"   : "directory",          // (String) mime type. Required.
    "ts"     : 1334163643,           // (Number) file modification time in unix timestamp. Required.
    "date"   : "30 Jan 2010 14:25",  // (String) last modification time (mime). Depricated but yet supported. Use ts instead.
    "size"   : 12345,                // (Number) file size in bytes
    "dirs"   : 1,                    // (Number) Only for directories. Marks if directory has child directories inside it. 0 (or not set) - no, 1 - yes. Do not need to calculate amount.
    "read"   : 1,                    // (Number) is readable
    "write"  : 1,                    // (Number) is writable
    "locked" : 0,                    // (Number) is file locked. If locked that object cannot be deleted,  renamed or moved
    "tmb"    : 'bac0d45b625f8d4633435ffbd52ca495.png' // (String) Only for images. Thumbnail file name, if file do not have thumbnail yet, but it can be generated than it must have value "1"
    "alias"  : "files/images",       // (String) For symlinks only. Symlink target path.
    "thash"  : "l1_c2NhbnMy",        // (String) For symlinks only. Symlink target hash.
    "dim"    : "640x480",            // (String) For images - file dimensions. Optionally.
    "isowner": true,                 // (Bool) has ownership. Optionally.
    "csscls" : "custom-icon",        // (String) CSS class name for holder icon. Optionally. It can include to options.
    "volumeid" : "l1_",              // (String) Volume id. For directory only. It can include to options.
    "netkey" : "",                   // (String) Netmount volume unique key, Required for netmount volume. It can include to options.
    "options" : {}                   // (Object) For volume root only. This value is same to cwd.options.
}

*/
