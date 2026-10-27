package sql

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Expression types used by the checker. "?" means an earlier error already
// produced the type, so no follow-up error is reported for it.
const (
	tInt, tFloat, tVarchar, tBool, tDate, tNull, tUnknown = "INT", "FLOAT", "VARCHAR", "BOOLEAN", "DATE", "NULL", "?"
)

type scopeEntry struct {
	label string
	table *Table
}

type scope struct {
	entries []scopeEntry
	parent  *scope
	sel     *Select // query that owns this scope; nil for UPDATE and DELETE
	broken  bool    // a table of this scope does not exist; do not cascade column errors
}

type analyzer struct {
	cat   *Catalog
	diags []Diagnostic
	stmt  int
	quiet bool
}

// Analyze checks every statement against the catalog. DDL statements are
// applied to cat as they are checked so later statements of the same script
// see their effect; pass a clone when the real database must not change.
func Analyze(stmts []Statement, cat *Catalog) []Diagnostic {
	a := &analyzer{cat: cat}
	for i, s := range stmts {
		a.stmt = i + 1
		a.statement(s)
	}
	return a.diags
}

func (a *analyzer) errAt(p Pos, format string, args ...any) {
	if a.quiet {
		return
	}
	a.diags = append(a.diags, Diagnostic{Phase: "semantic", Message: fmt.Sprintf(format, args...), Line: p.Line, Col: p.Col, Statement: a.stmt})
}

func (a *analyzer) statement(s Statement) {
	switch s := s.(type) {
	case *CreateTable:
		a.createTable(s)
	case *DropTable:
		if a.cat.Table(s.Name.Name) == nil {
			a.errAt(s.Name.Pos, "La tabla %q no existe", s.Name.Name)
			return
		}
		a.cat.DropTable(s.Name.Name)
	case *CreateIndex:
		a.createIndex(s)
	case *DropIndex:
		if a.cat.Indexes[s.Name.Key()] == nil {
			a.errAt(s.Name.Pos, "El índice %q no existe", s.Name.Name)
			return
		}
		a.cat.DropIndex(s.Name.Name)
	case *Insert:
		if t := a.cat.Table(s.Table.Name); t == nil {
			a.errAt(s.Table.Pos, "La tabla %q no existe", s.Table.Name)
		} else {
			buildInsertRows(t, s, a.errAt)
		}
	case *Update:
		a.update(s)
	case *Delete:
		a.delete(s)
	case *Select:
		a.selectStmt(s, nil)
	}
}

func (a *analyzer) createTable(s *CreateTable) {
	ok := true
	if a.cat.Table(s.Name.Name) != nil {
		a.errAt(s.Name.Pos, "La tabla %q ya existe", s.Name.Name)
		ok = false
	}
	t := &Table{Name: s.Name.Name}
	seen := map[string]bool{}
	pks := 0
	for _, c := range s.Columns {
		if seen[c.Name.Key()] {
			a.errAt(c.Name.Pos, "La columna %q está duplicada en la tabla %q", c.Name.Name, s.Name.Name)
			ok = false
		}
		seen[c.Name.Key()] = true
		if c.Type.Kind == "VARCHAR" && c.Type.Length <= 0 {
			a.errAt(c.Type.Pos, "La longitud de VARCHAR en la columna %q debe ser mayor que 0", c.Name.Name)
			ok = false
		}
		if c.PrimaryKey {
			pks++
			if pks == 2 {
				a.errAt(c.Name.Pos, "La tabla %q define más de una PRIMARY KEY", s.Name.Name)
				ok = false
			}
		}
		t.Columns = append(t.Columns, Column{
			Name:       c.Name.Name,
			Type:       ColType{Kind: c.Type.Kind, Length: c.Type.Length},
			PrimaryKey: c.PrimaryKey,
			NotNull:    c.NotNull || c.PrimaryKey,
		})
	}
	if ok {
		a.cat.AddTable(t)
	}
}

