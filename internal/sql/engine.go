package sql

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Engine runs SQL scripts against a database directory.
type Engine struct {
	Dir string
	// NoIndex makes queries ignore indexes; tests use it to check that an
	// indexed run returns the same rows as a full scan.
	NoIndex bool
}

// StmtResult is the outcome of one executed statement.
type StmtResult struct {
	Statement    int    `json:"statement"`
	Kind         string `json:"kind"`
	Message      string `json:"message"`
	RowsAffected int    `json:"rowsAffected"`
	// SELECT only: result set and how each table was read (index or full scan).
	Columns []string `json:"columns,omitempty"`
	Rows    [][]any  `json:"rows,omitempty"`
	Plan    []string `json:"plan,omitempty"`
}

type ColumnInfo struct {
	Column
	Type string `json:"type"`
}

type TableInfo struct {
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
	Rows    int          `json:"rows"`
}

// Report is everything the CLI and the IDE show for one run.
type Report struct {
	Tokens      []Token      `json:"tokens"`
	ParseTree   *TreeNode    `json:"parseTree"`
	AST         *TreeNode    `json:"ast"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Results     []StmtResult `json:"results"`
	Executed    bool         `json:"executed"`
	Schema      []TableInfo  `json:"schema"`
	Indexes     []IndexDef   `json:"indexes"`
}

// Run lexes, parses and analyses src. When execute is true and the script
// has no lexical, syntax or semantic errors, its statements run in order.
func (e *Engine) Run(src string, execute bool) Report {
	rep := Report{Diagnostics: []Diagnostic{}, Results: []StmtResult{}}
	toks, lexDiags := Lex(src)
	rep.Tokens = toks
	stmts, tree, synDiags := Parse(toks)
	rep.ParseTree = tree
	rep.AST = ASTTree(stmts)
	rep.Diagnostics = append(rep.Diagnostics, lexDiags...)
	rep.Diagnostics = append(rep.Diagnostics, synDiags...)

	cat, err := LoadCatalog(e.Dir)
	if err != nil {
		rep.Diagnostics = append(rep.Diagnostics, Diagnostic{Phase: "execution", Message: "No se pudo cargar la base de datos: " + err.Error(), Line: 1, Col: 1})
		return rep
	}
	if len(rep.Diagnostics) == 0 {
		rep.Diagnostics = append(rep.Diagnostics, Analyze(stmts, cat.Clone())...)
	}
	if execute && len(rep.Diagnostics) == 0 {
		rep.Executed = true
		e.execute(stmts, cat, &rep)
		if reloaded, err := LoadCatalog(e.Dir); err == nil {
			cat = reloaded
		}
	}
	rep.Schema = e.schemaInfo(cat)
	rep.Indexes = []IndexDef{}
	for _, k := range cat.idxOrd {
		rep.Indexes = append(rep.Indexes, *cat.Indexes[k])
	}
	return rep
}

func (e *Engine) schemaInfo(cat *Catalog) []TableInfo {
	out := []TableInfo{}
	for _, t := range cat.TableList() {
		ti := TableInfo{Name: t.Name, Rows: -1}
		for _, c := range t.Columns {
			ti.Columns = append(ti.Columns, ColumnInfo{Column: c, Type: c.Type.String()})
		}
		if rows, err := readRows(e.Dir, t); err == nil {
			ti.Rows = len(rows)
		}
		out = append(out, ti)
	}
	return out
}

func (e *Engine) execute(stmts []Statement, cat *Catalog, rep *Report) {
	if err := os.MkdirAll(e.Dir, 0o755); err != nil {
		rep.Diagnostics = append(rep.Diagnostics, Diagnostic{Phase: "execution", Message: err.Error(), Line: 1, Col: 1})
		return
	}
	for i, s := range stmts {
		n := i + 1
		res, err := e.executeOne(s, cat)
		if err != nil {
			p := s.Position()
			var pe *posErr
			if errors.As(err, &pe) {
				p = pe.Pos
			}
			rep.Diagnostics = append(rep.Diagnostics, Diagnostic{Phase: "execution", Message: err.Error(), Line: p.Line, Col: p.Col, Statement: n})
			return
		}
		res.Statement = n
		rep.Results = append(rep.Results, res)
	}
}

func (e *Engine) executeOne(s Statement, cat *Catalog) (StmtResult, error) {
	switch s := s.(type) {
	case *CreateTable:
		t := &Table{Name: s.Name.Name}
		for _, c := range s.Columns {
			t.Columns = append(t.Columns, Column{
				Name:       c.Name.Name,
				Type:       ColType{Kind: c.Type.Kind, Length: c.Type.Length},
				PrimaryKey: c.PrimaryKey,
				NotNull:    c.NotNull || c.PrimaryKey,
			})
		}
		// Data file first: a crash before the metadata write leaves an
		// unreferenced file, never a table without its data.
		if err := writeRows(e.Dir, t, nil); err != nil {
			return StmtResult{}, err
		}
		cat.AddTable(t)
		if err := cat.Save(e.Dir); err != nil {
			return StmtResult{}, err
		}
		return StmtResult{Kind: "CREATE TABLE", Message: fmt.Sprintf("Tabla %q creada", t.Name)}, nil
	case *DropTable:
		t := cat.Table(s.Name.Name)
		var idxFiles []string
		for _, ix := range cat.Indexes {
			if strings.EqualFold(ix.Table, t.Name) {
				idxFiles = append(idxFiles, indexFile(e.Dir, ix.Name))
			}
		}
		cat.DropTable(t.Name)
		if err := cat.Save(e.Dir); err != nil {
			return StmtResult{}, err
		}
		for _, f := range append(idxFiles, tableFile(e.Dir, t)) {
			if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
				return StmtResult{}, err
			}
		}
		return StmtResult{Kind: "DROP TABLE", Message: fmt.Sprintf("Tabla %q eliminada", t.Name)}, nil
	case *Insert:
		return e.insert(s, cat)
	case *Select:
		return e.query(s, cat)
	case *Update:
		return e.update(s, cat)
	case *Delete:
		return e.deleteRows(s, cat)
	case *CreateIndex:
		return e.createIndex(s, cat)
	case *DropIndex:
		return e.dropIndex(s, cat)
	}
	return StmtResult{}, fmt.Errorf("sentencia no soportada: %T", s)
}

func (e *Engine) insert(s *Insert, cat *Catalog) (StmtResult, error) {
	t := cat.Table(s.Table.Name)
	var problem error
	newRows := buildInsertRows(t, s, func(p Pos, f string, a ...any) {
		if problem == nil {
			problem = fmt.Errorf(f, a...)
		}
	})
	if problem != nil {
		return StmtResult{}, problem
	}
	rows, err := readRows(e.Dir, t)
	if err != nil {
		return StmtResult{}, err
	}
	pk := -1
	for i, c := range t.Columns {
		if c.PrimaryKey {
			pk = i
		}
	}
	if pk >= 0 {
		seen := map[any]bool{}
		for _, r := range rows {
			seen[r[pk]] = true
		}
		for n, r := range newRows {
			if seen[r[pk]] {
				return StmtResult{}, fmt.Errorf("Violación de PRIMARY KEY: el valor %v de la columna %q ya existe en la tabla %q (fila %d del INSERT); no se insertó ninguna fila", r[pk], t.Columns[pk].Name, t.Name, n+1)
			}
			seen[r[pk]] = true
		}
	}
	if err := e.writeTable(cat, t, append(rows, newRows...)); err != nil {
		return StmtResult{}, err
	}
	return StmtResult{Kind: "INSERT", RowsAffected: len(newRows), Message: fmt.Sprintf("%d fila(s) insertada(s) en %q", len(newRows), t.Name)}, nil
}

// DefaultDir is the database directory used when none is configured.
func DefaultDir() string { return filepath.Join("sqldata", "default") }
