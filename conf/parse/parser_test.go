package parse

import (
	"bytes"
	"reflect"
	"testing"
)

type testcase struct {
	src string
	res []Statement
}

var testCases = map[string]testcase{
	"empty": tc(``),
	"import": tc(`import "kalle/peter" as peter`,
		i("kalle/peter", "peter")),
	"importel": tc(`import "kalle/peter" as peter
`,
		i("kalle/peter", "peter")),
	"emptyrule": tc(`tgt:`,
		t("tgt", nil, nil)),
	"importtule": tc(`import:`,
		t("import", nil, nil)),
	"deps": tc(`import: a b/**.py d`,
		t("import", d("a", "b/**.py", "d"), nil)),
	"commands": tc(`import:c
  cmd1
  cmd2`,
		t("import", d("c"), c("cmd1", "cmd2"))),
	"multiple": tc(`
import "peter/sdf" as pd
import "viktor" as viktor

import:c
	  cmd1
kalle: a b c lisa
	  cmd2

lisa:
`,
		i("peter/sdf", "pd"), i("viktor", "viktor"),
		t("import", d("c"), c("cmd1")),
		t("kalle", d("a", "b", "c", "lisa"), c("cmd2")),
		t("lisa", nil, nil),
	),
	"importsf": tc(`import "kalle/peter" as peter
import "kalle/peter" as peter2`, i("kalle/peter", "peter"), i("kalle/peter", "peter2")),
	"importe": tc(`import "kalle/peter" as peter
peter:

:`, i("kalle/peter", "peter"), t("peter", nil, nil), e()),
}

func i(p, n string) ImportStatement {
	return ImportStatement{
		Name: n,
		Path: p,
	}
}

func tc(s string, stms ...Statement) testcase {
	return testcase{
		src: s,
		res: stms,
	}
}

func e() (e ErrorStatement) {
	return e
}

func t(name string, deps []string, cmds []string) TargetStatement {
	if deps == nil {
		deps = []string{}
	}
	if cmds == nil {
		cmds = []string{}
	}
	return TargetStatement{
		Name: name,
		Deps: deps,
		Cmds: cmds,
	}
}

func d(a ...string) []string {
	return a
}
func c(a ...string) []string {
	return a
}

func TestCase(t *testing.T) {
	for nm, tc := range testCases {
		t.Run(nm, func(t *testing.T) {
			stms := run(tc.src, t)
			check(tc, stms, t)
		})
	}
}

func run(input string, t *testing.T) []Statement {
	buff := bytes.NewBuffer([]byte(input))
	res := make([]Statement, 0, 10)
	p := New(buff)
	for {
		v, ok := <-p.Statements
		if !ok {
			break
		}
		res = append(res, v)
	}
	return res
}

func check(tc testcase, stms []Statement, t *testing.T) {
	for i, e := range tc.res {
		if i >= len(stms) {
			t.Error(i, "got to few statements, expected:", e)
			return
		}

		a := stms[i]
		if !equal(a, e) {
			t.Error(i, "got", a, "but expected", e)
		}
	}
	for i := len(tc.res); i < len(stms); i++ {
		t.Error("got to many, additional:", stms[i])
	}
}

func equal(a, b Statement) bool {
	// we cannot simply use DeepEqual since we do not want
	// to check on the positions here..
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}

	return true
}
