// Package yapar interpreta especificaciones .yalp sin acoplarse al runtime LR.
package yapar

import (
	"fmt"
	"os"
	"strings"
)

// RawProduction representa una producción cruda aún no tipada formalmente.
type RawProduction struct {
	Head   string
	Bodies [][]string
}

// YaparSpec contiene la salida mínima del parser de especificación .yalp.
type YaparSpec struct {
	Tokens       []string
	IgnoreTokens map[string]bool
	Productions  []RawProduction
	StartSymbol  string
}

// ParseFile carga y parsea un archivo .yalp desde disco.
func ParseFile(path string) (*YaparSpec, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading yalp file: %w", err)
	}
	return Parse(string(content))
}

// Parse interpreta un .yalp y devuelve su representación cruda.
func Parse(content string) (*YaparSpec, error) {
	clean := stripBlockComments(content)
	parts := strings.SplitN(clean, "%%", 2)
	if len(parts) != 2 {
		return nil, &SpecError{Message: "missing %% section separator"}
	}

	spec := &YaparSpec{
		IgnoreTokens: make(map[string]bool),
	}

	parseHeader(parts[0], spec)
	if err := parseProductions(parts[1], spec); err != nil {
		return nil, err
	}

	if len(spec.Productions) == 0 {
		return nil, &SpecError{Message: "no productions declared"}
	}
	if spec.StartSymbol == "" {
		return nil, &SpecError{Message: "missing start symbol"}
	}

	return spec, nil
}

func parseHeader(header string, spec *YaparSpec) {
	seenTokens := make(map[string]bool)
	for _, rawLine := range strings.Split(header, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "%token":
			for _, tok := range fields[1:] {
				if !seenTokens[tok] {
					spec.Tokens = append(spec.Tokens, tok)
					seenTokens[tok] = true
				}
			}
		case "IGNORE":
			for _, tok := range fields[1:] {
				spec.IgnoreTokens[tok] = true
			}
		}
	}
}

func parseProductions(section string, spec *YaparSpec) error {
	chunks := strings.Split(section, ";")
	for _, rawChunk := range chunks {
		chunk := strings.TrimSpace(rawChunk)
		if chunk == "" {
			continue
		}

		head, rest, ok := strings.Cut(chunk, ":")
		if !ok {
			return &SpecError{Message: fmt.Sprintf("invalid production %q", chunk)}
		}

		production := RawProduction{Head: strings.TrimSpace(head)}
		if production.Head == "" {
			return &SpecError{Message: "production head cannot be empty"}
		}
		if spec.StartSymbol == "" {
			spec.StartSymbol = production.Head
		}

		for _, alt := range strings.Split(rest, "|") {
			body := strings.Fields(strings.TrimSpace(alt))
			production.Bodies = append(production.Bodies, body)
		}

		spec.Productions = append(spec.Productions, production)
	}

	return nil
}

func stripBlockComments(content string) string {
	var b strings.Builder
	for len(content) > 0 {
		start := strings.Index(content, "/*")
		if start == -1 {
			b.WriteString(content)
			break
		}
		b.WriteString(content[:start])
		content = content[start+2:]
		end := strings.Index(content, "*/")
		if end == -1 {
			break
		}
		content = content[end+2:]
	}
	return b.String()
}
