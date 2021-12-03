package elfinder

import (
	"io/fs"
	"os"
	"testing"
)

func TestDecodePath(t *testing.T) {
	path := "/Users/eric/Documents/github/elfinder/example"

	lfs := os.DirFS(path)
	cwd, err := fs.Stat(lfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	err = fs.WalkDir(lfs, ".", func(path string, d fs.DirEntry, err error) error {
		t.Log(path, d.IsDir(), d.Name())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v\n", cwd)
}
