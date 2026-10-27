package sql

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// FormatTable renders a result set as an aligned text table.
func FormatTable(cols []string, rows [][]any) string {
	widths := make([]int, len(cols))
	cells := make([][]string, len(rows))
	for i, c := range cols {
		widths[i] = utf8.RuneCountInString(c)
	}
	for r, row := range rows {
		cells[r] = make([]string, len(row))
		for i, v := range row {
			s := strings.ReplaceAll(FormatValue(v), "\n", `\n`)
			cells[r][i] = s
			widths[i] = max(widths[i], utf8.RuneCountInString(s))
		}
	}
	line := func(vals []string) string {
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = v + strings.Repeat(" ", widths[i]-utf8.RuneCountInString(v))
		}
		return strings.TrimRight(strings.Join(parts, " | "), " ") + "\n"
	}
	var b strings.Builder
	b.WriteString(line(cols))
	seps := make([]string, len(cols))
	for i, w := range widths {
		seps[i] = strings.Repeat("-", w)
	}
	b.WriteString(strings.Join(seps, "-+-") + "\n")
	for _, row := range cells {
		b.WriteString(line(row))
	}
	return b.String()
}

// FormatResult renders one executed statement: its message, the result set
// of a SELECT and how each table was read.
func FormatResult(r StmtResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%d] %s\n", r.Statement, r.Message)
	if r.Columns != nil {
		b.WriteString(FormatTable(r.Columns, r.Rows))
	}
	for _, p := range r.Plan {
		b.WriteString("  plan: " + p + "\n")
	}
	return b.String()
}
