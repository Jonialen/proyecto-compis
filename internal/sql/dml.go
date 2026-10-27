package sql

import (
	"fmt"
	"os"
	"strings"
)

func (e *Engine) query(s *Select, cat *Catalog) (StmtResult, error) {
	x := newExecCtx(e.Dir, cat)
	x.noIndex = e.NoIndex
	res, err := x.runSelect(s, nil)
	if err != nil {
		return StmtResult{}, err
	}
	return StmtResult{
		Kind: "SELECT", Columns: res.cols, Rows: res.rows, Plan: x.plan, RowsAffected: len(res.rows),
		Message: fmt.Sprintf("%d fila(s)", len(res.rows)),
	}, nil
}

// matchingOffsets reports, through the index when one applies, which records
// can satisfy the WHERE clause. A nil set means every record must be checked.
func (x *execCtx) matchingOffsets(t *Table, where Expr, total int) (map[int64]bool, error) {
	tabs := []boundTab{{strings.ToLower(t.Name), t}}
	offs, li, ix, ps, err := x.candidateOffsets(t, x.indexPreds(tabs, 0, flattenAnd(where, nil)))
	if err != nil {
		return nil, err
	}
	if ix == nil {
		x.note("Recorrido completo de %q: %d registros", t.Name, total)
		return nil, nil
	}
	x.note("Índice %q usado en %q por %s: %d de %d registros evaluados", ix.Name, t.Name, condText(ix.Column, ps), len(offs), li.total)
	set := make(map[int64]bool, len(offs))
	for _, o := range offs {
		set[o] = true
	}
	return set, nil
}

func checkPrimaryKey(t *Table, rows [][]any) error {
	for pk, c := range t.Columns {
		if !c.PrimaryKey {
			continue
		}
		seen := map[any]bool{}
		for _, r := range rows {
			if seen[r[pk]] {
				return fmt.Errorf("Violación de PRIMARY KEY: el valor %v de la columna %q aparece más de una vez en la tabla %q; no se modificó ninguna fila", r[pk], c.Name, t.Name)
			}
			seen[r[pk]] = true
		}
	}
	return nil
}

func (e *Engine) update(s *Update, cat *Catalog) (StmtResult, error) {
	t := cat.Table(s.Table.Name)
	x := newExecCtx(e.Dir, cat)
	x.noIndex = e.NoIndex
	rows, offs, err := readRowsOffsets(e.Dir, t)
	if err != nil {
		return StmtResult{}, err
	}
	cand, err := x.matchingOffsets(t, s.Where, len(rows))
	if err != nil {
		return StmtResult{}, err
	}
	tabs := []boundTab{{strings.ToLower(t.Name), t}}
	next := make([][]any, len(rows))
	changed := 0
	for i, row := range rows {
		next[i] = row
		if cand != nil && !cand[offs[i]] {
			continue
		}
		en := &env{tabs: tabs, row: jrow{row}}
		if s.Where != nil {
			v, err := x.eval(s.Where, en)
			if err != nil {
				return StmtResult{}, err
			}
			if !isTrue(v) {
				continue
			}
		}
		updated := append([]any(nil), row...)
		for _, as := range s.Sets {
			v, err := x.eval(as.Value, en)
			if err != nil {
				return StmtResult{}, err
			}
			ci := t.columnIndex(as.Column.Name)
			cv, perr := coerce(v, t.Columns[ci], as.Value.Position())
			if perr != nil {
				return StmtResult{}, perr
			}
			updated[ci] = cv
		}
		next[i] = updated
		changed++
	}
	if err := checkPrimaryKey(t, next); err != nil {
		return StmtResult{}, err
	}
	if changed > 0 {
		if err := e.writeTable(cat, t, next); err != nil {
			return StmtResult{}, err
		}
	}
	return StmtResult{Kind: "UPDATE", RowsAffected: changed, Plan: x.plan, Message: fmt.Sprintf("%d fila(s) actualizada(s) en %q", changed, t.Name)}, nil
}

func (e *Engine) deleteRows(s *Delete, cat *Catalog) (StmtResult, error) {
	t := cat.Table(s.Table.Name)
	x := newExecCtx(e.Dir, cat)
	x.noIndex = e.NoIndex
	rows, offs, err := readRowsOffsets(e.Dir, t)
	if err != nil {
		return StmtResult{}, err
	}
	cand, err := x.matchingOffsets(t, s.Where, len(rows))
	if err != nil {
		return StmtResult{}, err
	}
	tabs := []boundTab{{strings.ToLower(t.Name), t}}
	var kept [][]any
	removed := 0
	for i, row := range rows {
		drop := cand == nil || cand[offs[i]]
		if drop && s.Where != nil {
			v, err := x.eval(s.Where, &env{tabs: tabs, row: jrow{row}})
			if err != nil {
				return StmtResult{}, err
			}
			drop = isTrue(v)
		}
		if drop {
			removed++
		} else {
			kept = append(kept, row)
		}
	}
	if removed > 0 {
		if err := e.writeTable(cat, t, kept); err != nil {
			return StmtResult{}, err
		}
	}
	return StmtResult{Kind: "DELETE", RowsAffected: removed, Plan: x.plan, Message: fmt.Sprintf("%d fila(s) eliminada(s) de %q", removed, t.Name)}, nil
}

// writeTable replaces the rows of a table and rebuilds its indexes so they
// always match the file. The table is written first: if the process stops
// before an index is rewritten, the stamp in the index header no longer
// matches and the index is rebuilt the next time it is read.
func (e *Engine) writeTable(cat *Catalog, t *Table, rows [][]any) error {
	if err := writeRows(e.Dir, t, rows); err != nil {
		return err
	}
	for _, key := range cat.idxOrd {
		if ix := cat.Indexes[key]; strings.EqualFold(ix.Table, t.Name) {
			if err := buildIndex(e.Dir, t, ix); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) createIndex(s *CreateIndex, cat *Catalog) (StmtResult, error) {
	t := cat.Table(s.Table.Name)
	ix := &IndexDef{Name: s.Name.Name, Table: t.Name, Column: t.Columns[t.columnIndex(s.Column.Name)].Name}
	// Index file first: a stop before the metadata write leaves an unused
	// file, never a catalog entry without its index.
	if err := buildIndex(e.Dir, t, ix); err != nil {
		return StmtResult{}, err
	}
	cat.AddIndex(ix)
	if err := cat.Save(e.Dir); err != nil {
		return StmtResult{}, err
	}
	return StmtResult{Kind: "CREATE INDEX", Message: fmt.Sprintf("Índice %q creado sobre %s(%s)", ix.Name, t.Name, ix.Column)}, nil
}

func (e *Engine) dropIndex(s *DropIndex, cat *Catalog) (StmtResult, error) {
	ix := cat.Indexes[s.Name.Key()]
	cat.DropIndex(ix.Name)
	if err := cat.Save(e.Dir); err != nil {
		return StmtResult{}, err
	}
	if err := os.Remove(indexFile(e.Dir, ix.Name)); err != nil && !os.IsNotExist(err) {
		return StmtResult{}, err
	}
	return StmtResult{Kind: "DROP INDEX", Message: fmt.Sprintf("Índice %q eliminado", ix.Name)}, nil
}