func (a *analyzer) createIndex(s *CreateIndex) {
	ok := true
	if a.cat.Indexes[s.Name.Key()] != nil {
		a.errAt(s.Name.Pos, "El índice %q ya existe", s.Name.Name)
		ok = false
	}
	t := a.cat.Table(s.Table.Name)
	if t == nil {
		a.errAt(s.Table.Pos, "La tabla %q no existe", s.Table.Name)
		return
	}
	if t.columnIndex(s.Column.Name) < 0 {
		a.errAt(s.Column.Pos, "La columna %q no existe en la tabla %q", s.Column.Name, t.Name)
		return
	}
	for _, ix := range a.cat.Indexes {
		if strings.EqualFold(ix.Table, t.Name) && strings.EqualFold(ix.Column, s.Column.Name) {
			a.errAt(s.Name.Pos, "Índice duplicado: la columna %q de %q ya está indexada por %q", s.Column.Name, t.Name, ix.Name)
			ok = false
		}
	}
	if ok {
		a.cat.AddIndex(&IndexDef{Name: s.Name.Name, Table: t.Name, Column: t.Columns[t.columnIndex(s.Column.Name)].Name})
	}
}

func (a *analyzer) update(s *Update) {
	t := a.cat.Table(s.Table.Name)
	sc := &scope{}
	if t == nil {
		a.errAt(s.Table.Pos, "La tabla %q no existe", s.Table.Name)
		sc.broken = true
	} else {
		sc.entries = []scopeEntry{{t.Name, t}}
		sc.entries[0].label = strings.ToLower(t.Name)
	}
	seen := map[string]bool{}
	for _, as := range s.Sets {
		var target *Column
		if t != nil {
			i := t.columnIndex(as.Column.Name)
			if i < 0 {
				a.errAt(as.Column.Pos, "La columna %q no existe en la tabla %q", as.Column.Name, t.Name)
			} else {
				target = &t.Columns[i]
			}
		}
		if seen[as.Column.Key()] {
			a.errAt(as.Column.Pos, "La columna %q se asigna más de una vez en UPDATE", as.Column.Name)
		}
		seen[as.Column.Key()] = true
		ty := a.infer(as.Value, sc, false)
		if target != nil {
			a.checkAssignable(*target, ty, as.Value)
		}
	}
	a.checkWhere(s.Where, sc)
}

func (a *analyzer) delete(s *Delete) {
	t := a.cat.Table(s.Table.Name)
	sc := &scope{}
	if t == nil {
		a.errAt(s.Table.Pos, "La tabla %q no existe", s.Table.Name)
		sc.broken = true
	} else {
		sc.entries = []scopeEntry{{strings.ToLower(t.Name), t}}
	}
	a.checkWhere(s.Where, sc)
}

func (a *analyzer) checkWhere(e Expr, sc *scope) {
	if e == nil {
		return
	}
	if ty := a.infer(e, sc, false); ty != tBool && ty != tUnknown && ty != tNull {
		a.errAt(e.Position(), "La condición debe ser de tipo BOOLEAN pero es %s", ty)
	}
}

func (a *analyzer) checkAssignable(col Column, ty string, e Expr) {
	switch {
	case ty == tUnknown:
	case ty == tNull:
		if col.NotNull {
			a.errAt(e.Position(), "La columna %q no admite NULL", col.Name)
		}
	case col.Type.Kind == ty:
	case col.Type.Kind == tFloat && ty == tInt:
	case col.Type.Kind == tDate && ty == tVarchar && isDateLiteral(e):
	default:
		a.errAt(e.Position(), "Tipos incompatibles: no se puede asignar un valor %s a la columna %q de tipo %s", ty, col.Name, col.Type)
	}
}

func isDateLiteral(e Expr) bool {
	l, ok := e.(*Literal)
	return ok && l.Kind == "STRING" && validDate(l.Text)
}

type selCol struct {
	name string
	ty   string
}

