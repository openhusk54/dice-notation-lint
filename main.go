package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dicelint <file> [file...]")
		os.Exit(2)
	}

	hasError := false
	for _, path := range args {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dicelint: %s: %v\n", path, err)
			hasError = true
			continue
		}

		source := string(data)
		lines := strings.Split(source, "\n")
		for _, f := range Lint(source) {
			printFinding(os.Stdout, path, lines, f)
			if f.Severity == SeverityError {
				hasError = true
			}
		}
	}

	if hasError {
		os.Exit(1)
	}
}

// printFinding renders a finding the way a compiler would: the location,
// the message, the rule that fired, the source line, and a caret under the
// exact column so the reader never has to go count characters by hand.
func printFinding(w io.Writer, path string, lines []string, f Finding) {
	fmt.Fprintf(w, "%s:%d:%d: %s: %s [%s]\n", path, f.Pos.Line, f.Pos.Col, f.Severity, f.Message, f.Rule)
	if f.Pos.Line-1 >= 0 && f.Pos.Line-1 < len(lines) {
		lineText := lines[f.Pos.Line-1]
		fmt.Fprintf(w, "    %s\n", lineText)
		fmt.Fprintf(w, "    %s^\n", strings.Repeat(" ", f.Pos.Col-1))
	}
}
