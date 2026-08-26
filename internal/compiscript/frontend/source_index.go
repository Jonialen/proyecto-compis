package frontend

import (
	"unicode/utf8"

	"genanalex/internal/compiscript/ast"
)

type sourceIndex struct {
	positions []ast.Position
}

func newSourceIndex(source []byte) sourceIndex {
	positions := []ast.Position{{Offset: 0, Line: 1, Column: 1}}
	offset, line, column := 0, 1, 1
	for len(source) > 0 {
		runeValue, size := utf8.DecodeRune(source)
		offset += size
		if runeValue == '\n' {
			line++
			column = 1
		} else {
			column++
		}
		positions = append(positions, ast.Position{Offset: offset, Line: line, Column: column})
		source = source[size:]
	}
	return sourceIndex{positions: positions}
}

func (i sourceIndex) positionAtScalar(scalar int) ast.Position {
	return i.positions[i.clampScalar(scalar)]
}

func (i sourceIndex) spanFromScalars(start, stop int) ast.Span {
	start = i.clampScalar(start)
	if stop < start {
		position := i.positionAtScalar(start)
		return ast.Span{Start: position, End: position}
	}
	return ast.Span{Start: i.positionAtScalar(start), End: i.positionAtScalar(stop + 1)}
}

func (i sourceIndex) scalarAtLineColumn(line, column int) int {
	for scalar, position := range i.positions {
		if position.Line == line && position.Column >= column+1 {
			return scalar
		}
	}
	return len(i.positions) - 1
}

func (i sourceIndex) clampScalar(scalar int) int {
	if scalar < 0 {
		return 0
	}
	if scalar >= len(i.positions) {
		return len(i.positions) - 1
	}
	return scalar
}
