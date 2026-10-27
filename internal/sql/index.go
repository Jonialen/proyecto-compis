package sql

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// An index lives in <indexname>.idx. The first record is a header
// ("#INDEX", name, table, column, stamp, rows); every other record is a key
// followed by the byte offsets, inside the table file, of the records that
// hold it, sorted by key. NULL keys are not indexed because no comparison
// matches them. The stamp (table file size and modification time) detects an
// index that does not match its table, which is then rebuilt.

type indexEntry struct {
	key  any
	offs []int64
}

type loadedIndex struct {
	entries []indexEntry
	total   int // records in the table when the index was built
}

type pred struct {
	op string // =, <, <=, >, >= or IN (v is then a []any of non-NULL constants)
	v  any
}

func indexFile(dir, name string) string {
	return filepath.Join(dir, strings.ToLower(name)+".idx")
}

func tableStamp(dir string, t *Table) (string, error) {
	fi, err := os.Stat(tableFile(dir, t))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d-%d", fi.Size(), fi.ModTime().UnixNano()), nil
}

func buildIndex(dir string, t *Table, ix *IndexDef) error {
	ci := t.columnIndex(ix.Column)
	if ci < 0 {
		return fmt.Errorf("el índice %q usa la columna inexistente %q", ix.Name, ix.Column)
	}
	rows, offs, err := readRowsOffsets(dir, t)
	if err != nil {
		return err
	}
	stamp, err := tableStamp(dir, t)
	if err != nil {
		return err
	}
	groups := map[any][]int64{}
	var keys []any
	for i, row := range rows {
		k := row[ci]
		if k == nil {
			continue
		}
		if _, seen := groups[k]; !seen {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], offs[i])
	}
	sort.SliceStable(keys, func(a, b int) bool { return compareValues(keys[a], keys[b]) < 0 })

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{"#INDEX", ix.Name, t.Name, t.Columns[ci].Name, stamp, strconv.Itoa(len(rows))})
	for _, k := range keys {
		rec := []string{encodeCell(k)}
		for _, o := range groups[k] {
			rec = append(rec, strconv.FormatInt(o, 10))
		}
		w.Write(rec)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return writeFileAtomic(indexFile(dir, ix.Name), buf.Bytes())
}

// loadIndex reads an index file, rebuilding it first when it is missing or
// does not match the current table file.
func loadIndex(dir string, t *Table, ix *IndexDef) (*loadedIndex, error) {
	for attempt := 0; ; attempt++ {
		li, ok, err := readIndexFile(dir, t, ix)
		if err != nil {
			return nil, err
		}
		if ok {
			return li, nil
		}
		if attempt > 0 {
			return nil, fmt.Errorf("el índice %q no se pudo reconstruir", ix.Name)
		}
		if err := buildIndex(dir, t, ix); err != nil {
			return nil, err
		}
	}
}

func readIndexFile(dir string, t *Table, ix *IndexDef) (*loadedIndex, bool, error) {
	data, err := os.ReadFile(indexFile(dir, ix.Name))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil || len(header) != 6 || header[0] != "#INDEX" {
		return nil, false, nil // unreadable header: rebuild
	}
	stamp, err := tableStamp(dir, t)
	if err != nil {
		return nil, false, err
	}
	ci := t.columnIndex(ix.Column)
	if !strings.EqualFold(header[1], ix.Name) || !strings.EqualFold(header[2], t.Name) || ci < 0 || !strings.EqualFold(header[3], t.Columns[ci].Name) || header[4] != stamp {
		return nil, false, nil
	}
	total, err := strconv.Atoi(header[5])
	if err != nil {
		return nil, false, nil
	}
	li := &loadedIndex{total: total}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 2 {
			return nil, false, nil
		}
		key, err := decodeCell(rec[0], t.Columns[ci].Type)
		if err != nil {
			return nil, false, nil
		}
		e := indexEntry{key: key}
		for _, o := range rec[1:] {
			off, err := strconv.ParseInt(o, 10, 64)
			if err != nil {
				return nil, false, nil
			}
			e.offs = append(e.offs, off)
		}
		li.entries = append(li.entries, e)
	}
	return li, true, nil
}

// keyMatches reports whether a key satisfies every predicate.
func keyMatches(k any, preds []pred) bool {
	for _, p := range preds {
		if p.op == "IN" {
			found := false
			for _, v := range p.v.([]any) {
				found = found || compareValues(k, v) == 0
			}
			if !found {
				return false
			}
			continue
		}
		c := compareValues(k, p.v)
		switch p.op {
		case "=":
			if c != 0 {
				return false
			}
		case "<":
			if c >= 0 {
				return false
			}
		case "<=":
			if c > 0 {
				return false
			}
		case ">":
			if c <= 0 {
				return false
			}
		case ">=":
			if c < 0 {
				return false
			}
		}
	}
	return true
}

// lookup returns, in table order, the offsets of the records whose key
// satisfies every predicate. Only the keys that can match are visited: the
// listed values of an IN, or the sorted slice between the bounds.
func (li *loadedIndex) lookup(preds []pred) []int64 {
	for i, p := range preds {
		if p.op != "IN" {
			continue
		}
		rest := append(append([]pred{}, preds[:i]...), preds[i+1:]...)
		var out []int64
		for _, v := range p.v.([]any) {
			j := sort.Search(len(li.entries), func(j int) bool { return compareValues(li.entries[j].key, v) >= 0 })
			if j < len(li.entries) && compareValues(li.entries[j].key, v) == 0 && keyMatches(li.entries[j].key, rest) {
				out = append(out, li.entries[j].offs...)
			}
		}
		slices.Sort(out)
		return slices.Compact(out)
	}
	start := 0
	for _, p := range preds {
		if p.op == "=" || p.op == ">" || p.op == ">=" {
			i := sort.Search(len(li.entries), func(i int) bool {
				c := compareValues(li.entries[i].key, p.v)
				if p.op == ">" {
					return c > 0
				}
				return c >= 0
			})
			start = max(start, i)
		}
	}
	var out []int64
	for i := start; i < len(li.entries); i++ {
		k := li.entries[i].key
		// Keys are sorted: once an upper bound or an equality is passed, stop.
		stop := false
		for _, p := range preds {
			c := compareValues(k, p.v)
			stop = stop || (p.op == "=" && c > 0) || (p.op == "<" && c >= 0) || (p.op == "<=" && c > 0)
		}
		if stop {
			break
		}
		if keyMatches(k, preds) {
			out = append(out, li.entries[i].offs...)
		}
	}
	slices.Sort(out)
	return out
}
