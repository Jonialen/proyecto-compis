package yapar

import (
	"encoding/json"
	"fmt"
	"time"

	"genanalex/internal/shared"
)

type MethodResult struct {
	Method   Method
	Report   *VisualizationReport
	Accepted *bool
	Error    string
	Duration time.Duration
}

type ComparisonReport struct {
	Methods   []MethodResult
	TotalTime time.Duration
}

func (r *ComparisonReport) HasSuccess() bool {
	if r == nil {
		return false
	}
	for _, result := range r.Methods {
		if result.Error == "" {
			return true
		}
	}
	return false
}

func BuildComparisonReport(g *Grammar, ff *FirstFollow, tokens []shared.Token, methods []Method) *ComparisonReport {
	started := time.Now()
	report := &ComparisonReport{Methods: make([]MethodResult, 0, len(methods))}

	for _, method := range methods {
		result := MethodResult{Method: method}
		methodStarted := time.Now()

		visReport, err := BuildVisReport(g, ff, method)
		if err != nil {
			result.Error = err.Error()
			result.Duration = time.Since(methodStarted)
			report.Methods = append(report.Methods, result)
			continue
		}
		result.Report = visReport

		if tokens != nil {
			parser, err := BuildParser(g, ff, method)
			if err != nil {
				result.Error = err.Error()
				result.Duration = time.Since(methodStarted)
				report.Methods = append(report.Methods, result)
				continue
			}
			parseResult, err := parser.Parse(tokens)
			if err != nil {
				result.Error = err.Error()
			} else if parseResult == nil {
				result.Error = "parse result was nil"
			} else {
				accepted := parseResult.Accepted
				result.Accepted = &accepted
				if !accepted {
					result.Error = "input was not accepted"
				}
			}
		}

		result.Duration = time.Since(methodStarted)
		report.Methods = append(report.Methods, result)
	}

	report.TotalTime = time.Since(started)
	return report
}

func RenderComparisonJSON(report *ComparisonReport) ([]byte, error) {
	type jsonState struct {
		ID      int               `json:"id"`
		Actions map[string]string `json:"actions"`
		Gotos   map[string]int    `json:"gotos"`
	}

	type jsonResult struct {
		Method       string      `json:"method"`
		Accepted     *bool       `json:"accepted"`
		Error        string      `json:"error,omitempty"`
		DurationMS   float64     `json:"duration_ms"`
		Terminals    []string    `json:"terminals"`
		NonTerminals []string    `json:"non_terminals"`
		States       []jsonState `json:"states"`
	}

	type payload struct {
		TotalTimeMS float64      `json:"total_time_ms"`
		Methods     []jsonResult `json:"methods"`
	}

	if report == nil {
		return json.Marshal(payload{})
	}

	out := payload{TotalTimeMS: durationMilliseconds(report.TotalTime), Methods: make([]jsonResult, 0, len(report.Methods))}
	for _, result := range report.Methods {
		jsonResult := jsonResult{
			Method:     string(result.Method),
			Accepted:   result.Accepted,
			Error:      result.Error,
			DurationMS: durationMilliseconds(result.Duration),
		}
		if result.Report != nil && result.Report.Table != nil {
			jsonResult.Terminals = append([]string(nil), result.Report.Table.Terminals()...)
			jsonResult.NonTerminals = append([]string(nil), result.Report.Table.NonTerminals()...)
			states := result.Report.Table.States()
			jsonResult.States = make([]jsonState, 0, len(states))
			for _, state := range states {
				row := jsonState{ID: state, Actions: map[string]string{}, Gotos: map[string]int{}}
				for _, terminal := range jsonResult.Terminals {
					if action := renderActionCell(result.Report.Table, state, terminal); action != "" {
						row.Actions[terminal] = action
					}
				}
				for _, nonTerminal := range jsonResult.NonTerminals {
					if target, ok := result.Report.Table.GotoAt(state, nonTerminal); ok {
						row.Gotos[nonTerminal] = target
					}
				}
				jsonResult.States = append(jsonResult.States, row)
			}
		}
		out.Methods = append(out.Methods, jsonResult)
	}

	return json.Marshal(out)
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func (r MethodResult) String() string {
	if r.Error != "" {
		return fmt.Sprintf("%s: %s", r.Method, r.Error)
	}
	if r.Accepted == nil {
		return fmt.Sprintf("%s: ok", r.Method)
	}
	return fmt.Sprintf("%s: accepted=%t", r.Method, *r.Accepted)
}
