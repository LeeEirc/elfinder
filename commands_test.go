package elfinder

import (
	"testing"
)

func TestDecodePath(t *testing.T) {
	id, path, err := parseTarget("X0_Lw")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(id, path)
	//ret := strings.SplitN("X0_Lw", "_", 2)
	//t.Log(ret)

}
