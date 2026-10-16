package yapar

import (
	"errors"
	"fmt"
	"testing"
)

func TestParseFormatErrorIsNotGrammarError(t *testing.T) {
	spec, err := Parse("%token ID\nstart : ID ;")
	if err == nil || spec != nil {
		t.Fatalf("Parse() = (%#v, %v), want no spec and a format error", spec, err)
	}
	wrapped := fmt.Errorf("parse spec: %w", err)
	var specErr *SpecError
	if !errors.As(wrapped, &specErr) {
		t.Fatalf("error type = %T, want *SpecError", err)
	}
	var grammarErr *GrammarError
	if errors.As(wrapped, &grammarErr) {
		t.Fatal("format error must not be classified as a grammar error")
	}
}

func TestGrammarErrorFormatting(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  *GrammarError
		want string
	}{
		{name: "nil", want: "<nil>"},
		{name: "message", err: &GrammarError{Message: "nil spec"}, want: "yapar spec: nil spec"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
