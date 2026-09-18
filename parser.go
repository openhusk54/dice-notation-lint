package main

import (
	"fmt"
	"strconv"
)

// Node is anything in the expression tree. Position always points at the
// leftmost token that produced the node, so lint rules can report findings
// without recomputing locations.
type Node interface {
	Position() Position
}

type NumberLit struct {
	Value int
	Pos   Position
}

func (n *NumberLit) Position() Position { return n.Pos }

type Modifier struct {
	Kind  TokenType // TokKeepHigh, TokKeepLow, TokDropHigh, TokDropLow
	Count *NumberLit
	Pos   Position
}

type DiceExpr struct {
	Count    *NumberLit // nil means an implicit count of 1, e.g. "d20"
	DPos     Position
	Sides    *NumberLit // nil when Percent is true
	Percent  bool
	Modifier *Modifier
}

func (d *DiceExpr) Position() Position {
	if d.Count != nil {
		return d.Count.Pos
	}
	return d.DPos
}

type BinaryExpr struct {
	Op    TokenType
	OpPos Position
	Left  Node
	Right Node
}

func (b *BinaryExpr) Position() Position { return b.Left.Position() }

type UnaryExpr struct {
	Op    TokenType
	OpPos Position
	Right Node
}

func (u *UnaryExpr) Position() Position { return u.OpPos }

type ParenExpr struct {
	LParenPos Position
	Inner     Node
}

func (p *ParenExpr) Position() Position { return p.LParenPos }

// parseError carries the position of the offending token through a panic so
// a single recover point per statement can turn it into a Finding without
// threading error returns through every parse function.
type parseError struct {
	pos Position
	msg string
}

type Parser struct {
	lex      *Lexer
	cur      Token
	peek     Token
	findings []Finding
}

func NewParser(source string) *Parser {
	p := &Parser{lex: NewLexer(source)}
	p.cur = p.lex.NextToken()
	p.peek = p.lex.NextToken()
	return p
}

func (p *Parser) next() {
	p.cur = p.peek
	p.peek = p.lex.NextToken()
}

func (p *Parser) errorf(pos Position, format string, args ...interface{}) {
	panic(parseError{pos: pos, msg: fmt.Sprintf(format, args...)})
}

// ParseProgram treats each newline-separated line as one statement, so a
// syntax error on one line doesn't stop the rest of the file from being
// checked.
func (p *Parser) ParseProgram() []Node {
	var stmts []Node
	for p.cur.Type == TokNewline {
		p.next()
	}
	for p.cur.Type != TokEOF {
		if stmt := p.parseStatement(); stmt != nil {
			stmts = append(stmts, stmt)
		}
		for p.cur.Type == TokNewline {
			p.next()
		}
	}
	return stmts
}

func (p *Parser) parseStatement() (result Node) {
	defer func() {
		if r := recover(); r != nil {
			pe, ok := r.(parseError)
			if !ok {
				panic(r)
			}
			p.findings = append(p.findings, Finding{
				Pos:      pe.pos,
				Severity: SeverityError,
				Rule:     "syntax",
				Message:  pe.msg,
			})
			p.recoverToNewline()
			result = nil
		}
	}()
	return p.parseExpr()
}

func (p *Parser) recoverToNewline() {
	for p.cur.Type != TokNewline && p.cur.Type != TokEOF {
		p.next()
	}
}

func (p *Parser) parseExpr() Node {
	left := p.parseTerm()
	for p.cur.Type == TokPlus || p.cur.Type == TokMinus {
		op, opPos := p.cur.Type, p.cur.Pos
		p.next()
		right := p.parseTerm()
		left = &BinaryExpr{Op: op, OpPos: opPos, Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseTerm() Node {
	left := p.parseUnary()
	for p.cur.Type == TokStar || p.cur.Type == TokSlash {
		op, opPos := p.cur.Type, p.cur.Pos
		p.next()
		right := p.parseUnary()
		left = &BinaryExpr{Op: op, OpPos: opPos, Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseUnary() Node {
	if p.cur.Type == TokMinus {
		opPos := p.cur.Pos
		p.next()
		return &UnaryExpr{Op: TokMinus, OpPos: opPos, Right: p.parseUnary()}
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() Node {
	switch p.cur.Type {
	case TokLParen:
		lp := p.cur.Pos
		p.next()
		inner := p.parseExpr()
		if p.cur.Type != TokRParen {
			p.errorf(p.cur.Pos, "expected ')' to close '(' opened at line %d, column %d", lp.Line, lp.Col)
		}
		p.next()
		return &ParenExpr{LParenPos: lp, Inner: inner}
	case TokInt:
		numTok := p.cur
		numVal := p.parseIntLiteral(numTok)
		p.next()
		if p.cur.Type == TokD {
			return p.parseDice(&NumberLit{Value: numVal, Pos: numTok.Pos})
		}
		return &NumberLit{Value: numVal, Pos: numTok.Pos}
	case TokD:
		return p.parseDice(nil)
	default:
		p.errorf(p.cur.Pos, "expected a number or a dice expression, found %s", tokenDesc(p.cur))
		return nil
	}
}

func (p *Parser) parseDice(count *NumberLit) Node {
	dPos := p.cur.Pos
	p.next() // consume 'd'

	var sides *NumberLit
	percent := false
	switch p.cur.Type {
	case TokInt:
		sidesTok := p.cur
		sides = &NumberLit{Value: p.parseIntLiteral(sidesTok), Pos: sidesTok.Pos}
		p.next()
	case TokPercent:
		percent = true
		p.next()
	default:
		p.errorf(p.cur.Pos, "expected a number of sides (or '%%') after 'd', found %s", tokenDesc(p.cur))
	}

	dice := &DiceExpr{Count: count, DPos: dPos, Sides: sides, Percent: percent}

	if isModifierToken(p.cur.Type) {
		modPos := p.cur.Pos
		kind := p.cur.Type
		p.next()
		var modCount *NumberLit
		if p.cur.Type == TokInt {
			ct := p.cur
			modCount = &NumberLit{Value: p.parseIntLiteral(ct), Pos: ct.Pos}
			p.next()
		}
		dice.Modifier = &Modifier{Kind: kind, Count: modCount, Pos: modPos}
	}

	return dice
}

func (p *Parser) parseIntLiteral(tok Token) int {
	val, err := strconv.Atoi(tok.Literal)
	if err != nil {
		p.errorf(tok.Pos, "invalid integer literal %q", tok.Literal)
	}
	return val
}

func isModifierToken(t TokenType) bool {
	return t == TokKeepHigh || t == TokKeepLow || t == TokDropHigh || t == TokDropLow
}

func tokenDesc(t Token) string {
	switch t.Type {
	case TokEOF:
		return "end of input"
	case TokNewline:
		return "end of line"
	default:
		return "'" + t.Literal + "'"
	}
}
