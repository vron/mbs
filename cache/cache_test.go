package cache

import (
	"os"
	"testing"
)

func TestSimple(t *testing.T) {
	os.Remove("./test.test")
	c, err := Open("./test.test")
	if err != nil {
		t.Error(err)
	}

	if !c.Set("aa", v("a")) {
		t.Error("expected true")
	}
	if c.Set("aa", v("a")) {
		t.Error("expected true")
	}
	if !c.Set("aa", v("b")) {
		t.Error("expected true")
	}
	if !c.Set("ab", v("a")) {
		t.Error("expected true")
	}
	os.Remove("./test.test")
}

func v(s string) []byte {
	b := []byte(s)
	for len(b) < ValueSize {
		b = append(b, 0)
	}
	return b
}
