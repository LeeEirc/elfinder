package elfinder

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func NewConnector(opts ...Options) *Connector {
	opt := option{
		Logger: os.Stdout,
	}
	for _, setter := range opts {
		setter(&opt)
	}

	volsMap := make(map[string]NewVolume, len(opt.Vols))
	for i := range opt.Vols {
		vid := MD5ID(opt.Vols[i].Name())
		volsMap[vid] = opt.Vols[i]
	}
	var defaultVol NewVolume
	if len(opt.Vols) > 0 {
		defaultVol = opt.Vols[0]
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
	Logger     io.Writer
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
		var data = ElfinderErr{
			Errs: []string{errConnect, r.Method},
		}
		if err := SendJson(w, data); err != nil {
			log.Println(err)
		}
		return
	}
	if err := formParseFunc(r); err != nil {
		var data = ElfinderErr{
			Errs: []string{errCmdParams, err.Error()},
		}
		if err := SendJson(w, data); err != nil {
			log.Println(err)
		}
		return
	}
	cmd, err := parseCommand(r)
	if err != nil {
		var data = ElfinderErr{
			Errs: []string{errCmdParams, err.Error()},
		}
		if err := SendJson(w, data); err != nil {
			log.Println(err)
		}
		return
	}
	fmt.Println(r.URL.Query())
	handleFunc, ok := supportedCommands[cmd]
	if !ok {
		var data = ElfinderErr{
			Errs: []string{errCmdNoSupport, cmd},
		}
		if err := SendJson(w, data); err != nil {
			log.Println(err)
		}
		return
	}
	handleFunc(c, r, w)
}

func (c *Connector) GetVolByTarget(target string) (string, string, error) {
	vid, vPath, err := DecodeTarget(target)
	if err != nil {
		return "", "", err
	}
	return vid, vPath, err
}

type Options func(*option)

type option struct {
	Vols   []NewVolume
	Logger io.Writer
}

func WithVolumes(vols ...NewVolume) Options {
	return func(o *option) {
		o.Vols = vols
	}
}

func WithLogger(w io.Writer) Options {
	return func(o *option) {
		o.Logger = w
	}
}
