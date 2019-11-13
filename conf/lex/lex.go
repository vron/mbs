// Package lex breaks a build config file into tokens, using limited lookahead.
package lex

import (
	"errors"
	"io"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// variable instead of constant for testing purposes
var blockSize = 1024 * 4

// Constants for the TokenTypes.
const (
	EOF = iota
	Keyword
	ImportPath
	ImportName
	Target
	Colon
	Dependency
	Command
	Comment
	Continuation
	Newline
	Indent
	Error
)

var typeNames = map[TokenType]string{
	EOF:          "EOF",
	Keyword:      "KEY",
	ImportPath:   "IPH",
	ImportName:   "INM",
	Target:       "TGT",
	Colon:        "COL",
	Dependency:   "DEP",
	Command:      "CMD",
	Comment:      "CMT",
	Continuation: "CNT",
	Newline:      "ENT",
	Indent:       "IND",
	Error:        "ERR",
}

const eof = -1

// A Lexer is not safe for concurrent use, it scans one file linearly from
// start to end.
type Lexer struct {
	state lexState
	err   *Token

	input       io.Reader
	inputBuffer []byte

	tokenBuffer *Token
	results     chan Token

	pos       int // current position, offset from startPos
	startPos  int // position in buffer of start of current
	widthLast int // width of the last rune read from input

	lineCount                    int
	columnCount, lastColumnCount int
	startLine, startColumn       int
}

// A TokenType represents the type of token it is.
type TokenType int

// A Token represents a tokenized part of the input.
type Token struct {
	Type TokenType
	Val  string

	Pos Pos
}

type Pos struct {
	Line   int
	Column int // Column in unicode characters
	Length int // Length in unicode characters
}

// String prints the string representation of the token
func (t Token) String() string {
	return typeNames[t.Type] + "(" + strconv.Quote(t.Val) + ")[" + strconv.Itoa(t.Pos.Line) + ":" + strconv.Itoa(t.Pos.Column) + "+" + strconv.Itoa(t.Pos.Length) + "]"
}

// Error returns an error if the token represents an error, nil otherwise.
func (t Token) Error() error {
	if t.Type == Error {
		return errors.New(t.Val)
	}
	return nil
}

// New creates a Lexer reading from input and delivering the tokens
// on results.
func New(input io.Reader, results chan Token) *Lexer {
	return &Lexer{
		input:       input,
		results:     results,
		inputBuffer: make([]byte, 0, 2*blockSize),
		lineCount:   1,
	}
}

// Lex starts lexing the input and delivering results, a call to Lex blocks until
// EOF or an error is reached.
func (l *Lexer) Lex() {
	l.state = l.lexLine
	for l.state != nil {
		tstate := l.state()
		if l.state == nil {
			break
		}
		l.state = tstate
	}
	if l.tokenBuffer != nil {
		l.results <- *l.tokenBuffer
	}
	close(l.results)
}

type lexState func() lexState

func (l *Lexer) error(s string) lexState {
	if l.err != nil {
		panic("error called when allready called before")
	}
	l.err = &Token{
		Val: "expected " + s + " but found: '" + string(l.peek()) + "'",
	}
	l.emit(Error)
	return nil
}

func (l *Lexer) lexLine() lexState {
	// invariant: assumes always at start of a statement line (non-cont.)
	indent := l.accept(isSpace)
	if indent >= 1 {
		l.emit(Indent)
		return l.maybeComment(l.lexCommand, true)
	}
	if isNewline(l.peek()) {
		return l.lexNewline
	}
	return l.maybeComment(l.lexRuleOrStatement, true)
}

func (l *Lexer) lexRuleOrStatement() lexState {
	if l.peek() == eof {
		l.emit(EOF)
		return nil
	}
	if le := l.accept(isNameCharacter); le <= 0 {
		return l.error("rule or statement")
	}
	// look-ahead on token to differentiate import statement and rule
	l.emit(-1)
	l.consumeSpace()
	switch l.peek() {
	case '"':
		l.tokenBuffer.Type = Keyword
		return l.lexImportPath
	case ':':
		l.tokenBuffer.Type = Target
		return l.lexRuleColon
	}
	return l.error("':' or '\"'")
}

func (l *Lexer) lexImportPath() lexState {
	if l.next() != '"' {
		return l.error("'\"'")
	}
	l.discard()
	for r := l.next(); r != '"'; r = l.next() {
	}
	if l.peek() == eof {
		return l.error("'\"'")
	}
	l.backup()
	l.emit(ImportPath)
	l.next()
	l.discard()
	return l.maybeComment(l.lexImportAs, false)
}

func (l *Lexer) lexImportAs() lexState {
	l.consumeSpace()
	if isNewline(l.peek()) {
		return l.lexNewline
	}
	if l.next() != 'a' {
		return l.error("keyword 'as'")
	}
	if l.next() != 's' {
		return l.error("keyword 'as'")
	}
	l.emit(Keyword)
	return l.lexImportName
}

func (l *Lexer) lexImportName() lexState {
	l.consumeSpace()
	if le := l.accept(isNameCharacter); le <= 0 {
		return l.error("name of import")
	}
	l.emit(ImportName)
	return l.maybeComment(l.lexNewline, true)
}

func (l *Lexer) lexRuleColon() lexState {
	if l.next() != ':' {
		return l.error("':'")
	}
	l.emit(Colon)
	return l.maybeComment(l.lexDep, true)
}

func (l *Lexer) lexRule() lexState {
	l.next()
	l.emit(Colon)
	l.consumeSpace()

	if isNewline(l.peek()) {
		return l.lexLine // no deps
	}

	return l.lexDep
}

func (l *Lexer) lexDep() lexState {
	// TODO: Should support line continuations here also
	l.consumeSpace()
	if isNewline(l.peek()) {
		return l.lexNewline
	}
	if l.accept(isDepCharacter) <= 0 {
		return l.error("dependency or newline")
	}
	l.emit(Dependency)
	return l.maybeComment(l.lexDep, true)
}

func (l *Lexer) lexCommand() lexState {
	// gulp everything in until eol, checking for line cont. this is a hack
	// to avoid having to understand the shell syntax, also break on EOF
	l.consumeSpace()
	var lastNonSpace rune = -1
	for r := l.next(); !isNewline(r) && r != eof; r = l.next() {
		if !isSpace(r) {
			lastNonSpace = r
		}
	}
	l.backup()
	l.emit(Command)
	if lastNonSpace == '\\' {
		l.emit(Continuation)
		return l.lexNewlineCmd
	}
	return l.lexLine
}

func (l *Lexer) lexNewline() lexState {
	l.consumeSpace()
	r := l.next()
	if r == '\r' {
		r = l.next()
	}
	if r == '\n' {
		l.emit(Newline)
		return l.lexLine
	}
	return l.error("linebreak")
}

func (l *Lexer) lexNewlineCmd() lexState {
	l.consumeSpace()
	r := l.next()
	if r == '\r' {
		r = l.next()
	}
	if r == '\n' {
		l.emit(Newline)
		return l.lexCommand
	}
	return l.error("linebreak")
}

func (l *Lexer) lexComment() lexState {
	if l.next() != '#' {
		return l.error("a comment, '#'")
	}
	for r := l.next(); !isNewline(r); r = l.next() {
	}
	l.backup()
	l.emit(Comment)
	return l.lexLine
}

func (l *Lexer) emit(typ TokenType) Token {
	var tok Token
	if l.err != nil {
		tok = *l.err
	} else {
		le := l.columnCount - l.startColumn
		if le < 0 {
			le = 0
		}
		tok = Token{
			Val: string(l.inputBuffer[l.startPos : l.startPos+l.pos]),
			Pos: Pos{
				Line:   l.startLine,
				Column: l.startColumn,
				Length: le, // This assumes no tokens break aline
			},
		}
	}
	tok.Type = typ
	if l.tokenBuffer != nil {
		l.results <- *l.tokenBuffer
	}
	l.tokenBuffer = &tok

	l.startPos += l.pos
	l.pos = 0
	l.startLine = l.lineCount
	l.startColumn = l.columnCount
	return tok
}

func (l *Lexer) discard() {
	l.startPos += l.pos
	l.pos = 0
	l.startLine = l.lineCount
	l.startColumn = l.columnCount
}

func (l *Lexer) consumeSpace() (indent int) {
	for r := l.next(); isSpace(r); r = l.next() {
		indent++
		if r == '\t' {
			indent++
		}
	}
	l.backup()
	l.discard()
	return
}

func (l *Lexer) accept(f func(rune) bool) (length int) {
	for ; f(l.next()); length++ {
	}
	l.backup()
	return
}

func (l *Lexer) next() rune {
	l.maybeLoad()
	if l.startPos+l.pos >= len(l.inputBuffer) {
		l.widthLast = 0
		return eof
	}
	r, w := utf8.DecodeRune(l.inputBuffer[l.startPos+l.pos:])
	l.widthLast = w
	l.pos += w
	if r == '\n' {
		l.lineCount++
		l.lastColumnCount = l.columnCount
		l.columnCount = 0
	} else {
		l.columnCount++
	}
	return r
}

func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *Lexer) backup() {
	l.pos -= l.widthLast
	if l.widthLast == 1 && l.inputBuffer[l.pos+l.startPos] == '\n' {
		l.lineCount--
		l.columnCount = l.lastColumnCount
	} else {
		l.columnCount--
	}
}

