package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	compiscript "genanalex/internal/compiscript"
	"genanalex/internal/lexbuild"
	"genanalex/internal/shared"
	"genanalex/internal/yapar"
)

func main() {
	addr := ":8080"
	fmt.Printf("[*] Furlantran running at http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, newMux()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("web")))
	mux.HandleFunc("/api/process", cors(handleProcess))
	mux.HandleFunc("/api/compiscript/analyze", cors(handleCompiscriptAnalyze))
	mux.HandleFunc("/api/health", cors(handleHealth))
	return mux
}

type compiscriptRequest struct {
	Source *string `json:"source"`
}

func handleCompiscriptAnalyze(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeErr(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req compiscriptRequest
	if err := decoder.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Source == nil {
		writeErr(w, http.StatusBadRequest, "source is required")
		return
	}
	json.NewEncoder(w).Encode(compiscript.Analyze([]byte(*req.Source)))
}

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── Request / Response types ──────────────────────────────────────────────────

type processRequest struct {
	YalexContent string   `json:"yalex_content"`
	YaparContent string   `json:"yapar_content"`
	InputContent string   `json:"input_content"`
	Methods      []string `json:"methods"`
}

type tokenInfo struct {
	Type   string `json:"type"`
	Lexeme string `json:"lexeme"`
	Line   int    `json:"line"`
}

type methodResult struct {
	TableJSON    json.RawMessage `json:"table_json,omitempty"`
	TableText    string          `json:"table_text,omitempty"`
	AutomatonDOT string          `json:"automaton_dot,omitempty"`
	Accepted     *bool           `json:"accepted"`
	Error        string          `json:"error,omitempty"`
}

type processResponse struct {
	Tokens        []tokenInfo              `json:"tokens"`
	LexicalErrors []string                 `json:"lexical_errors"`
	Methods       map[string]*methodResult `json:"methods"`
	GrammarError  string                   `json:"grammar_error,omitempty"`
	Success       bool                     `json:"success"`
}

// ── Main handler ──────────────────────────────────────────────────────────────

func handleProcess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req processRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if strings.TrimSpace(req.YaparContent) == "" {
		writeErr(w, http.StatusBadRequest, "yapar_content is required")
		return
	}

	resp := &processResponse{Methods: make(map[string]*methodResult)}

	// Parse .yalp grammar
	yalpPath, cleanYalp, err := tempFile(req.YaparContent, "ide-*.yalp")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanYalp()

	spec, err := yapar.ParseFile(yalpPath)
	if err != nil {
		resp.GrammarError = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	grammar, err := yapar.BuildGrammar(spec)
	if err != nil {
		resp.GrammarError = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	ff, err := yapar.ComputeFirstFollow(grammar)
	if err != nil {
		resp.GrammarError = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Tokenize input when both .yal and input are provided
	var tokens []shared.Token
	if strings.TrimSpace(req.YalexContent) != "" && strings.TrimSpace(req.InputContent) != "" {
		tokens, resp.LexicalErrors, err = tokenize(req.YalexContent, req.InputContent)
		if err != nil {
			resp.GrammarError = err.Error()
			json.NewEncoder(w).Encode(resp)
			return
		}
		for _, tok := range tokens {
			resp.Tokens = append(resp.Tokens, tokenInfo{Type: tok.Type, Lexeme: tok.Lexeme, Line: tok.Line})
		}
	}

	// Build visualization + parse result for each method
	for _, method := range resolveMethodList(req.Methods) {
		result := &methodResult{}

		report, err := yapar.BuildVisReport(grammar, ff, method)
		if err != nil {
			result.Error = err.Error()
			resp.Methods[string(method)] = result
			continue
		}

		if tableJSON, err := yapar.RenderTableJSON(report); err == nil {
			result.TableJSON = json.RawMessage(tableJSON)
		}
		result.TableText = yapar.RenderTableText(report)

		if dot, err := yapar.RenderAutomatonDOT(report); err == nil {
			result.AutomatonDOT = dot
		}

		if len(tokens) > 0 {
			parser, buildErr := yapar.BuildParser(grammar, ff, method)
			if buildErr != nil {
				result.Error = buildErr.Error()
			} else {
				parseResult, parseErr := parser.Parse(tokens)
				if parseErr != nil {
					errStr := parseErr.Error()
					result.Error = errStr
					no := false
					result.Accepted = &no
				} else if parseResult != nil {
					result.Accepted = &parseResult.Accepted
				}
			}
		}

		resp.Methods[string(method)] = result
	}

	resp.Success = true
	json.NewEncoder(w).Encode(resp)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func tokenize(yalContent, inputContent string) ([]shared.Token, []string, error) {
	yalPath, cleanYal, err := tempFile(yalContent, "ide-*.yal")
	if err != nil {
		return nil, nil, fmt.Errorf("write .yal temp file: %w", err)
	}
	defer cleanYal()

	inputPath, cleanInput, err := tempFile(inputContent, "ide-*.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("write input temp file: %w", err)
	}
	defer cleanInput()

	lexResult, err := lexbuild.CompileYALFile(yalPath, false)
	if err != nil {
		return nil, nil, fmt.Errorf("compile .yal: %w", err)
	}

	toks, lexErrs, err := lexbuild.TokenizeFile(lexResult.DFAEntries, inputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("tokenize: %w", err)
	}
	return toks, lexErrs, nil
}

func resolveMethodList(raw []string) []yapar.Method {
	if len(raw) == 0 {
		return []yapar.Method{yapar.MethodLL1, yapar.MethodSLR, yapar.MethodLALR}
	}
	var out []yapar.Method
	for _, s := range raw {
		if m, err := yapar.ParseMethod(strings.ToLower(strings.TrimSpace(s))); err == nil {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return []yapar.Method{yapar.MethodSLR}
	}
	return out
}

func tempFile(content, pattern string) (string, func(), error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", nil, err
	}
	name := f.Name()
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(name)
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", nil, err
	}
	return name, func() { os.Remove(name) }, nil
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