func (a *analyzer) selectStmt(s *Select, parent *scope) []selCol {
	sc := &scope{parent: parent, sel: s}
	labels := map[string]bool{}
	add := func(tr TableRef) {
		t := a.cat.Table(tr.Name.Name)
		if t == nil {
			a.errAt(tr.Name.Pos, "La tabla %q no existe", tr.Name.Name)
			sc.broken = true
			return
		}
		if labels[tr.Label()] {
			a.errAt(tr.Pos, "Referencia ambigua: el nombre o alias %q se usa más de una vez en FROM", tr.Label())
			return
		}
		labels[tr.Label()] = true
		sc.entries = append(sc.entries, scopeEntry{tr.Label(), t})
	}
	add(s.From)
	for _, j := range s.Joins {
		add(j.Table)
		if ty := a.infer(j.On, sc, false); ty != tBool && ty != tUnknown {
			a.errAt(j.On.Position(), "La condición de JOIN debe ser de tipo BOOLEAN pero es %s", ty)
		}
	}
	a.checkWhere(s.Where, sc)
	for _, g := range s.GroupBy {
		a.infer(g, sc, false)
	}

	var cols []selCol
	aggregate := len(s.GroupBy) > 0 || s.Having != nil && containsAgg(s.Having)
	for _, it := range s.Items {
		if it.Star {
			for _, en := range sc.entries {
				for _, c := range en.table.Columns {
					cols = append(cols, selCol{c.Name, c.Type.Kind})
				}
			}
			continue
		}
		ty := a.infer(it.Expr, sc, true)
		cols = append(cols, selCol{itemName(it), ty})
		if containsAgg(it.Expr) {
			aggregate = true
		}
	}
	if aggregate {
		a.checkGrouping(s, sc)
	}
	if s.Having != nil {
		if ty := a.infer(s.Having, sc, true); ty != tBool && ty != tUnknown {
			a.errAt(s.Having.Position(), "La condición de HAVING debe ser de tipo BOOLEAN pero es %s", ty)
		}
	}
	for _, o := range s.OrderBy {
		if c, ok := o.Expr.(*ColumnRef); ok && c.Qualifier == "" && hasAlias(s, c.Name) {
			continue
		}
		a.infer(o.Expr, sc, true)
	}
	return cols
}

func hasAlias(s *Select, name string) bool {
	for _, it := range s.Items {
		if it.Alias != "" && strings.EqualFold(it.Alias, name) {
			return true
		}
	}
	return false
}

func itemName(it SelectItem) string {
	if it.Alias != "" {
		return it.Alias
	}
	switch e := it.Expr.(type) {
	case *ColumnRef:
		return e.Name
	case *FuncCall:
		return strings.ToLower(e.Name)
	}
	return "expr"
}

func containsAgg(e Expr) bool {
	switch e := e.(type) {
	case *FuncCall:
		return true
	case *Binary:
		return containsAgg(e.Left) || containsAgg(e.Right)
	case *Unary:
		return containsAgg(e.X)
	case *IsNull:
		return containsAgg(e.X)
	case *InExpr:
		if containsAgg(e.X) {
			return true
		}
		for _, x := range e.List {
			if containsAgg(x) {
				return true
			}
		}
	}
	return false
}

func bareColumns(e Expr, out *[]*ColumnRef) {
	switch e := e.(type) {
	case *ColumnRef:
		*out = append(*out, e)
	case *Binary:
		bareColumns(e.Left, out)
		bareColumns(e.Right, out)
	case *Unary:
		bareColumns(e.X, out)
	case *IsNull:
		bareColumns(e.X, out)
	case *InExpr:
		bareColumns(e.X, out)
		for _, x := range e.List {
			bareColumns(x, out)
		}
	}
}

// checkGrouping enforces that, once a query aggregates, every column outside
// an aggregate function is one of the GROUP BY columns.
func (a *analyzer) checkGrouping(s *Select, sc *scope) {
	key := func(c *ColumnRef) (string, bool) {
		a.quiet = true
		defer func() { a.quiet = false }()
		label, col, ok := a.resolve(sc, c)
		return label + "." + strings.ToLower(col.Name), ok
	}
	grouped := map[string]bool{}
	for _, g := range s.GroupBy {
		if c, ok := g.(*ColumnRef); ok {
			if k, ok := key(c); ok {
				grouped[k] = true
			}
		}
	}
	var bare []*ColumnRef
	for _, it := range s.Items {
		if it.Star {
			a.errAt(it.Pos, "SELECT * no puede combinarse con agregación o GROUP BY")
			continue
		}
		bareColumns(it.Expr, &bare)
	}
	if s.Having != nil {
		bareColumns(s.Having, &bare)
	}
	for _, o := range s.OrderBy {
		if c, ok := o.Expr.(*ColumnRef); ok && c.Qualifier == "" && hasAlias(s, c.Name) {
			continue
		}
		bareColumns(o.Expr, &bare)
	}
	for _, c := range bare {
		if k, ok := key(c); ok && !grouped[k] {
			a.errAt(c.Pos, "La columna %q debe aparecer en GROUP BY o dentro de una función de agregación", c.Name)
		}
	}
}

