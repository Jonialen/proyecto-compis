package lexer

import (
	"fmt"
	"os"
	"strings"
)

// SourceFile holds the content and metadata of an input source file.
type SourceFile struct {
	Path    string
	Content string
	Lines   []string
}

// ReadSource reads a source file and returns a SourceFile.
// Normalizes \r\n → \n.
func ReadSource(path string) (*SourceFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading source file %q: %w", path, err)
	}

	content := string(data)
	// Normalize \r\n → \n
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Also normalize lone \r → \n
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")

	return &SourceFile{
		Path:    path,
		Content: content,
		Lines:   lines,
	}, nil
}
