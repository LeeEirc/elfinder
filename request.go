package elfinder

type ELFRequest struct {
	Cmd        string   `form:"cmd"`
	Init       bool     `form:"init"`
	Tree       bool     `form:"tree"`
	Name       string   `form:"name"`
	Target     string   `form:"target"`
	Targets    []string `form:"targets[]"`
	Dirs       []string `form:"dirs[]"`
	Mode       string   `form:"mode"`
	Bg         string   `form:"bg"`
	Width      int      `form:"width"`
	Height     int      `form:"height"`
	X          int      `form:"x"`
	Y          int      `form:"y"`
	Degree     int      `form:"degree"`
	Quality    int      `form:"quality"`
	Renames    []string `form:"renames[]"`
	Suffix     string   `form:"suffix"`
	Intersect  []string `form:"intersect[]"`
	Chunk      string   `form:"chunk"`
	UploadPath []string `form:"upload_path[]"`
	Cid        int      `form:"cid"`
	Content    string   `form:"content"`
	Dst        string   `form:"dst"`
	Src        string   `form:"src"`
	Cut        bool     `form:"cut"`
	Type       string   `form:"type"`
	MakeDir    bool     `form:"makedir"`
}