// resolve finds the column a reference points to, searching the scope chain
// from the innermost query outwards. It reports unknown or ambiguous names.
func (a *analyzer) resolve(sc *scope, ref *ColumnRef) (string, *Column, bool) {
	broken := false
	for s := sc; s != nil; s = s.parent {
		broken = broken || s.broken
		type hit struct {
			label string
			col   *Column
		}
		var hits []hit
		tableSeen := false
		for _, en := range s.entries {
			if ref.Qualifier != "" {
				if en.label != strings.ToLower(ref.Qualifier) {
					continue
				}
				tableSeen = true
			}
			if i := en.table.columnIndex(ref.Name); i >= 0 {
				hits = append(hits, hit{en.label, &en.table.Columns[i]})
			}
		}
		switch {
		case len(hits) > 1:
			a.errAt(ref.Pos, "Referencia ambigua: la columna %q existe en más de una tabla; califíquela con el nombre o alias de la tabla", ref.Name)
			return "", &Column{Name: ref.Name}, false
		case len(hits) == 1:
			for x := sc; x != s; x = x.parent {
				if x.sel != nil {
					x.sel.Correlated = true
				}
			}
			return hits[0].label, hits[0].col, true
		case tableSeen:
			a.errAt(ref.Pos, "La columna %q no existe en la tabla %q", ref.Name, ref.Qualifier)
			return "", &Column{Name: ref.Name}, false
		}
	}
	if !broken {
		if ref.Qualifier != "" {
			a.errAt(ref.Pos, "Referencia inválida: %q no es una tabla ni un alias de esta consulta", ref.Qualifier)
		} else {
			a.errAt(ref.Pos, "La columna %q no existe", ref.Name)
		}
	}
	return "", &Column{Name: ref.Name}, false
}

func isNumeric(t string) bool { return t == tInt || t == tFloat }

func (a *analyzer) infer(e Expr, sc *scope, allowAgg bool) string {
	switch e := e.(type) {
	case *Literal:
		switch e.Kind {
		case "INT":
			return tInt
		case "FLOAT":
			return tFloat
		case "STRING":
			return tVarchar
		case "BOOLEAN":
			return tBool
		}
		return tNull
	case *ColumnRef:
		if _, col, ok := a.resolve(sc, e); ok {
			return col.Type.Kind
		}
		return tUnknown
	case *Unary:
		ty := a.infer(e.X, sc, allowAgg)
		if ty == tUnknown {
			return tUnknown
		}
		if e.Op == "NOT" {
			if ty != tBool && ty != tNull {
				a.errAt(e.Pos, "El operador NOT requiere un valor BOOLEAN pero recibió %s", ty)
				return tUnknown
			}
			return tBool
		}
		if !isNumeric(ty) {
			a.errAt(e.Pos, "El operador '-' requiere un valor numérico pero recibió %s", ty)
			return tUnknown
		}
		return ty
	case *Binary:
		return a.inferBinary(e, sc, allowAgg)
	case *FuncCall:
		return a.inferAgg(e, sc, allowAgg)
	case *IsNull:
		a.infer(e.X, sc, allowAgg)
		return tBool
	case *InExpr:
		lt := a.infer(e.X, sc, allowAgg)
		if e.Sub != nil {
			cols := a.selectStmt(e.Sub, sc)
			if len(cols) != 1 {
				a.errAt(e.Sub.Pos, "La subconsulta de IN debe retornar exactamente una columna pero retorna %d", len(cols))
			} else {
				a.checkComparable(e.Pos, "IN", lt, cols[0].ty, e.X, nil)
			}
		}
		for _, x := range e.List {
			a.checkComparable(e.Pos, "IN", lt, a.infer(x, sc, allowAgg), e.X, x)
		}
		return tBool
	case *ExistsExpr:
		a.selectStmt(e.Sub, sc)
		return tBool
	case *SubqueryExpr:
		cols := a.selectStmt(e.Sub, sc)
		if len(cols) != 1 {
			a.errAt(e.Sub.Pos, "Subconsulta escalar inválida: retorna %d columnas y debe retornar exactamente una", len(cols))
			return tUnknown
		}
		return cols[0].ty
	}
	return tUnknown
}

