package frontend

import (
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
)

func TestSourceIndexConvertsUTF8ScalarsToHalfOpenByteSpans(t *testing.T) {
	index := newSourceIndex([]byte("éx\n😄"))

	for _, test := range []struct {
		name       string
		start, end int
		want       ast.Span
	}{
		{
			name:  "two byte scalar",
			start: 0,
			end:   0,
			want:  ast.Span{Start: ast.Position{Offset: 0, Line: 1, Column: 1}, End: ast.Position{Offset: 2, Line: 1, Column: 2}},
		},
		{
			name:  "four byte scalar on second line",
			start: 3,
			end:   3,
			want:  ast.Span{Start: ast.Position{Offset: 4, Line: 2, Column: 1}, End: ast.Position{Offset: 8, Line: 2, Column: 2}},
		},
		{
			name:  "zero width end of input",
			start: 4,
			end:   3,
			want:  ast.Span{Start: ast.Position{Offset: 8, Line: 2, Column: 2}, End: ast.Position{Offset: 8, Line: 2, Column: 2}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := index.spanFromScalars(test.start, test.end); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("spanFromScalars(%d, %d) = %#v, want %#v", test.start, test.end, got, test.want)
			}
		})
	}
}

func TestSourceIndexUsesOneBasedLineAndColumnCoordinates(t *testing.T) {
	index := newSourceIndex([]byte("a\nβ"))

	if got, want := index.positionAtScalar(2), (ast.Position{Offset: 2, Line: 2, Column: 1}); got != want {
		t.Fatalf("positionAtScalar(2) = %#v, want %#v", got, want)
	}
	if got, want := index.positionAtScalar(3), (ast.Position{Offset: 4, Line: 2, Column: 2}); got != want {
		t.Fatalf("positionAtScalar(3) = %#v, want %#v", got, want)
	}
}
