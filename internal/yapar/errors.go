// Package yapar modela errores y contratos base del parser sintáctico.
package yapar

import (
	"errors"
	"fmt"
)

// ErrNotImplemented marca etapas del pipeline aún no implementadas.
var ErrNotImplemented = errors.New("yapar: not implemented")

// SpecError representa errores al interpretar un archivo .yalp.
type SpecError struct {
	Line    int
	Message string
}

func (e *SpecError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Line > 0 {
		return fmt.Sprintf("yapar spec line %d: %s", e.Line, e.Message)
	}
	return fmt.Sprintf("yapar spec: %s", e.Message)
}

// GrammarConflictError representa conflictos ACTION/GOTO detectados en la tabla.
type GrammarConflictError struct {
	State   int
	Symbol  string
	Kind    string
	Current Action
	New     Action
}

func (e *GrammarConflictError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"yapar grammar conflict (%s) at state %d, symbol %q: existing=%s new=%s",
		e.Kind,
		e.State,
		e.Symbol,
		formatAction(e.Current),
		formatAction(e.New),
	)
}

// SyntaxError representa errores del simulador LR sobre un stream de tokens.
type SyntaxError struct {
	Line     int
	GotType  string
	Lexeme   string
	Expected []string
}

func (e *SyntaxError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if len(e.Expected) == 0 {
		return fmt.Sprintf("syntax error at line %d: got %q (%q)", e.Line, e.GotType, e.Lexeme)
	}
	return fmt.Sprintf("syntax error at line %d: got %q (%q), expected %v", e.Line, e.GotType, e.Lexeme, e.Expected)
}