func (a *analyzer) inferAgg(e *FuncCall, sc *scope, allowAgg bool) string {
	if !allowAgg {
		a.errAt(e.Pos, "La función de agregación %s no se permite en esta cláusula", e.Name)
		return tUnknown
	}
	if e.Star {
		return tInt
	}
	if containsAgg(e.Arg) {
		a.errAt(e.Pos, "Las funciones de agregación no se pueden anidar")
		return tUnknown
	}
	ty := a.infer(e.Arg, sc, false)
	switch e.Name {
	case "COUNT":
		return tInt
	case "SUM", "AVG":
		if ty != tUnknown && !isNumeric(ty) {
			a.errAt(e.Pos, "%s requiere una columna numérica pero recibió %s", e.Name, ty)
			return tUnknown
		}
		if e.Name == "AVG" {
			return tFloat
		}
		return ty
	}
	return ty // MIN, MAX
}

func (a *analyzer) inferBinary(e *Binary, sc *scope, allowAgg bool) string {
	l := a.infer(e.Left, sc, allowAgg)
	r := a.infer(e.Right, sc, allowAgg)
	switch e.Op {
	case "AND", "OR":
		for _, t := range []string{l, r} {
			if t != tBool && t != tNull && t != tUnknown {
				a.errAt(e.Pos, "El operador %s requiere valores BOOLEAN pero recibió %s", e.Op, t)
				return tUnknown
			}
		}
		return tBool
	case "+", "-", "*", "/":
		if l == tUnknown || r == tUnknown {
			return tUnknown
		}
		if !isNumeric(l) || !isNumeric(r) {
			a.errAt(e.Pos, "El operador '%s' requiere valores numéricos pero recibió %s y %s", e.Op, l, r)
			return tUnknown
		}
		if l == tFloat || r == tFloat {
			return tFloat
		}
		return tInt
	}
	if e.Op != "=" && e.Op != "<>" && (l == tBool || r == tBool) {
		a.errAt(e.Pos, "El operador '%s' no se puede aplicar a valores BOOLEAN", e.Op)
		return tBool
	}
	a.checkComparable(e.Pos, e.Op, l, r, e.Left, e.Right)
	return tBool
}

func (a *analyzer) checkComparable(p Pos, op, l, r string, le, re Expr) {
	switch {
	case l == tUnknown || r == tUnknown || l == tNull || r == tNull:
	case isNumeric(l) && isNumeric(r):
	case l == r:
	case l == tDate && r == tVarchar && re != nil && isDateLiteral(re):
	case r == tDate && l == tVarchar && le != nil && isDateLiteral(le):
	default:
		a.errAt(p, "Tipos incompatibles: no se puede aplicar '%s' entre %s y %s", op, l, r)
	}
}

// --- constant values (INSERT) ---

func kindOf(v any) string {
	switch v.(type) {
	case int64:
		return tInt
	case float64:
		return tFloat
	case string:
		return tVarchar
	case bool:
		return tBool
	}
	return tNull
}

type posErr struct {
	Pos
	Msg string
}

func (p *posErr) Error() string { return p.Msg }

func evalConst(e Expr) (any, *posErr) {
	switch e := e.(type) {
	case *Literal:
		switch e.Kind {
		case "INT":
			n, err := strconv.ParseInt(e.Text, 10, 64)
			if err != nil {
				return nil, &posErr{e.Pos, fmt.Sprintf("El entero %s está fuera del rango de INT", e.Text)}
			}
			return n, nil
		case "FLOAT":
			f, err := strconv.ParseFloat(e.Text, 64)
			if err != nil || math.IsInf(f, 0) {
				return nil, &posErr{e.Pos, fmt.Sprintf("El número %s está fuera del rango de FLOAT", e.Text)}
			}
			return f, nil
		case "STRING":
			return e.Text, nil
		case "BOOLEAN":
			return e.Text == "TRUE", nil
		}
		return nil, nil
	case *Unary:
		v, err := evalConst(e.X)
		if err != nil {
			return nil, err
		}
		if e.Op == "-" {
			switch x := v.(type) {
			case int64:
				return -x, nil
			case float64:
				return -x, nil
			}
		}
		return nil, &posErr{e.Pos, fmt.Sprintf("El operador %s no es aplicable a un valor %s en VALUES", e.Op, kindOf(v))}
	case *Binary:
		l, err := evalConst(e.Left)
		if err != nil {
			return nil, err
		}
		r, err := evalConst(e.Right)
		if err != nil {
			return nil, err
		}
		if l == nil || r == nil {
			return nil, nil
		}
		return arith(e.Op, l, r, e.Pos)
	case *ColumnRef:
		return nil, &posErr{e.Position(), fmt.Sprintf("Referencia inválida: VALUES no puede usar la columna %q", e.Name)}
	}
	return nil, &posErr{e.Position(), "VALUES solo admite valores constantes"}
}

