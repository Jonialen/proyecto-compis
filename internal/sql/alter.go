package sql

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// alterTable applies schema-only changes to the analyzer's shadow catalog.
func (a *analyzer) alterTable(s *AlterTable) {
	t := a.cat.Table(s.Table.Name)
	if t == nil {
		a.errAt(s.Table.Pos, "La tabla %q no existe", s.Table.Name)
		return
	}
	switch s.Action {
	case "ADD":
		// Use CREATE's column validation, including key count and type limits.
		defs := make([]ColumnDef, 0, len(t.Columns)+1)
		for _, c := range t.Columns {
			defs = append(defs, ColumnDef{
				Name:       Ident{Name: c.Name, Pos: s.Pos},
				Type:       DataType{Kind: c.Type.Kind, Length: c.Type.Length, Pos: s.Pos},
				PrimaryKey: c.PrimaryKey, NotNull: c.NotNull,
			})
		}
		// The shadow catalog has no other tables, so foreign keys are checked
		// afterwards against the real one and existing ones are carried over.
		added := s.Column
		added.References = nil
		defs = append(defs, added)
		shadow := NewCatalog()
		check := &analyzer{cat: shadow, stmt: a.stmt}
		check.createTable(&CreateTable{Pos: s.Pos, Name: s.Table, Columns: defs})
		if len(check.diags) != 0 {
			a.diags = append(a.diags, check.diags...)
			return
		}
		next := *shadow.Table(t.Name)
		next.Name = t.Name
		for i, c := range t.Columns {
			next.Columns[i].References = c.References
		}
		if s.Column.References != nil && !applyForeignKeys(&next, []ForeignKeyDef{*s.Column.References}, a.cat, a.errAt) {
			return
		}
		t.Columns = next.Columns
	case "DROP":
		i := t.columnIndex(s.Name.Name)
		if i < 0 {
			a.errAt(s.Name.Pos, "La columna %q no existe en la tabla %q", s.Name.Name, t.Name)
			return
		}
		if len(t.Columns) == 1 {
			a.errAt(s.Name.Pos, "No se puede eliminar la última columna de la tabla %q", t.Name)
			return
		}
		// A column's own FOREIGN KEY is dropped with it; a referenced key is not.
		for _, l := range a.cat.referencesTo(t.Name) {
			if l.key == i && !(l.child == t && l.col == i) {
				a.errAt(s.Name.Pos, "No se puede eliminar la columna %q de la tabla %q: la FOREIGN KEY %s la referencia; elimine antes esa tabla o columna", t.Columns[i].Name, t.Name, l)
				return
			}
		}
		for _, key := range append([]string(nil), a.cat.idxOrd...) {
			ix := a.cat.Indexes[key]
			if strings.EqualFold(ix.Table, t.Name) && strings.EqualFold(ix.Column, s.Name.Name) {
				a.cat.DropIndex(ix.Name)
			}
		}
		t.Columns = append(t.Columns[:i:i], t.Columns[i+1:]...)
	case "RENAME":
		if other := a.cat.Table(s.Name.Name); other != nil && other != t {
			a.errAt(s.Name.Pos, "La tabla %q ya existe", s.Name.Name)
			return
		}
		a.cat.RenameTable(t.Name, s.Name.Name)
	}
}

