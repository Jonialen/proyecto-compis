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

// LL1ConflictError representa conflictos predict/predict al llenar la tabla LL(1).
type LL1ConflictError struct {
	NonTerminal  string
	Terminal     string
	ExistingProd int
	ConflictProd int
}

func (e *LL1ConflictError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"yapar ll1 conflict at non-terminal %q with lookahead %q: existing production=%d conflicting production=%d",
		e.NonTerminal,
		e.Terminal,
		e.ExistingProd,
		e.ConflictProd,
	)
}

// LeftRecursionError representa recursión izquierda directa incompatible con LL(1).
type LeftRecursionError struct {
	NonTerminal  string
	ProductionID int
}

func (e *LeftRecursionError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"yapar ll1 left recursion at non-terminal %q in production %d",
		e.NonTerminal,
		e.ProductionID,
	)
}
