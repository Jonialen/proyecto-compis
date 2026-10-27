// Package sql implements a small SQL interpreter that stores databases as
// plain files: lexer, parser, AST, semantic analysis and execution.
package sql

import (
	"fmt"
	"strings"
	"unicode"
)

// Diagnostic is an error located in the source text.
type Diagnostic struct {
	Phase     string `json:"phase"` // lexical, syntax, semantic or execution
	Message   string `json:"message"`
	Line      int    `json:"line"`
	Col       int    `json:"col"`
	Statement int    `json:"statement,omitempty"` // 1-based statement number, 0 when not applicable
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%d:%d: error %s: %s", d.Line, d.Col, d.Phase, d.Message)
}

// Token is a lexical unit. Type is the upper-case keyword, an operator
// symbol, or one of IDENT, INT, FLOAT, STRING, ILLEGAL, EOF.
type Token struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}

var keywords = map[string]bool{
	"CREATE": true, "DROP": true, "TABLE": true, "INDEX": true, "ON": true,
	"INSERT": true, "INTO": true, "VALUES": true, "UPDATE": true, "SET": true,
	"DELETE": true, "SELECT": true, "FROM": true, "WHERE": true, "AND": true,
	"OR": true, "NOT": true, "ORDER": true, "BY": true, "ASC": true, "DESC": true,
	"INNER": true, "JOIN": true, "GROUP": true, "HAVING": true, "IN": true,
	"EXISTS": true, "COUNT": true, "SUM": true, "AVG": true, "MIN": true,
	"MAX": true, "AS": true, "NULL": true, "TRUE": true, "FALSE": true,
	"PRIMARY": true, "KEY": true, "IS": true,
	"INT": true, "FLOAT": true, "VARCHAR": true, "BOOLEAN": true, "DATE": true,
}

// Lex splits src into tokens. Unknown characters and malformed literals are
// reported as lexical diagnostics; the token stream always ends with EOF.
func Lex(src string) ([]Token, []Diagnostic) {
	var toks []Token
	var diags []Diagnostic
	rs := []rune(src)
	i, line, col := 0, 1, 1

	adv := func() rune {
		r := rs[i]
		i++
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		return r
	}
	peekAt := func(k int) rune {
		if i+k < len(rs) {
			return rs[i+k]
		}
		return 0
	}
	lexErr := func(l, c int, format string, args ...any) {
		diags = append(diags, Diagnostic{Phase: "lexical", Message: fmt.Sprintf(format, args...), Line: l, Col: c})
	}
	isWord := func(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

	for i < len(rs) {
		r := rs[i]
		startLine, startCol := line, col
		switch {
		case unicode.IsSpace(r):
			adv()
		case r == '-' && peekAt(1) == '-':
			for i < len(rs) && rs[i] != '\n' {
				adv()
			}
		case r == '/' && peekAt(1) == '*':
			adv()
			adv()
			closed := false
			for i < len(rs) {
				if rs[i] == '*' && peekAt(1) == '/' {
					adv()
					adv()
					closed = true
					break
				}
				adv()
			}
			if !closed {
				lexErr(startLine, startCol, "Comentario de bloque sin cerrar")
			}
		case r == '_' || unicode.IsLetter(r):
			var b strings.Builder
			for i < len(rs) && isWord(rs[i]) {
				b.WriteRune(adv())
			}
			text := b.String()
			if up := strings.ToUpper(text); keywords[up] {
				toks = append(toks, Token{up, text, startLine, startCol})
			} else {
				toks = append(toks, Token{"IDENT", text, startLine, startCol})
			}
		case unicode.IsDigit(r):
			var b strings.Builder
			for i < len(rs) && unicode.IsDigit(rs[i]) {
				b.WriteRune(adv())
			}
			typ := "INT"
			if i < len(rs) && rs[i] == '.' && unicode.IsDigit(peekAt(1)) {
				typ = "FLOAT"
				b.WriteRune(adv())
				for i < len(rs) && unicode.IsDigit(rs[i]) {
					b.WriteRune(adv())
				}
			}
			if i < len(rs) && (rs[i] == '_' || unicode.IsLetter(rs[i])) {
				for i < len(rs) && isWord(rs[i]) {
					b.WriteRune(adv())
				}
				lexErr(startLine, startCol, "Número mal formado %q", b.String())
				toks = append(toks, Token{"ILLEGAL", b.String(), startLine, startCol})
				continue
			}
			toks = append(toks, Token{typ, b.String(), startLine, startCol})
		case r == '\'':
			adv()
			var b strings.Builder
			closed := false
			for i < len(rs) {
				c := adv()
				if c == '\'' {
					if i < len(rs) && rs[i] == '\'' {
						adv()
						b.WriteRune('\'')
						continue
					}
					closed = true
					break
				}
				b.WriteRune(c)
			}
			if !closed {
				lexErr(startLine, startCol, "Cadena de texto sin cerrar")
				toks = append(toks, Token{"ILLEGAL", "'" + b.String(), startLine, startCol})
				continue
			}
			toks = append(toks, Token{"STRING", b.String(), startLine, startCol})
		default:
			two := string(r) + string(peekAt(1))
			switch two {
			case "<=", ">=", "<>", "!=":
				adv()
				adv()
				typ := two
				if two == "!=" {
					typ = "<>"
				}
				toks = append(toks, Token{typ, two, startLine, startCol})
				continue
			}
			if strings.ContainsRune("=<>+-*/(),;.", r) {
				adv()
				toks = append(toks, Token{string(r), string(r), startLine, startCol})
				continue
			}
			adv()
			lexErr(startLine, startCol, "Carácter no reconocido %q", string(r))
			toks = append(toks, Token{"ILLEGAL", string(r), startLine, startCol})
		}
	}
	toks = append(toks, Token{"EOF", "", line, col})
	return toks, diags
}
