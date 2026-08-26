package compiscript

import (
	"sort"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
)

// Analyze parses source and returns one deterministic frontend/semantic report.
func Analyze(source []byte) model.AnalysisReport {
	program, frontendDiagnostics := frontend.Parse(source)
	scopes, semanticDiagnostics := semantic.Analyze(program)
	diagnostics := append(model.Diagnostics{}, frontendDiagnostics...)
	diagnostics = append(diagnostics, semanticDiagnostics...)
	sort.SliceStable(diagnostics, func(i, j int) bool {
		left, right := diagnostics[i], diagnostics[j]
		if left.Span.Start.Offset != right.Span.Start.Offset {
			return left.Span.Start.Offset < right.Span.Start.Offset
		}
		if left.Span.End.Offset != right.Span.End.Offset {
			return left.Span.End.Offset < right.Span.End.Offset
		}
		if left.Phase != right.Phase {
			return left.Phase < right.Phase
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		return left.Message < right.Message
	})
	return model.AnalysisReport{
		AST:         astView(program),
		Diagnostics: diagnostics,
		Scopes:      scopes,
	}
}
