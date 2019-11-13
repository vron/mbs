// package conf parses config files
package conf

import (
	"io"
	"strings"

	"github.com/vron/mbs/conf/parse"
)

type Makefile struct {
	Imports map[string]*Import
	Targets map[string]*Target
}

type Import struct {
	Pos  Pos
	Path string
	Name string
}

type Target struct {
	Pos  Pos
	Name string
	Deps []Dependency
	Cmds []Command
}

type Dependency struct {
	Pos      Pos
	Target   string
	Import   string
	Filename string
}

type Command struct {
	Pos Pos
	Cmd string
}

type ParseError struct {
	Pos Pos
	Err string
}

type Pos struct {
	Line   int
	Column int // Column in unicode characters
	Length int // Length in unicode characters
}

func (pe ParseError) Error() string {
	return pe.Err
}

// Parse is a blocking call that blocks until the entire file is parsed.
func Parse(input io.Reader) (*Makefile, error) {
	m := &Makefile{
		Imports: make(map[string]*Import, 5),
		Targets: make(map[string]*Target, 50),
	}
	if err := m.drainParser(input); err != nil {
		return nil, err
	}
	return m, m.check()
}

func (m *Makefile) drainParser(input io.Reader) error {
	p := parse.New(input)
	for {
		stm, ok := <-p.Statements
		if !ok {
			break
		}
		switch s := stm.(type) {
		case parse.ImportStatement:
			if m.Imports[s.Name] != nil {
				return ParseError{
					Err: "allready an import named: '" + s.Name + "'",
					Pos: Pos(s.NamePos),
				}
			}
			m.Imports[s.Name] = &Import{
				Pos:  Pos(s.PathPos),
				Name: s.Name,
				Path: s.Path,
			}
		case parse.TargetStatement:
			if m.Targets[s.Name] != nil {
				return ParseError{
					Err: "allready a target named: '" + s.Name + "'",
					Pos: Pos(s.NamePos),
				}
			}
			deps := make([]Dependency, len(s.Deps))
			for i, v := range s.Deps {
				deps[i].Target = v // As a first put them all in Target, we will seperate later
				deps[i].Pos = Pos(s.DepsPos[i])
			}
			cmds := make([]Command, len(s.Cmds))
			for i, v := range s.Cmds {
				cmds[i].Cmd = v // As a first put them all in Target, we will seperate later
				cmds[i].Pos = Pos(s.CmdsPos[i])
			}
			m.Targets[s.Name] = &Target{
				Pos:  Pos(s.NamePos),
				Name: s.Name,
				Deps: deps, // TOOD: Need to attach positions here.
				Cmds: cmds,
			}
		case parse.ErrorStatement:
			return ParseError{
				Pos: Pos(s.Pos),
				Err: s.Err,
			}
		default:
			panic("unknown statement type")
		}
	}
	return nil
}

func (m *Makefile) check() error {
	// run through all targets, splitting the deps into either local target,
	// imported target or file based on what is defined.
	for k, v := range m.Targets {
		for i, d := range v.Deps {
			if k == d.Target {
				return ParseError{
					Err: "a target cannot depend on itself",
					Pos: d.Pos,
				}
			}
			if m.Targets[d.Target] != nil {
				continue // This was a local target that exist
			}
			parts := strings.Split(d.Target, ".")
			if len(parts) > 1 && m.Imports[parts[0]] != nil {
				v.Deps[i].Import = parts[0]
				v.Deps[i].Target = strings.Join(parts[1:], ".")
				continue // This refered to the import, ok
			}
			// so it must refer to a file
			v.Deps[i].Filename = d.Target
			v.Deps[i].Target = ""
		}
	}
	return nil
}
