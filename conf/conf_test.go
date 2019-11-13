package conf

import (
	"bytes"
	"reflect"
	"testing"
)

func TestErrorSameImport(tt *testing.T) {
	src := `
import "sadf" as dd
import "fdaf" as dd`
	ensure(tt, src)
}

func TestErrorItself(tt *testing.T) {
	src := `
a: a`
	ensure(tt, src)
}

func TestComplicated(tt *testing.T) {
	src := `import "peter/sdf" as pd
import "viktor" as viktor

import:c
		cmd1
kalle: a b import lisa
		cmd2

lisa: viktor.a
`
	m := m(
		[]*Import{
			i(p(1, 8, 9), "peter/sdf", "pd"),
			i(p(2, 8, 6), "viktor", "viktor"),
		},
		[]*Target{
			t(p(4, 0, 6), "import",
				[]Dependency{
					d(p(4, 7, 1), "", "", "c"),
				},
				[]Command{
					c(p(5, 2, 4), "cmd1"),
				},
			),
			t(p(6, 0, 5), "kalle",
				[]Dependency{

					d(p(6, 7, 1), "", "", "a"),
					d(p(6, 9, 1), "", "", "b"),
					d(p(6, 11, 6), "import", "", ""),
					d(p(6, 18, 4), "lisa", "", ""),
				},
				[]Command{
					c(p(7, 2, 4), "cmd2"),
				},
			),
			t(p(9, 0, 4), "lisa",
				[]Dependency{
					d(p(9, 6, 8), "a", "viktor", ""),
				},
				[]Command{},
			),
		})
	check(tt, src, m)
}

func check(tt *testing.T, src string, m *Makefile) {
	r := bytes.NewReader([]byte(src))
	m2, e := Parse(r)
	if e != nil {
		tt.Error(e)
		return
	}
	if !reflect.DeepEqual(m, m2) {
		tt.Error("expected same")
	}
}

func ensure(tt *testing.T, src string) {
	r := bytes.NewReader([]byte(src))
	_, e := Parse(r)
	if e == nil {
		tt.Error("expected error but got none")
	}
}

func m(i []*Import, t []*Target) *Makefile {
	m := &Makefile{
		Imports: map[string]*Import{},
		Targets: map[string]*Target{},
	}
	for _, v := range i {
		m.Imports[v.Name] = v
	}
	for _, v := range t {
		m.Targets[v.Name] = v
	}
	return m
}

func p(a, b, c int) Pos {
	return Pos{a, b, c}
}

func i(p Pos, path, name string) *Import {
	return &Import{
		Pos:  p,
		Path: path,
		Name: name,
	}
}
func t(p Pos, name string, deps []Dependency, cmds []Command) *Target {
	return &Target{
		Pos:  p,
		Name: name,
		Deps: deps,
		Cmds: cmds,
	}
}

func d(p Pos, t, i, f string) Dependency {
	return Dependency{p, t, i, f}
}
func c(p Pos, c string) Command {
	return Command{p, c}
}
