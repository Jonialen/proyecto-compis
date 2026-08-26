package frontend

import (
	"sort"
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
	"github.com/antlr4-go/antlr/v4"
)

type diagnosticCollector struct {
	index       sourceIndex
	diagnostics model.Diagnostics
}

func newDiagnosticCollector(index sourceIndex) *diagnosticCollector {
	return &diagnosticCollector{index: index}
}

func (c *diagnosticCollector) listener(phase model.Phase) *diagnosticListener {
	return &diagnosticListener{
		DefaultErrorListener: antlr.NewDefaultErrorListener(),
		collector:            c,
		phase:                phase,
	}
}

type diagnosticListener struct {
	*antlr.DefaultErrorListener
	collector *diagnosticCollector
	phase     model.Phase
}

func (l *diagnosticListener) SyntaxError(_ antlr.Recognizer, offendingSymbol interface{}, line, column int, message string, _ antlr.RecognitionException) {
	span := l.collector.index.spanFromScalars(l.collector.index.scalarAtLineColumn(line, column), l.collector.index.scalarAtLineColumn(line, column))
	if token, ok := offendingSymbol.(antlr.Token); ok {
		span = l.syntaxSpan(token, message)
	}
	l.collector.diagnostics = append(l.collector.diagnostics, model.Diagnostic{
		Code:    string(l.phase),
		Phase:   l.phase,
		Message: message,
		Span:    span,
	})
}

func (l *diagnosticListener) syntaxSpan(token antlr.Token, message string) ast.Span {
	if token.GetTokenType() == antlr.TokenEOF || strings.HasPrefix(message, "missing ") {
		position := l.collector.index.positionAtScalar(token.GetStart())
		return ast.Span{Start: position, End: position}
	}
	return l.collector.index.spanFromScalars(token.GetStart(), token.GetStop())
}

func orderDiagnostics(diagnostics model.Diagnostics) model.Diagnostics {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		return diagnostics[i].Span.Start.Offset < diagnostics[j].Span.Start.Offset
	})
	return diagnostics
}
