package lexer

import (
	_ "embed"
	"testing"
)

//go:embed test.acf
var testACF string

func TestLex(t *testing.T) {
	tokens, err := Lex(testACF)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", tokens)
}
