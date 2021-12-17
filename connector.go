package elfinder

import (
	"net/http"
	"sync"
	"time"
)

func NewConnector(opts ...Options) *Connector {
	opt := option{
		Logger: &globalLogger,
	}
	for _, setter := range opts {
		setter(&opt)
	}

	volsMap := make(map[string]FsVolume, len(opt.Vols))
	for i := range opt.Vols {
		vid := MD5ID(opt.Vols[i].Name())
		volsMap[vid] = opt.Vols[i]
	}
	var defaultVol FsVolume
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
	DefaultVol FsVolume
	Vols       map[string]FsVolume
	Created    time.Time
	Logger     Logger
	mux        sync.Mutex
}

func (c *Connector) GetVolId(v FsVolume) string {
	for id, vol := range c.Vols {
		if vol.Name() == v.Name() {
			return id
		}
	}
	return ""
}
func (c *Connector) GetFsById(id string) FsVolume {
	return c.Vols[id]
}

func (c *Connector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	formParseFunc, ok := supportedMethods[r.Method]
	if !ok {
		c.Logger.Errorf("not support http method %s", r.Method)
		var data = ElfinderErr{
			Errs: []string{errConnect, r.Method},
		}
		if err := SendJson(w, data); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	if err := formParseFunc(r); err != nil {
		c.Logger.Errorf("HTTP form parse err: %s", err)
		var data = ElfinderErr{
			Errs: []string{errCmdParams, err.Error()},
		}
		if err := SendJson(w, data); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	cmd, err := parseCommand(r)
	if err != nil {
		c.Logger.Errorf("Parse command err: %s %+v", err, r.URL.Query())
		var data = ElfinderErr{
			Errs: []string{errCmdParams, err.Error()},
		}
		if err := SendJson(w, data); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	handleFunc, ok := supportedCommands[cmd]
	if !ok {
		c.Logger.Errorf("Command `%s` not supported", cmd)
		var data = ElfinderErr{
			Errs: []string{errCmdNoSupport, cmd},
		}
		if err := SendJson(w, data); err != nil {
			c.Logger.Error(err)
		}
		return
	}
	handleFunc(c, r, w)
}

func (c *Connector) parseTarget(target string) (vid, vPath string, err error) {
	return DecodeTarget(target)
}

type Options func(*option)

type option struct {
	Vols   []FsVolume
	Logger Logger
}

func WithVolumes(vols ...FsVolume) Options {
	return func(o *option) {
		o.Vols = vols
	}
}

func WithLogger(logger Logger) Options {
	return func(o *option) {
		o.Logger = logger
	}
}
