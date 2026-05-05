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
	clean, err := stripBlockComments(content)
	if err != nil {
		return nil, err
	}

	header, productions, lineOffset, err := splitSections(clean)
	if err != nil {
		return nil, err
	}

	spec := &YaparSpec{IgnoreTokens: make(map[string]bool)}
	if err := parseHeader(header, spec); err != nil {
		return nil, err
	}
	if err := parseProductions(productions, lineOffset, spec); err != nil {
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

func parseHeader(header string, spec *YaparSpec) error {
	seenTokens := make(map[string]bool)
	for index, rawLine := range strings.Split(header, "\n") {
		lineNumber := index + 1
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
			if len(fields) == 1 {
				return &SpecError{Line: lineNumber, Message: "%token requires at least one token name"}
			}
			for _, tok := range fields[1:] {
				if seenTokens[tok] {
					return &SpecError{Line: lineNumber, Message: fmt.Sprintf("token %q declared more than once", tok)}
				}
				spec.Tokens = append(spec.Tokens, tok)
				seenTokens[tok] = true
			}
		case "IGNORE":
			if len(fields) == 1 {
				return &SpecError{Line: lineNumber, Message: "IGNORE requires at least one token name"}
			}
			for _, tok := range fields[1:] {
				spec.IgnoreTokens[tok] = true
			}
		default:
			return &SpecError{Line: lineNumber, Message: fmt.Sprintf("unexpected header directive %q", fields[0])}
		}
	}

	return nil
}

func parseProductions(section string, lineOffset int, spec *YaparSpec) error {
	parser := productionParser{scanner: newSpecScanner(section, lineOffset), spec: spec}
	return parser.parse()
}

func splitSections(content string) (header string, productions string, lineOffset int, err error) {
	separator := strings.Index(content, "%%")
	if separator == -1 {
		return "", "", 0, &SpecError{Message: "missing %% section separator"}
	}
	if second := strings.Index(content[separator+2:], "%%"); second != -1 {
		absolute := separator + 2 + second
		line := 1 + strings.Count(content[:absolute], "\n")
		return "", "", 0, &SpecError{Line: line, Message: "multiple %% section separators are not allowed"}
	}

	header = content[:separator]
	productions = content[separator+2:]
	lineOffset = 1 + strings.Count(header, "\n")
	return header, productions, lineOffset, nil
}

func stripBlockComments(content string) (string, error) {
	var builder strings.Builder
	line := 1

	for i := 0; i < len(content); {
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '*' {
			commentLine := line
			i += 2
			closed := false
			for i < len(content) {
				if i+1 < len(content) && content[i] == '*' && content[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				if content[i] == '\n' {
					builder.WriteByte('\n')
					line++
				}
				i++
			}
			if !closed {
				return "", &SpecError{Line: commentLine, Message: "unterminated block comment"}
			}
			continue
		}

		builder.WriteByte(content[i])
		if content[i] == '\n' {
			line++
		}
		i++
	}

	return builder.String(), nil
}

type specScanner struct {
	input []rune
	pos   int
	line  int
}

type specToken struct {
	kind string
	text string
	line int
}

func newSpecScanner(input string, lineOffset int) *specScanner {
	return &specScanner{input: []rune(input), line: lineOffset}
}

func (s *specScanner) next() specToken {
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		if ch == '\n' {
			s.line++
			s.pos++
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\r' {
			s.pos++
			continue
		}

		line := s.line
		s.pos++
		switch ch {
		case ':', '|', ';':
			return specToken{kind: string(ch), text: string(ch), line: line}
		default:
			start := s.pos - 1
			for s.pos < len(s.input) {
				next := s.input[s.pos]
				if next == '\n' || next == ' ' || next == '\t' || next == '\r' || next == ':' || next == '|' || next == ';' {
					break
				}
				s.pos++
			}
			return specToken{kind: "symbol", text: string(s.input[start:s.pos]), line: line}
		}
	}

	return specToken{kind: "eof", line: s.line}
}

type productionParser struct {
	scanner *specScanner
	spec    *YaparSpec
	look    specToken
	hasLook bool
}

func (p *productionParser) parse() error {
	for {
		tok := p.peek()
		switch tok.kind {
		case "eof":
			return nil
		case ";", "|", ":":
			return &SpecError{Line: tok.line, Message: fmt.Sprintf("unexpected %q before production head", tok.text)}
		case "symbol":
			if err := p.parseProduction(); err != nil {
				return err
			}
		default:
			return &SpecError{Line: tok.line, Message: fmt.Sprintf("unexpected token %q", tok.text)}
		}
	}
}

func (p *productionParser) parseProduction() error {
	head := p.consume()
	if head.kind != "symbol" || strings.TrimSpace(head.text) == "" {
		return &SpecError{Line: head.line, Message: "production head cannot be empty"}
	}

	colon := p.consume()
	if colon.kind != ":" {
		if colon.kind == "eof" {
			return &SpecError{Line: head.line, Message: fmt.Sprintf("missing ':' after production head %q", head.text)}
		}
		return &SpecError{Line: colon.line, Message: fmt.Sprintf("expected ':' after production head %q, got %q", head.text, colon.text)}
	}

	production := RawProduction{Head: head.text}
	if p.spec.StartSymbol == "" {
		p.spec.StartSymbol = head.text
	}

	for {
		body, err := p.parseAlternative(head.text)
		if err != nil {
			return err
		}
		production.Bodies = append(production.Bodies, body)

		separator := p.consume()
		switch separator.kind {
		case "|":
			continue
		case ";":
			p.spec.Productions = append(p.spec.Productions, production)
			return nil
		case "eof":
			return &SpecError{Line: head.line, Message: fmt.Sprintf("production %q must end with ';'", head.text)}
		default:
			return &SpecError{Line: separator.line, Message: fmt.Sprintf("expected '|' or ';' after production %q, got %q", head.text, separator.text)}
		}
	}
}

func (p *productionParser) parseAlternative(head string) ([]string, error) {
	body := make([]string, 0)
	for {
		tok := p.peek()
		switch tok.kind {
		case "symbol":
			body = append(body, p.consume().text)
		case ":":
			return nil, &SpecError{Line: tok.line, Message: fmt.Sprintf("unexpected ':' inside production %q", head)}
		case "|", ";", "eof":
			return body, nil
		default:
			return nil, &SpecError{Line: tok.line, Message: fmt.Sprintf("unexpected token %q inside production %q", tok.text, head)}
		}
	}
}

func (p *productionParser) peek() specToken {
	if !p.hasLook {
		p.look = p.scanner.next()
		p.hasLook = true
	}
	return p.look
}

func (p *productionParser) consume() specToken {
	tok := p.peek()
	p.hasLook = false
	return tok
}