// coerce converts a constant to the representation of the column type.
func coerce(v any, col Column, p Pos) (any, *posErr) {
	if v == nil {
		if col.NotNull {
			return nil, &posErr{p, fmt.Sprintf("La columna %q no admite NULL", col.Name)}
		}
		return nil, nil
	}
	mismatch := func() (any, *posErr) {
		return nil, &posErr{p, fmt.Sprintf("Tipos incompatibles: no se puede asignar un valor %s a la columna %q de tipo %s", kindOf(v), col.Name, col.Type)}
	}
	switch col.Type.Kind {
	case tInt:
		if n, ok := v.(int64); ok {
			return n, nil
		}
	case tFloat:
		switch x := v.(type) {
		case int64:
			return float64(x), nil
		case float64:
			return x, nil
		}
	case tBool:
		if b, ok := v.(bool); ok {
			return b, nil
		}
	case tVarchar:
		if s, ok := v.(string); ok {
			if n := len([]rune(s)); n > col.Type.Length {
				return nil, &posErr{p, fmt.Sprintf("El valor excede la longitud máxima de %s en la columna %q (longitud %d)", col.Type, col.Name, n)}
			}
			return s, nil
		}
	case tDate:
		if s, ok := v.(string); ok {
			if !validDate(s) {
				return nil, &posErr{p, fmt.Sprintf("Fecha inválida %q para la columna %q (se esperaba YYYY-MM-DD)", s, col.Name)}
			}
			return s, nil
		}
	}
	return mismatch()
}

// buildInsertRows validates an INSERT against the table schema and returns
// the rows converted to column types. Problems are passed to report; the
// result is only meaningful when report was never called.
func buildInsertRows(t *Table, ins *Insert, report func(Pos, string, ...any)) [][]any {
	targets := make([]int, 0, len(t.Columns))
	failed := false
	if len(ins.Columns) == 0 {
		for i := range t.Columns {
			targets = append(targets, i)
		}
	} else {
		seen := map[int]bool{}
		for _, c := range ins.Columns {
			i := t.columnIndex(c.Name)
			switch {
			case i < 0:
				report(c.Pos, "La columna %q no existe en la tabla %q", c.Name, t.Name)
				failed = true
			case seen[i]:
				report(c.Pos, "La columna %q se repite en el INSERT", c.Name)
				failed = true
			default:
				seen[i] = true
				targets = append(targets, i)
			}
		}
		if !failed {
			for i, col := range t.Columns {
				if !seen[i] && col.NotNull {
					report(ins.Pos, "La columna %q es NOT NULL y el INSERT no le asigna valor", col.Name)
					failed = true
				}
			}
		}
	}
	if failed {
		return nil
	}

	var rows [][]any
	for n, exprs := range ins.Rows {
		if len(exprs) != len(targets) {
			report(exprs[0].Position(), "La fila %d tiene %d valores pero se esperaban %d", n+1, len(exprs), len(targets))
			failed = true
			continue
		}
		row := make([]any, len(t.Columns))
		for k, e := range exprs {
			v, perr := evalConst(e)
			if perr == nil {
				v, perr = coerce(v, t.Columns[targets[k]], e.Position())
			}
			if perr != nil {
				report(perr.Pos, "%s", perr.Msg)
				failed = true
				continue
			}
			row[targets[k]] = v
		}
		rows = append(rows, row)
	}
	if failed {
		return nil
	}
	return rows
}
