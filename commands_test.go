package elfinder

import (
	"io/fs"
	"os"
	"testing"

	"github.com/LeeEirc/elfinder/utils"
)

func TestDecodePath(t *testing.T) {
	path := "/Users/eric/Documents/github/elfinder/example"

	lfs := os.DirFS(path)
	cwd, err := fs.Stat(lfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("%+v\n", cwd)
	fm := cwd.Mode()
	t.Log(fm)
	newfm := fs.FileMode(0200)
	t.Logf("%b", fm)
	t.Log(utils.ReadWritePem(fm))
	t.Log(utils.ReadWritePem(newfm))
	t.Logf("%b", 1<<uint(9-1-0))
	t.Logf("%b", 1<<uint(9-1-1))
	//errs = volumes.WalkDir(lfs, ".", func(path string, d volumes.DirEntry, errs error) error {
	//	t.Log(path, d.IsDir(), d.Name())
	//	return nil
	//})
	//if errs != nil {
	//	t.Fatal(errs)
	//}

}
