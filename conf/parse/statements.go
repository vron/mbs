package parse

import "github.com/vron/mbs/conf/lex"

// A Statement represents part of a conf file.
type Statement interface {
	String() string
}

// A TargetStatement represents a build target.
type TargetStatement struct {
	Name    string
	NamePos lex.Pos
	Deps    []string // TODO: handle filenames with stars by escaping
	DepsPos []lex.Pos
	Cmds    []string
	CmdsPos []lex.Pos
}

// TODO: handle filenames with spaces by quotes
// TODO: handle shell invocations

// A ImportStatement repsenets the inclusion of another file.
type ImportStatement struct {
	Name    string
	NamePos lex.Pos
	Path    string
	PathPos lex.Pos
}

// A ErrorStatement reports an error occuring during the parsing
type ErrorStatement struct {
	Err string
	Pos lex.Pos
}

func (es ErrorStatement) String() string {
	return "err:" + es.Err
}

func (ts TargetStatement) String() string {
	s := ts.Name + ": "
	for _, d := range ts.Deps {
		s += d + " "
	}
	s += "\n"
	for _, d := range ts.Cmds {
		s += "\t" + d + "\n"
	}
	return "tgt:" + s + "\n"
}

func (is ImportStatement) String() string {
	s := "import "
	s += `"` + is.Path + `" as ` + is.Name
	s += "\n\n"
	return "imp:" + s
}