func (e *Engine) alterTable(s *AlterTable, cat *Catalog) (StmtResult, error) {
	next := cat.Clone()
	a := &analyzer{cat: next}
	a.alterTable(s)
	if len(a.diags) != 0 {
		return StmtResult{}, fmt.Errorf("%s", a.diags[0].Message)
	}
	old := cat.Table(s.Table.Name)
	t := next.Table(s.Table.Name)
	if s.Action == "RENAME" {
		t = next.Table(s.Name.Name)
	}
	rows, err := readRows(e.Dir, old)
	if err != nil {
		return StmtResult{}, err
	}
	// Check persisted constraints before dropping or transforming any data.
	for _, row := range rows {
		for i, c := range old.Columns {
			if _, err := coerce(row[i], c, s.Pos); err != nil {
				return StmtResult{}, err
			}
		}
	}
	if err := checkPrimaryKey(old, rows); err != nil {
		return StmtResult{}, err
	}
	switch s.Action {
	case "ADD":
		if len(rows) > 0 && t.Columns[len(t.Columns)-1].NotNull {
			return StmtResult{}, fmt.Errorf("No se puede agregar la columna %q con NOT NULL a la tabla %q con registros", s.Column.Name.Name, old.Name)
		}
		for i := range rows {
			rows[i] = append(rows[i], nil)
		}
	case "DROP":
		ci := old.columnIndex(s.Name.Name)
		for i, row := range rows {
			rows[i] = append(row[:ci:ci], row[ci+1:]...)
		}
	}
	oldFile, newFile := tableFile(e.Dir, old), tableFile(e.Dir, t)
	if oldFile != newFile {
		if _, err := os.Lstat(newFile); err == nil {
			return StmtResult{}, fmt.Errorf("El archivo de destino de la tabla %q ya existe", t.Name)
		} else if !os.IsNotExist(err) {
			return StmtResult{}, err
		}
	}

	// Generate complete files with existing encoders/index builders. Renaming
	// the staged CSV preserves its timestamp, so index offsets/stamps stay valid.
	stage, err := os.MkdirTemp(e.Dir, ".alter-*")
	if err != nil {
		return StmtResult{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(stage)
		}
	}()
	if err := writeRows(stage, t, rows); err != nil {
		return StmtResult{}, err
	}
	files := []alterFile{{name: filepath.Base(newFile), replace: true}}
	for _, key := range next.idxOrd {
		ix := next.Indexes[key]
		if strings.EqualFold(ix.Table, t.Name) {
			if err := buildIndex(stage, t, ix); err != nil {
				return StmtResult{}, err
			}
			files = append(files, alterFile{name: filepath.Base(indexFile(stage, ix.Name)), replace: true})
		}
	}
	for _, key := range cat.idxOrd {
		ix := cat.Indexes[key]
		if strings.EqualFold(ix.Table, old.Name) && next.Indexes[key] == nil {
			files = append(files, alterFile{name: filepath.Base(indexFile(e.Dir, ix.Name))})
		}
	}
	if oldFile != newFile {
		files = append(files, alterFile{name: filepath.Base(oldFile)})
	}
	if err := next.Save(stage); err != nil {
		return StmtResult{}, err
	}
	files = append(files, alterFile{name: metadataFile, replace: true})
	if err := installAlter(e.Dir, stage, files); err != nil {
		// Do not discard an original if recovery itself was interrupted.
		left, readErr := os.ReadDir(filepath.Join(stage, "backup"))
		if len(left) > 0 || (readErr != nil && !os.IsNotExist(readErr)) {
			cleanup = false
			return StmtResult{}, fmt.Errorf("%w; archivos de recuperación en %q", err, stage)
		}
		return StmtResult{}, err
	}
	*cat = *next
	message := fmt.Sprintf("Tabla %q modificada", old.Name)
	if s.Action == "RENAME" {
		message = fmt.Sprintf("Tabla %q renombrada a %q", old.Name, t.Name)
	}
	return StmtResult{Kind: "ALTER TABLE", Message: message}, nil
}

type alterFile struct {
	name    string
	replace bool
	existed bool
}

// installAlter preflights every target, retains originals by rename, and
// restores them in reverse order on ordinary installation failures. Backups
// preserve bytes and mtimes, including the stamps of the original indexes.
func installAlter(dir, stage string, files []alterFile) error {
	for i := range files {
		info, err := os.Lstat(filepath.Join(dir, files[i].name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("El archivo %q no es un archivo regular", files[i].name)
		}
		files[i].existed = true
	}
	backup := filepath.Join(stage, "backup")
	if err := os.Mkdir(backup, 0700); err != nil {
		return err
	}
	installed := 0
	rollback := func(cause error) error {
		for i := installed - 1; i >= 0; i-- {
			f := files[i]
			dest := filepath.Join(dir, f.name)
			if f.existed {
				if err := os.Rename(filepath.Join(backup, f.name), dest); err != nil {
					cause = errors.Join(cause, fmt.Errorf("No se pudo restaurar %q: %w", f.name, err))
				}
			} else if f.replace {
				if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
					cause = errors.Join(cause, err)
				}
			}
		}
		return cause
	}
	for _, f := range files {
		dest := filepath.Join(dir, f.name)
		if f.existed {
			if err := os.Rename(dest, filepath.Join(backup, f.name)); err != nil {
				return rollback(err)
			}
		}
		installed++
		if f.replace {
			if err := os.Rename(filepath.Join(stage, f.name), dest); err != nil {
				return rollback(err)
			}
		}
	}
	return nil
}
