package main

import (
	"net/http"

	"genanalex/internal/compiscript"
)

// Generate assembly only. Running user programs is deliberately not an HTTP API.
func handleCompiscriptMIPS(w http.ResponseWriter, r *http.Request) {
	handleCompiscriptRequest(w, r, func(source []byte) any {
		report := compiscript.CompileMIPS(source)
		text := ""
		if report.TAC != nil {
			text = report.TAC.String()
		}
		return struct {
			compiscript.MIPSCompilationReport
			TACText string `json:"tacText"`
		}{report, text}
	})
}
