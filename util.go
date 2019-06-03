package elfinder

import (
	"encoding/base64"
	"crypto/md5"
	"encoding/hex"
	"os"
)

func Decode64(s string) (string, error) {
	t, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(t), nil
}

func Encode64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func CreateHash(volumeId, path string) string {
	return volumeId + "_" + Encode64(path)
}

func GenerateID(path string) string {
	ctx := md5.New()
	ctx.Write([]byte(path))
	return hex.EncodeToString(ctx.Sum(nil))
}

func ReadWritePem(pem os.FileMode) (readable, writeable byte) {
	if pem&(1<<uint(9-1-0)) != 0 {
		readable = 1
	}
	if pem&(1<<uint(9-1-1)) != 0 {
		writeable = 1
	}
	return
}


var damo =`
{
    "api":"2.1",
    "cwd":{
        "dirs":1,
        "hash":"a5f40d93b2831f5c0e979b210462c00e_Lw",
        "hidden":0,
        "locked":1,
        "mime":"directory",
        "name":"Home",
        "read":1,
        "size":832,
        "ts":1558928508.1905725,
        "volume_id":"a5f40d93b2831f5c0e979b210462c00e",
        "write":1
    },
    "files":[
        {
            "dirs":1,
            "hash":"a5f40d93b2831f5c0e979b210462c00e_MTExLjIzMC4yMDYuNTE",
            "hidden":0,
            "locked":0,
            "mime":"directory",
            "name":"111.230.206.51",
            "phash":"a5f40d93b2831f5c0e979b210462c00e_Lw",
            "read":1,
            "size":832,
            "ts":1558928508.1905725,
            "write":1
        },
        {
            "dirs":1,
            "hash":"a5f40d93b2831f5c0e979b210462c00e_Lw",
            "hidden":0,
            "locked":1,
            "mime":"directory",
            "name":"Home",
            "read":1,
            "size":832,
            "ts":1558928508.1905725,
            "volume_id":"a5f40d93b2831f5c0e979b210462c00e",
            "write":1
        }
    ],
    "options":{
        "archivers":{
            "create":[

            ],
            "extract":[

            ]
        },
        "copyOverwrite":1,
        "disabled":[
            "chmod"
        ],
        "separator":"/",
        "uiCmdMap":[

        ]
    },
    "uplMaxSize":"10M"
}
`
