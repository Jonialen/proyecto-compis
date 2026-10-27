package sql

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Rows are stored one file per table: dir/<table>.csv with a header line of
// column names. NULL is written as \N; text starting with a backslash gets
// an extra backslash so it cannot be confused with the NULL marker.

const nullMarker = `\N`

func tableFile(dir string, t *Table) string {
	return filepath.Join(dir, strings.ToLower(t.Name)+".csv")
}

func encodeCell(v any) string {
	switch x := v.(type) {
	case nil:
		return nullMarker
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	case string:
		if strings.HasPrefix(x, `\`) {
			return `\` + x
		}
		return x
	}
	return fmt.Sprint(v)
}

func decodeCell(s string, ct ColType) (any, error) {
	if s == nullMarker {
		return nil, nil
	}
	switch ct.Kind {
	case "INT":
		return strconv.ParseInt(s, 10, 64)
	case "FLOAT":
		return strconv.ParseFloat(s, 64)
	case "BOOLEAN":
		switch s {
		case "TRUE":
			return true, nil
		case "FALSE":
			return false, nil
		}
		return nil, fmt.Errorf("booleano inválido %q", s)
	case "DATE":
		if !validDate(s) {
			return nil, fmt.Errorf("fecha inválida %q", s)
		}
		return s, nil
	}
	return strings.TrimPrefix(s, `\`), nil
}

// csvField quotes a cell when needed. An empty string is always written as
// "" so a one-column record is never a blank line, which readers skip.
func csvField(s string) string {
	if s == "" {
		return `""`
	}
	first, _ := utf8.DecodeRuneInString(s)
	if strings.ContainsAny(s, ",\"\r\n") || unicode.IsSpace(first) {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func writeRows(dir string, t *Table, rows [][]any) error {
	var buf bytes.Buffer
	line := func(cells []string) {
		for i, c := range cells {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(csvField(c))
		}
		buf.WriteByte('\n')
	}
	header := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		header[i] = c.Name
	}
	line(header)
	for _, row := range rows {
		rec := make([]string, len(row))
		for i, v := range row {
			rec[i] = encodeCell(v)
		}
		line(rec)
	}
	return writeFileAtomic(tableFile(dir, t), buf.Bytes())
}

// readRowsOffsets reads a whole table and the byte offset at which each
// record starts, so an index can later fetch single records by position.
func readRowsOffsets(dir string, t *Table) ([][]any, []int64, error) {
	data, err := os.ReadFile(tableFile(dir, t))
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo leer la tabla %q: %w", t.Name, err)
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = len(t.Columns)
	header, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("archivo de la tabla %q sin encabezado o corrupto: %w", t.Name, err)
	}
	for i, c := range t.Columns {
		if !strings.EqualFold(header[i], c.Name) {
			return nil, nil, fmt.Errorf("el encabezado de %q no coincide con el esquema (columna %q)", t.Name, c.Name)
		}
	}
	var rows [][]any
	var offsets []int64
	for {
		off := r.InputOffset()
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("archivo de la tabla %q corrupto: %w", t.Name, err)
		}
		row, err := decodeRecord(t, rec, len(rows)+1)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, row)
		offsets = append(offsets, off)
	}
	return rows, offsets, nil
}

func decodeRecord(t *Table, rec []string, n int) ([]any, error) {
	row := make([]any, len(rec))
	for i, cell := range rec {
		v, err := decodeCell(cell, t.Columns[i].Type)
		if err != nil {
			return nil, fmt.Errorf("tabla %q registro %d columna %q: %v", t.Name, n, t.Columns[i].Name, err)
		}
		row[i] = v
	}
	return row, nil
}

func readRows(dir string, t *Table) ([][]any, error) {
	rows, _, err := readRowsOffsets(dir, t)
	return rows, err
}

// readRowsAt reads only the records that start at the given offsets.
func readRowsAt(dir string, t *Table, offsets []int64) ([][]any, error) {
	f, err := os.Open(tableFile(dir, t))
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la tabla %q: %w", t.Name, err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(offsets))
	for _, off := range offsets {
		r := csv.NewReader(io.NewSectionReader(f, off, fi.Size()-off))
		r.FieldsPerRecord = len(t.Columns)
		rec, err := r.Read()
		if err != nil {
			return nil, fmt.Errorf("archivo de la tabla %q corrupto en el byte %d: %w", t.Name, off, err)
		}
		row, err := decodeRecord(t, rec, len(rows)+1)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}
