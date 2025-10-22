package acf

import (
	_ "embed"
	"testing"
)

//go:embed test.acf
var testACF string

func TestParse(t *testing.T) {
	v, err := FromString(testACF)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", v)
}
