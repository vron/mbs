// Package parse assembles lex tokens into import and target statements.
package parse

import (
	"io"
	"sync"

	"github.com/vron/mbs/conf/lex"
)

// A Parser is used to parse a single file.
type Parser struct {
	// Will keep outputting statements until EOF at which point the channel is closed.
	Statements chan Statement

	buff, last *lex.Token

	mu sync.Mutex

	l   *lex.Lexer
	lex chan lex.Token
}

// New builds a new parser and starts to parse the input. The results are reached through
// the Statements channel.
func New(input io.Reader) *Parser {
	// use relatively large buffers so processing can continue
	// despite no readers currently waiting
	buffSize := 500
	lexc := make(chan lex.Token, buffSize)
	l := lex.New(input, lexc)
	go l.Lex()
	p := &Parser{
		l:          l,
		lex:        lexc,
		Statements: make(chan Statement, buffSize),
	}
	p.mu.Lock()
	go p.parse()
	return p
}

// Wait blocks until either the first error or the entire reader has
// been parsed and sent on the p.Statements chan
func (p *Parser) Wait() {
	// wait until done
	p.mu.Lock()
	p.mu.Unlock()
	return
}

// Abort reading and processing the input, block until termination is successful.
func (p *Parser) Abort() {
	// TODO: Implement, abort instead of waiting until end...
	p.mu.Lock()
	p.mu.Unlock()
}

func (p *Parser) parse() {
	// we are not building a tree, so really works more like a lexer for now.

	for state := p.parseStatement; state != nil; state = state() {
	}
	close(p.Statements)
	p.mu.Unlock()
}

type parseState func() parseState

func (p *Parser) eatCommentNewline() {
	tok := p.next()
	for tok.Type == lex.Newline || tok.Type == lex.Comment {
		tok = p.next()
	}
	p.backup()
}

func (p *Parser) parseStatement() parseState {
	// a statement is either and import pr a target, i.e we expect the line
	// not to start with an indent..

	// TODO: Make this work a lot better to find error case
	p.eatCommentNewline()
	tok := p.next()

	if tok.Type == lex.Keyword && tok.Val == "import" {
		path := p.next()
		if path.Type != lex.ImportPath {
			return p.error("expected import path", path)
		}
		as := p.next()
		if as.Type != lex.Keyword || as.Val != "as" {
			return p.error("expected as after import path", as)
		}
		name := p.next()
		if name.Type != lex.ImportName {
			return p.error("expected import name", name)
		}

		// we have all we need to issue an import statement, first
		// parse out the name if none is given
		importName := name.Val
		if importName == "" {
			return p.error("expected an import name that can be used", name)
		}
		p.Statements <- ImportStatement{
			Path:    path.Val,
			PathPos: path.Pos,
			Name:    name.Val,
			NamePos: name.Pos,
		}
		return p.parseStatement
	}

	if tok.Type == lex.Target {
		colon := p.next()
		if colon.Type != lex.Colon {
			return p.error("colon", colon)
		}
		// TODO: support multiple lines in a good way here too

		deps := []string{}
		depspos := []lex.Pos{}

		for n := p.next(); n.Type == lex.Dependency; n = p.next() {
			deps = append(deps, n.Val)
			depspos = append(depspos, n.Pos)
		}
		p.backup()

		// eat comments and a newline
		for tok := p.next(); tok.Type == lex.Newline || tok.Type == lex.Comment; {
			tok = p.next()
		}
		p.backup()

		// Now we have to options, either the next line is a indent + a command
		// or it is something else, in which case we parse it separately.
		// TODO: Handle comment lines
		cmds := []string{}
		cmdspos := []lex.Pos{}
		for {
			cmd, pos, ok := p.maybeReadCommand()
			if ok {
				cmds = append(cmds, cmd)
				cmdspos = append(cmdspos, pos)
			} else {
				break
			}
		}
		p.Statements <- TargetStatement{
			Name:    tok.Val,
			NamePos: tok.Pos,
			Deps:    deps,
			DepsPos: depspos,
			Cmds:    cmds,
			CmdsPos: cmdspos,
		}

		return p.parseStatement

	}

	if tok.Type == lex.EOF {
		return nil
	}

	return p.error("expected import or target", tok)
}

func (p *Parser) maybeReadCommand() (string, lex.Pos, bool) {
	// try to read out a command, else return false if not possible
	p.eatCommentNewline()
	if p.next().Type != lex.Indent {
		p.backup()
		return "", lex.Pos{}, false
	}
	t := p.next()
	if t.Type != lex.Command {
		p.error("expected command after indent", t)
		return "", lex.Pos{}, false
	}
	return t.Val, t.Pos, true
}

func (p *Parser) next() lex.Token {
	if p.buff != nil {
		a := *p.buff
		p.buff = nil
		return a
	}
	t := <-p.lex
	p.last = &t
	return t
}

func (p *Parser) backup() {
	p.buff = p.last
}

func (p *Parser) error(e string, t lex.Token) parseState {
	p.Statements <- ErrorStatement{
		Err: e,
		Pos: t.Pos,
	}
	return nil
}
