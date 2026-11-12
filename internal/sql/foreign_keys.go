package sql

import (
	"fmt"
	"strings"
)

// Foreign keys are single-column and always restrictive: a write that would
// leave a child value without its parent key is rejected before any file is
// written. There is no CASCADE and no SET NULL.
//
// Checks use the state at the end of the statement, so one statement may
// both remove a key and every row that references it (for example a
// self-referencing table emptied by DELETE FROM).

// fkLink is one FOREIGN KEY: child.Columns[col] references parent.Columns[key].
type fkLink struct {
	child  *Table
	col    int
	parent *Table
	key    int
}

func (l fkLink) String() string { return l.child.Name + "." + l.child.Columns[l.col].Name }

// links lists every resolvable foreign key in table order.
func (c *Catalog) links() []fkLink {
	var out []fkLink
	for _, t := range c.TableList() {
		for i, col := range t.Columns {
			if col.References == nil {
				continue
			}
			parent := c.Table(col.References.Table)
			if parent == nil {
				continue
			}
			if key := parent.columnIndex(col.References.Column); key >= 0 {
				out = append(out, fkLink{child: t, col: i, parent: parent, key: key})
			}
		}
	}
	return out
}

// referencesTo lists the foreign keys whose parent is the named table.
func (c *Catalog) referencesTo(name string) []fkLink {
	var out []fkLink
	for _, l := range c.links() {
		if strings.EqualFold(l.parent.Name, name) {
			out = append(out, l)
		}
	}
	return out
}

// dropBlocker returns a foreign key of another table that references name.
// A table's own self-reference is dropped with it, so it does not block.
func (c *Catalog) dropBlocker(name string) (fkLink, bool) {
	for _, l := range c.referencesTo(name) {
		if !strings.EqualFold(l.child.Name, name) {
			return l, true
		}
	}
	return fkLink{}, false
}

func dropTableBlockedMsg(name string, l fkLink) string {
	return fmt.Sprintf("No se puede eliminar la tabla %q: la FOREIGN KEY %s la referencia; elimine antes esa tabla o columna", name, l)
}

// checkForeignKeys validates loaded metadata so a broken reference is an
// error instead of a silently ignored constraint.
func (c *Catalog) checkForeignKeys() error {
	for _, t := range c.TableList() {
		for _, col := range t.Columns {
			r := col.References
			if r == nil {
				continue
			}
			parent := c.Table(r.Table)
			if parent == nil {
				return fmt.Errorf("la FOREIGN KEY %s.%s referencia la tabla inexistente %q", t.Name, col.Name, r.Table)
			}
			k := parent.columnIndex(r.Column)
			if k < 0 || !parent.Columns[k].PrimaryKey || parent.Columns[k].Type.Kind != col.Type.Kind {
				return fmt.Errorf("la FOREIGN KEY %s.%s no referencia una PRIMARY KEY compatible (%s(%s))", t.Name, col.Name, r.Table, r.Column)
			}
		}
	}
	return nil
}

// foreignKeyDecls collects inline REFERENCES and table-level FOREIGN KEY
// clauses of a column list in source order.
func foreignKeyDecls(cols []ColumnDef, table []ForeignKeyDef) []ForeignKeyDef {
	var out []ForeignKeyDef
	for _, c := range cols {
		if c.References != nil {
			out = append(out, *c.References)
		}
	}
	return append(out, table...)
}

// applyForeignKeys validates declarations against cat and stores them on t.
// A reference to t itself (by name) is a self-reference. It reports every
// problem and returns false if there was any.
func applyForeignKeys(t *Table, decls []ForeignKeyDef, cat *Catalog, report func(Pos, string, ...any)) bool {
	ok := true
	for _, d := range decls {
		ci := t.columnIndex(d.Column.Name)
		if ci < 0 {
			report(d.Column.Pos, "La columna %q de la FOREIGN KEY no existe en la tabla %q", d.Column.Name, t.Name)
			ok = false
			continue
		}
		parent := cat.Table(d.Table.Name)
		if strings.EqualFold(d.Table.Name, t.Name) {
			parent = t
		}
		if parent == nil {
			report(d.Table.Pos, "La tabla referenciada %q no existe", d.Table.Name)
			ok = false
			continue
		}
		pi := parent.columnIndex(d.RefColumn.Name)
		if pi < 0 {
			report(d.RefColumn.Pos, "La columna %q no existe en la tabla %q", d.RefColumn.Name, parent.Name)
			ok = false
			continue
		}
		child, key := t.Columns[ci], parent.Columns[pi]
		switch {
		case !key.PrimaryKey:
			report(d.RefColumn.Pos, "La FOREIGN KEY %s.%s debe referenciar la PRIMARY KEY de %q, y %q no lo es", t.Name, child.Name, parent.Name, key.Name)
		case key.Type.Kind != child.Type.Kind:
			report(d.Pos, "Tipos incompatibles en la FOREIGN KEY: %s.%s es %s y %s.%s es %s", t.Name, child.Name, child.Type, parent.Name, key.Name, key.Type)
		case child.References != nil:
			report(d.Pos, "La columna %q ya tiene una FOREIGN KEY", child.Name)
		default:
			t.Columns[ci].References = &ForeignKey{Table: parent.Name, Column: key.Name}
			continue
		}
		ok = false
	}
	return ok
}

// fkRows reads t, or returns rows when t is the table being written.
func (e *Engine) fkRows(t, written *Table, rows [][]any) ([][]any, error) {
	if strings.EqualFold(t.Name, written.Name) {
		return rows, nil
	}
	return readRows(e.Dir, t)
}

// checkChildValues verifies that the written rows of t only reference keys
// present in their parents. final is the whole table t after the write.
func (e *Engine) checkChildValues(cat *Catalog, t *Table, written, final [][]any, tail string) error {
	for _, l := range cat.links() {
		if !strings.EqualFold(l.child.Name, t.Name) {
			continue
		}
		var keys map[any]bool
		for _, r := range written {
			v := r[l.col]
			if v == nil {
				continue
			}
			if keys == nil {
				prows, err := e.fkRows(l.parent, t, final)
				if err != nil {
					return err
				}
				keys = map[any]bool{}
				for _, p := range prows {
					keys[p[l.key]] = true
				}
			}
			if !keys[v] {
				return fmt.Errorf("Violación de FOREIGN KEY: el valor %v de %s no existe en %s.%s; %s", v, l, l.parent.Name, l.parent.Columns[l.key].Name, tail)
			}
		}
	}
	return nil
}

// checkRemovedKeys rejects a write of t that removes (by DELETE or by
// changing it with UPDATE) a key value still referenced by some row.
func (e *Engine) checkRemovedKeys(cat *Catalog, t *Table, old, final [][]any, fail func(v any, l fkLink) error) error {
	links := cat.referencesTo(t.Name)
	if len(links) == 0 {
		return nil
	}
	key := links[0].key
	remaining := map[any]bool{}
	for _, r := range final {
		remaining[r[key]] = true
	}
	removed := map[any]bool{}
	for _, r := range old {
		if v := r[key]; v != nil && !remaining[v] {
			removed[v] = true
		}
	}
	if len(removed) == 0 {
		return nil
	}
	for _, l := range links {
		rows, err := e.fkRows(l.child, t, final)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if v := r[l.col]; v != nil && removed[v] {
				return fail(v, l)
			}
		}
	}
	return nil
}
