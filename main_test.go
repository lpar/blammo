package blammo

import (
	"testing"
)

var spliceTests = []struct {
	txt string
	ins string
	pos int
	out string
}{
	{"abcdefg", "zzz", 3, "abczzzdefg"},
	{"abcdefg", "1234567890", 6, "abcdef1234567890g"},
}

func TestSplice(t *testing.T) {
	for _, tdat := range spliceTests {
		t.Run(tdat.txt, func(t *testing.T) {
			x := string(splice([]byte(tdat.txt), []byte(tdat.ins), tdat.pos))
			if x != tdat.out {
				t.Errorf("got %s, expected %s", x, tdat.out)
			}
		})
	}
}
