package compiscript

import (
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
	return model.AnalysisReport{
		AST:         astView(program),
		Diagnostics: diagnostics,
		Scopes:      scopes,
	}
}
