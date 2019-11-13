package lex

import (
	"bytes"
	"testing"
)

type testcase struct {
	src string
	res []Token
}

var testCases = map[string]testcase{
	"empty": tc(``, t(EOF, "")),
	"import": tc(`import "kalle/peter" as peter`,
		t(Keyword, "import"), t(ImportPath, "kalle/peter"), t(Keyword, "as"), t(ImportName, "peter"), t(EOF, "")),
	"importel": tc(`import "kalle/peter" as peter
`,
		t(Keyword, "import"), t(ImportPath, "kalle/peter"), t(Keyword, "as"), t(ImportName, "peter"), t(Newline, "\n"), t(EOF, "")),
	"emptyrule": tc(`tgt:`,
		t(Target, "tgt"), t(Colon, ":"), t(EOF, "")),
	"importtule": tc(`import:`,
		t(Target, "import"), t(Colon, ":"), t(EOF, "")),
	"deps": tc(`import: a b/**.py d`,
		t(Target, "import"), t(Colon, ":"), t(Dependency, "a"), t(Dependency, "b/**.py"), t(Dependency, "d"), t(EOF, "")),
	"commands": tc(`import:
  cmd1
  cmd2`,
		t(Target, "import"), t(Colon, ":"), t(Newline, "\n"), t(Indent, "  "), t(Command, "cmd1"), t(Newline, "\n"), t(Indent, "  "), t(Command, "cmd2"), t(EOF, "")),
}

func tc(s string, tokens ...Token) testcase {
	return testcase{
		src: s,
		res: tokens,
	}
}

func t(t TokenType, v string) Token {
	return Token{
		Type: t,
		Val:  v,
	}
}

func TestLargeBuffer(t *testing.T) {
	blockSize = 1024 * 1024
	for nm, tc := range testCases {
		t.Run(nm, func(t *testing.T) {
			tokens := run(tc.src, t)
			check(tc, tokens, t)
		})
	}
}

func TestSmallBuffer(t *testing.T) {
	blockSize = 7
	for nm, tc := range testCases {
		t.Run(nm, func(t *testing.T) {
			tokens := run(tc.src, t)
			check(tc, tokens, t)
		})
	}
}

func run(input string, t *testing.T) []Token {
	buff := bytes.NewBuffer([]byte(input))
	c := make(chan Token, 10000)
	l := New(buff, c)
	l.Lex()
	res := make([]Token, 0, len(c))
	for {
		v, ok := <-c
		if !ok {
			break
		}
		res = append(res, v)
	}
	return res
}

func check(tc testcase, results []Token, t *testing.T) {
	for i, e := range tc.res {
		if i >= len(results) {
			t.Error(i, "got to few tokens, expected", e)
			return
		}

		a := results[i]
		if e.Type != a.Type || e.Val != a.Val {
			t.Error(i, "got", a, "but expected", e)
		}
	}
	for i := len(tc.res); i < len(results); i++ {
		t.Error("got to many, additional:", results[i])
	}
}
