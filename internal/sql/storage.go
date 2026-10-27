package sql

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func writeRows(dir string, t *Table, rows [][]any) error {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		header[i] = c.Name
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		rec := make([]string, len(row))
		for i, v := range row {
			rec[i] = encodeCell(v)
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return writeFileAtomic(tableFile(dir, t), buf.Bytes())
}

func readRows(dir string, t *Table) ([][]any, error) {
	data, err := os.ReadFile(tableFile(dir, t))
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la tabla %q: %w", t.Name, err)
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = len(t.Columns)
	recs, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("archivo de la tabla %q corrupto: %w", t.Name, err)
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("archivo de la tabla %q sin encabezado", t.Name)
	}
	for i, c := range t.Columns {
		if !strings.EqualFold(recs[0][i], c.Name) {
			return nil, fmt.Errorf("el encabezado de %q no coincide con el esquema (columna %q)", t.Name, c.Name)
		}
	}
	rows := make([][]any, 0, len(recs)-1)
	for n, rec := range recs[1:] {
		row := make([]any, len(rec))
		for i, cell := range rec {
			v, err := decodeCell(cell, t.Columns[i].Type)
			if err != nil {
				return nil, fmt.Errorf("tabla %q registro %d columna %q: %v", t.Name, n+1, t.Columns[i].Name, err)
			}
			row[i] = v
		}
		rows = append(rows, row)
	}
	return rows, nil
}
