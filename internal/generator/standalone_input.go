package generator

// standaloneInputTemplate is shared source for explicit lexer-process input.
// Each parser retains its original loadTokens implementation for -tokens.
const standaloneInputTemplate = `
func loadStandaloneInput(tokensPath, lexerPath, sourcePath string) ([]Token, error) {
	provided := make(map[string]bool)
	flag.Visit(func(option *flag.Flag) { provided[option.Name] = true })
	if provided["tokens"] {
		if provided["lexer"] || provided["src"] {
			return nil, fmt.Errorf("-tokens cannot be combined with -lexer or -src")
		}
		return loadTokens(tokensPath)
	}
	if !provided["lexer"] || !provided["src"] || lexerPath == "" || sourcePath == "" {
		return nil, fmt.Errorf("-lexer and -src must be provided together")
	}
	// Arguments are passed directly; lexerPath is an explicit trusted executable.
	output, err := exec.Command(lexerPath, "-src", sourcePath, "-format", "json").Output()
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok && len(failure.Stderr) > 0 {
			return nil, fmt.Errorf("lexer failed: %s", strings.TrimSpace(string(failure.Stderr)))
		}
		return nil, fmt.Errorf("lexer failed: %w", err)
	}
	var tokens []Token
	if err := json.Unmarshal(output, &tokens); err != nil {
		return nil, fmt.Errorf("decode lexer JSON: %w", err)
	}
	return tokens, nil
}
`
