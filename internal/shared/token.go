// Package shared define contratos reutilizables entre subsistemas.
package shared

// Token es el contrato compartido mínimo entre lexer y parser.
type Token struct {
	Type   string
	Lexeme string
	Line   int
}
