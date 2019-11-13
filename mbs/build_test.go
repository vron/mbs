package mbs

import (
	"bytes"
	"context"
	"encoding/hex"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vron/mbs/cache"
)

// TODO: When not using doublestar, move to in memory FS for testing.

func initFs() {
	cleanFs()
	write("README")
	write("log.txt")
	write("src/python/a.py")
	write("src/python/b.py")
	write("src/python/lib/lib.py")
	rand.Seed(0)
}

func cleanFs() {
	err := os.RemoveAll("./test")
	if err != nil {
		panic(err.Error())
	}
}

func write(path string, cont ...string) {
	data := ""
	if cont == nil {
		buff := make([]byte, 100)
		_, err := rand.Read(buff)
		if err != nil {
			panic(err.Error())
		}
		data = hex.EncodeToString(buff)
	} else {
		for _, v := range cont {
			data += v
		}
	}
	p := filepath.Join("test/data", path)
	if err := os.MkdirAll(filepath.Dir(p), 0777); err != nil {
		panic(err.Error())
	}
	if err := ioutil.WriteFile(p, []byte(data), 0777); err != nil {
		panic(err.Error())
	}
}

func expect(t *testing.T, target, output string) {
	c, err := cache.Open("test/cache")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer c.Close()
	buf := bytes.NewBuffer(nil)
	b := NewBuilder(c, Options{LogOutput: true, Stdout: buf})
	if target == "" {
		err = b.Build(context.Background(), "test/data/Makefile", []string{})
	} else {
		err = b.Build(context.Background(), "test/data/Makefile", []string{target})
	}

	if err != nil {
		t.Error(err)
	}

	out := strings.Replace(string(buf.Bytes()), "\n", "", -1)
	if out != output {
		t.Error("output not matching", "'"+out+"'", "!=", "'"+output+"'")
	}
}

func TestSimpleDoublestar(t *testing.T) {
	mf := `
all: **/*.py
	echo a
`

	initFs()
	defer cleanFs()

	write("Makefile", mf)
	expect(t, "all", "a")
	expect(t, "all", "")
	write("src/python/a.py")
	expect(t, "all", "a")
	expect(t, "all", "")
	write("src/python/lib/lib.py")
	expect(t, "all", "a")
	expect(t, "all", "")
}

func TestSimpleStar(t *testing.T) {
	mf := `
all: src/python/*.py
	echo a
`

	initFs()
	defer cleanFs()

	write("Makefile", mf)
	expect(t, "all", "a")
	expect(t, "all", "")
	write("src/python/a.py")
	expect(t, "all", "a")
	expect(t, "all", "")
	write("src/python/lib/lib.py")
	expect(t, "all", "")
	expect(t, "all", "")
}

func TestMultifile(t *testing.T) {
	mf2 := `
import "src/python" as py

b: a *.py
	echo b

a: lib/lib.py
	echo a
`
	mf1 := `
import "src/python" as py

all: py.b log.txt
	echo c
`

	initFs()
	defer cleanFs()

	write("Makefile", mf1)
	write("src/python/Makefile", mf2)

	expect(t, "all", "abc")
	expect(t, "all", "")
	write("src/python/a.py")
	expect(t, "all", "bc")
	expect(t, "all", "")
	write("src/python/lib/lib.py")
	expect(t, "all", "abc")
	expect(t, "all", "")
}

// todo: test so folders correct