func (l *Lexer) maybeLoad() {
	if l.input == nil {
		return // reached the end of the input
	}
	// don't load if not needed
	if len(l.inputBuffer)-(l.pos+l.startPos) >= 1 {
		return
	}
	l.load()
}

func (l *Lexer) load() {
	if l.startPos < blockSize {
		// we are reading a long token, nothing to do but to grow memory
	} else {
		// common case, shift the data down to avoid allocating more memory
		copy(l.inputBuffer[:], l.inputBuffer[l.startPos-4:])
		l.inputBuffer = l.inputBuffer[:len(l.inputBuffer)-(l.startPos-4)]
		l.startPos = 4
	}

	var buff []byte
	if true || cap(l.inputBuffer)-len(l.inputBuffer) < blockSize {
		// we must likely grow the slice to add the data
		buff = make([]byte, blockSize)
	} else {
		buff = l.inputBuffer[len(l.inputBuffer) : len(l.inputBuffer)+blockSize]
	}
	n, err := l.input.Read(buff)
	buff = buff[:n]
	if err == io.EOF {
		l.input = nil
	} else if err != nil {
		l.error("error reading input: " + err.Error())
		return // TODO: Check if we need to add some data to make sure that the calling func can continue without lots of checks or panicic, the data will not amtter since the error will terminate it all relatively soon
	}

	l.inputBuffer = append(l.inputBuffer, buff...)
}

func (l *Lexer) maybeComment(alt lexState, acceptEOF bool) lexState {
	l.consumeSpace()
	if l.peek() == '#' {
		return l.lexComment
	}
	if acceptEOF && l.peek() == eof {
		l.emit(EOF)
		return nil
	}
	return alt
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r) && r != '\n' && r != '\r'
}

func isNameCharacter(r rune) bool {
	return unicode.In(r, unicode.Digit, unicode.Letter) || r == '_'
}

func isDepCharacter(r rune) bool {
	return unicode.In(r, unicode.Digit, unicode.Letter) || r == '_' || r == '.' || r == '*' || r == '/' || r == '\\'
}

func isNewline(r rune) bool {
	return r == '\n' || r == '\r'
}
