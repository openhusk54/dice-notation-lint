package main

import (
	"fmt"
	"sort"
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	if s == SeverityError {
		return "error"
	}
	return "warning"
}

type Finding struct {
	Pos      Position
	Severity Severity
	Rule     string
	Message  string
}

// Lint parses source as a list of newline-separated dice expressions and
// returns every finding, syntax and semantic alike, ordered by position so
// output reads top to bottom the way the source file does.
func Lint(source string) []Finding {
	parser := NewParser(source)
	stmts := parser.ParseProgram()

	findings := append([]Finding{}, parser.findings...)
	for _, stmt := range stmts {
		findings = append(findings, checkNode(stmt)...)
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Pos.Line != findings[j].Pos.Line {
			return findings[i].Pos.Line < findings[j].Pos.Line
		}
		return findings[i].Pos.Col < findings[j].Pos.Col
	})
	return findings
}

func checkNode(n Node) []Finding {
	var findings []Finding
	switch node := n.(type) {
	case *BinaryExpr:
		findings = append(findings, checkNode(node.Left)...)
		findings = append(findings, checkNode(node.Right)...)
	case *UnaryExpr:
		findings = append(findings, checkNode(node.Right)...)
	case *ParenExpr:
		findings = append(findings, checkNode(node.Inner)...)
	case *DiceExpr:
		findings = append(findings, checkDice(node)...)
	}
	return findings
}

func checkDice(d *DiceExpr) []Finding {
	var findings []Finding

	count := 1
	if d.Count != nil {
		count = d.Count.Value
		switch {
		case count == 0:
			findings = append(findings, Finding{
				Pos:      d.Count.Pos,
				Severity: SeverityError,
				Rule:     "zero-dice-count",
				Message:  "dice count is 0, this expression always rolls nothing",
			})
		case count > 1000:
			findings = append(findings, Finding{
				Pos:      d.Count.Pos,
				Severity: SeverityWarning,
				Rule:     "huge-dice-count",
				Message:  fmt.Sprintf("rolling %d dice in one expression is unusual and may be a typo", count),
			})
		}
	}

	if !d.Percent && d.Sides != nil {
		switch d.Sides.Value {
		case 0:
			findings = append(findings, Finding{
				Pos:      d.Sides.Pos,
				Severity: SeverityError,
				Rule:     "zero-sided-die",
				Message:  "a die cannot have 0 sides",
			})
		case 1:
			findings = append(findings, Finding{
				Pos:      d.Sides.Pos,
				Severity: SeverityWarning,
				Rule:     "one-sided-die",
				Message:  "a d1 always rolls 1, consider replacing it with the constant 1",
			})
		}
	}

	if d.Modifier != nil {
		modCount := 1
		modPos := d.Modifier.Pos
		if d.Modifier.Count != nil {
			modCount = d.Modifier.Count.Value
			modPos = d.Modifier.Count.Pos
		}
		switch {
		case modCount == 0:
			findings = append(findings, Finding{
				Pos:      modPos,
				Severity: SeverityWarning,
				Rule:     "modifier-zero-count",
				Message:  "keeping or dropping 0 dice has no effect on the roll",
			})
		case modCount > count:
			findings = append(findings, Finding{
				Pos:      modPos,
				Severity: SeverityError,
				Rule:     "keep-drop-exceeds-count",
				Message:  fmt.Sprintf("modifier affects %d dice but only %d are rolled", modCount, count),
			})
		}
	}

	return findings
}
