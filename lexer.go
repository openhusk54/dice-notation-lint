package main

import "strings"

type TokenType int

const (
	TokEOF TokenType = iota
	TokNewline
	TokInt
	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokPercent
	TokBang
	TokLParen
	TokRParen
	TokD
	TokKeepHigh
	TokKeepLow
	TokDropHigh
	TokDropLow
	TokIllegal
)

// Position is 1-based on both axes, matching how editors and compilers report
// locations, so a finding's Pos can be printed straight into a message.
type Position struct {
	Line int
	Col  int
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

type Lexer struct {
	input []rune
	pos   int
	line  int
	col   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(input), pos: 0, line: 1, col: 1}
}

func (l *Lexer) peekChar() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

// skipSpacesAndComments stops at '\n' rather than consuming it, since the
// parser treats newlines as statement separators.
func (l *Lexer) skipSpacesAndComments() {
	for {
		ch := l.peekChar()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
			continue
		}
		if ch == '#' {
			for l.peekChar() != '\n' && l.peekChar() != 0 {
				l.advance()
			}
			continue
		}
		break
	}
}

func (l *Lexer) NextToken() Token {
	l.skipSpacesAndComments()
	pos := Position{Line: l.line, Col: l.col}
	ch := l.peekChar()

	switch {
	case ch == 0:
		return Token{Type: TokEOF, Pos: pos}
	case ch == '\n':
		l.advance()
		return Token{Type: TokNewline, Literal: "\n", Pos: pos}
	case isDigit(ch):
		return l.readNumber(pos)
	case isLetter(ch):
		return l.readIdent(pos)
	default:
		l.advance()
		switch ch {
		case '+':
			return Token{Type: TokPlus, Literal: "+", Pos: pos}
		case '-':
			return Token{Type: TokMinus, Literal: "-", Pos: pos}
		case '*':
			return Token{Type: TokStar, Literal: "*", Pos: pos}
		case '/':
			return Token{Type: TokSlash, Literal: "/", Pos: pos}
		case '%':
			return Token{Type: TokPercent, Literal: "%", Pos: pos}
		case '!':
			return Token{Type: TokBang, Literal: "!", Pos: pos}
		case '(':
			return Token{Type: TokLParen, Literal: "(", Pos: pos}
		case ')':
			return Token{Type: TokRParen, Literal: ")", Pos: pos}
		default:
			return Token{Type: TokIllegal, Literal: string(ch), Pos: pos}
		}
	}
}

func (l *Lexer) readNumber(pos Position) Token {
	start := l.pos
	for isDigit(l.peekChar()) {
		l.advance()
	}
	return Token{Type: TokInt, Literal: string(l.input[start:l.pos]), Pos: pos}
}

// readIdent covers the die operator and every keep/drop modifier; anything
// else that looks like a word is illegal, which the parser turns into a
// pointed syntax error rather than a lexer-level one.
func (l *Lexer) readIdent(pos Position) Token {
	start := l.pos
	for isLetter(l.peekChar()) {
		l.advance()
	}
	lit := string(l.input[start:l.pos])
	switch strings.ToLower(lit) {
	case "d":
		return Token{Type: TokD, Literal: lit, Pos: pos}
	case "kh":
		return Token{Type: TokKeepHigh, Literal: lit, Pos: pos}
	case "kl":
		return Token{Type: TokKeepLow, Literal: lit, Pos: pos}
	case "dh":
		return Token{Type: TokDropHigh, Literal: lit, Pos: pos}
	case "dl":
		return Token{Type: TokDropLow, Literal: lit, Pos: pos}
	default:
		return Token{Type: TokIllegal, Literal: lit, Pos: pos}
	}
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
