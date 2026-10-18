package compiscript

import (
	"sort"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

// CompilationReport keeps compilation separate from the existing analysis API.
// TAC is nil whenever any stage reports an error.
type CompilationReport struct {
	TAC         *tac.Program         `json:"tac"`
	Diagnostics model.Diagnostics    `json:"diagnostics"`
	Scopes      model.ScopeSnapshots `json:"scopes"`
	Codegen     *tac.CodegenReport   `json:"codegen,omitempty"`
}

// Compile parses and analyzes once, then lowers the supported language subset.
func Compile(source []byte) CompilationReport {
	program, frontendDiagnostics := frontend.Parse(source)
	scopes, semanticDiagnostics := semantic.Analyze(program)
	diagnostics := append(model.Diagnostics{}, frontendDiagnostics...)
	diagnostics = append(diagnostics, semanticDiagnostics...)
	// Keep Analyze's ordering without changing its established public behavior.
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
	report := CompilationReport{Diagnostics: diagnostics, Scopes: scopes}
	if len(diagnostics) == 0 {
		report.TAC, report.Diagnostics = tac.Generate(program, scopes)
		if report.TAC != nil && len(report.Diagnostics) == 0 {
			report.Codegen = tac.BuildCodegenReport(report.TAC, scopes)
		}
	}
	return report
}
