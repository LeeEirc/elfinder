package connection

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/LeeEirc/elfinder/errs"
	"github.com/LeeEirc/elfinder/log"
	"github.com/LeeEirc/elfinder/utils"
	"github.com/LeeEirc/elfinder/volumes"
)

func NewConnector(opts ...Options) *Connector {
	opt := option{
		Logger: &log.GlobalLogger,
	}
	for _, setter := range opts {
		setter(&opt)
	}

	volsMap := make(map[string]volumes.FsVolume, len(opt.Vols))
	for i := range opt.Vols {
		vid := utils.MD5ID(opt.Vols[i].Name())
		volsMap[vid] = opt.Vols[i]
	}
	var defaultVol volumes.FsVolume
	if len(opt.Vols) > 0 {
		defaultVol = opt.Vols[0]
	}
	return &Connector{
		DefaultVol: defaultVol,
		Vols:       volsMap,
		Created:    time.Now(),
		Logger:     opt.Logger,
	}
}

type Connector struct {
	DefaultVol volumes.FsVolume
	Vols       map[string]volumes.FsVolume
	Created    time.Time
	Logger     log.Logger
	mux        sync.Mutex
}

func (c *Connector) GetVolId(v volumes.FsVolume) string {
	for id, vol := range c.Vols {
		if vol.Name() == v.Name() {
			return id
		}
	}
	return ""
}
func (c *Connector) GetFsById(id string) volumes.FsVolume {
	return c.Vols[id]
}

func (c *Connector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	formParseFunc, ok := supportedMethods[r.Method]
	if !ok {
		c.Logger.Errorf("not support http method %s", r.Method)
		if err := SendJson(w, NewErr(errs.ERRCmdParams, fmt.Errorf("method: %s", r.Method))); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	if err := formParseFunc(r); err != nil {
		c.Logger.Errorf("HTTP form parse errs: %s", err)
		if err := SendJson(w, NewErr(errs.ERRCmdParams, err)); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	cmd, err := parseCommand(r)
	if err != nil {
		c.Logger.Errorf("Parse command errs: %s %+v", err, r.URL.Query())
		if err := SendJson(w, NewErr(errs.ERRCmdParams, err)); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	handleFunc, ok := supportedCommands[cmd]
	if !ok {
		c.Logger.Errorf("Command `%s` not supported", cmd)
		err = fmt.Errorf("command `%s` not supported", cmd)
		if err := SendJson(w, NewErr(errs.ERRUsupportType, err)); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	handleFunc(c, r, w)
}

func (c *Connector) ParseTarget(target string) (vid, vPath string, err error) {
	return DecodeTarget(target)
}

type Options func(*option)

type option struct {
	Vols   []volumes.FsVolume
	Logger log.Logger
}

func WithVolumes(vols ...volumes.FsVolume) Options {
	return func(o *option) {
		o.Vols = vols
	}
}

func WithLogger(logger log.Logger) Options {
	return func(o *option) {
		o.Logger = logger
	}
}
