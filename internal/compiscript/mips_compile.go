package compiscript

import (
	"errors"

	"genanalex/internal/compiscript/mips"
	"genanalex/internal/compiscript/model"
)

// MIPSCompilationReport adds target assembly without changing Compile's TAC API.
// Compilation never runs the source program or invokes an external simulator.
type MIPSCompilationReport struct {
	CompilationReport
	MIPSText string `json:"mipsText"`
}

func CompileMIPS(source []byte) MIPSCompilationReport {
	report := MIPSCompilationReport{CompilationReport: Compile(source)}
	if report.TAC == nil || len(report.Diagnostics) != 0 {
		return report
	}
	assembly, err := mips.Generate(report.TAC)
	if err != nil {
		diagnostic := model.Diagnostic{Code: "MIPS_GENERATION", Phase: model.Phase("codegen"), Message: err.Error()}
		var located *mips.Error
		if errors.As(err, &located) {
			diagnostic.Span = located.Span
		}
		report.Diagnostics = append(report.Diagnostics, diagnostic)
		return report
	}
	report.MIPSText = assembly
	return report
}
