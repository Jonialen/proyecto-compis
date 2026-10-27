package sql

import (
	"cmp"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Query execution. Values are nil (NULL), int64, float64, string or bool.
// Conditions use three-valued logic: a comparison with NULL is NULL, and a
// row is kept only when its condition is true.

type jrow [][]any // one row per table of the query, in FROM/JOIN order

type boundTab struct {
	label string
	table *Table
}

// env is the evaluation context: the tables of one query level, the current
// row, the enclosing query (for correlated subqueries) and, while evaluating
// an aggregate query, the rows of the current group.
type env struct {
	tabs   []boundTab
	row    jrow
	parent *env
	group  []jrow
}

func (e *env) lookup(ref *ColumnRef) (any, bool) {
	for x := e; x != nil; x = x.parent {
		for i, bt := range x.tabs {
			if ref.Qualifier != "" && bt.label != strings.ToLower(ref.Qualifier) {
				continue
			}
			if ci := bt.table.columnIndex(ref.Name); ci >= 0 {
				if i >= len(x.row) || x.row[i] == nil {
					return nil, true // empty group: no row to read from
				}
				return x.row[i][ci], true
			}
		}
	}
	return nil, false
}

type queryResult struct {
	cols []string
	rows [][]any
}

const maxPlanNotes = 20

// execCtx holds what one statement needs while it runs. It is created per
// statement, so the caches never outlive a write.
type execCtx struct {
	dir     string
	cat     *Catalog
	tables  map[string][][]any
	indexes map[string]*loadedIndex
	subs    map[*Select]*queryResult
	plan    []string
	seen    map[string]bool
	noIndex bool // tests: force full scans to compare against index results
}

func newExecCtx(dir string, cat *Catalog) *execCtx {
	return &execCtx{dir: dir, cat: cat, tables: map[string][][]any{}, indexes: map[string]*loadedIndex{}, subs: map[*Select]*queryResult{}, seen: map[string]bool{}}
}

func (x *execCtx) note(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if x.seen[msg] {
		return
	}
	x.seen[msg] = true
	switch {
	case len(x.plan) < maxPlanNotes:
		x.plan = append(x.plan, msg)
	case len(x.plan) == maxPlanNotes:
		x.plan = append(x.plan, "…")
	}
}

// FormatValue renders a value for display: NULL, text as is, TRUE/FALSE.
func FormatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	if s, ok := v.(string); ok {
		return s
	}
	return encodeCell(v)
}

func literalText(v any) string {
	if s, ok := v.(string); ok {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return FormatValue(v)
}

func compareValues(a, b any) int {
	switch x := a.(type) {
	case int64:
		switch y := b.(type) {
		case int64:
			return cmp.Compare(x, y)
		case float64:
			return cmp.Compare(float64(x), y)
		}
	case float64:
		switch y := b.(type) {
		case int64:
			return cmp.Compare(x, float64(y))
		case float64:
			return cmp.Compare(x, y)
		}
	case string:
		if y, ok := b.(string); ok {
			return strings.Compare(x, y)
		}
	case bool:
		if y, ok := b.(bool); ok {
			switch {
			case x == y:
				return 0
			case !x:
				return -1
			}
			return 1
		}
	}
	return 0
}

func asBool(v any) any {
	if b, ok := v.(bool); ok {
		return b
	}
	return nil
}

func isTrue(v any) bool { b, ok := v.(bool); return ok && b }

func (x *execCtx) eval(e Expr, en *env) (any, error) {
	switch e := e.(type) {
	case *Literal:
		v, perr := evalConst(e)
		if perr != nil {
			return nil, perr
		}
		return v, nil
	case *ColumnRef:
		v, ok := en.lookup(e)
		if !ok {
			return nil, &posErr{e.Pos, fmt.Sprintf("La columna %q no existe", e.Name)}
		}
		return v, nil
	case *Unary:
		v, err := x.eval(e.X, en)
		if err != nil || v == nil {
			return nil, err
		}
		if e.Op == "NOT" {
			return !isTrue(v), nil
		}
		switch n := v.(type) {
		case int64:
			if n == math.MinInt64 {
				return nil, &posErr{e.Pos, "Desbordamiento de entero"}
			}
			return -n, nil
		case float64:
			return -n, nil
		}
		return nil, &posErr{e.Pos, "El operador '-' requiere un valor numérico"}
	case *Binary:
		return x.evalBinary(e, en)
	case *IsNull:
		v, err := x.eval(e.X, en)
		if err != nil {
			return nil, err
		}
		return (v == nil) != e.Not, nil
	case *InExpr:
		return x.evalIn(e, en)
	case *ExistsExpr:
		res, err := x.subquery(e.Sub, en)
		if err != nil {
			return nil, err
		}
		return len(res.rows) > 0, nil
	case *SubqueryExpr:
		res, err := x.subquery(e.Sub, en)
		if err != nil {
			return nil, err
		}
		switch {
		case len(res.rows) > 1:
			return nil, &posErr{e.Pos, fmt.Sprintf("La subconsulta escalar retornó %d registros; debe retornar como máximo uno", len(res.rows))}
		case len(res.rows) == 0:
			return nil, nil
		}
		return res.rows[0][0], nil
	case *FuncCall:
		return x.evalAggregate(e, en)
	}
	return nil, &posErr{e.Position(), "Expresión no soportada"}
}

func (x *execCtx) evalBinary(e *Binary, en *env) (any, error) {
	l, err := x.eval(e.Left, en)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case "AND":
		if b, ok := l.(bool); ok && !b {
			return false, nil
		}
	case "OR":
		if isTrue(l) {
			return true, nil
		}
	}
	r, err := x.eval(e.Right, en)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case "AND":
		if b, ok := r.(bool); ok && !b {
			return false, nil
		}
		if l == nil || r == nil {
			return nil, nil
		}
		return true, nil
	case "OR":
		if isTrue(r) {
			return true, nil
		}
		if l == nil || r == nil {
			return nil, nil
		}
		return false, nil
	case "+", "-", "*", "/":
		if l == nil || r == nil {
			return nil, nil
		}
		v, perr := arith(e.Op, l, r, e.Pos)
		if perr != nil {
			return nil, perr
		}
		return v, nil
	}
	if l == nil || r == nil {
		return nil, nil
	}
	c := compareValues(l, r)
	switch e.Op {
	case "=":
		return c == 0, nil
	case "<>":
		return c != 0, nil
	case "<":
		return c < 0, nil
	case "<=":
		return c <= 0, nil
	case ">":
		return c > 0, nil
	}
	return c >= 0, nil // >=
}

func (x *execCtx) evalIn(e *InExpr, en *env) (any, error) {
	lv, err := x.eval(e.X, en)
	if err != nil {
		return nil, err
	}
	var set []any
	if e.Sub != nil {
		res, err := x.subquery(e.Sub, en)
		if err != nil {
			return nil, err
		}
		for _, row := range res.rows {
			set = append(set, row[0])
		}
	} else {
		for _, item := range e.List {
			v, err := x.eval(item, en)
			if err != nil {
				return nil, err
			}
			set = append(set, v)
		}
	}
	if len(set) == 0 {
		return e.Not, nil
	}
	found, sawNull := false, lv == nil
	for _, v := range set {
		if v == nil {
			sawNull = true
		} else if lv != nil && compareValues(lv, v) == 0 {
			found = true
			break
		}
	}
	switch {
	case found:
		return !e.Not, nil
	case sawNull:
		return nil, nil
	}
	return e.Not, nil
}

func (x *execCtx) subquery(s *Select, en *env) (*queryResult, error) {
	if !s.Correlated {
		if res, ok := x.subs[s]; ok {
			return res, nil
		}
	}
	res, err := x.runSelect(s, en)
	if err != nil {
		return nil, err
	}
	if !s.Correlated {
		x.subs[s] = res
	}
	return res, nil
}

func (x *execCtx) evalAggregate(e *FuncCall, en *env) (any, error) {
	if en.group == nil {
		return nil, &posErr{e.Pos, fmt.Sprintf("La función %s solo se puede usar en una consulta agregada", e.Name)}
	}
	if e.Star {
		return int64(len(en.group)), nil
	}
	var vals []any
	for _, g := range en.group {
		v, err := x.eval(e.Arg, &env{tabs: en.tabs, row: g, parent: en.parent})
		if err != nil {
			return nil, err
		}
		if v != nil {
			vals = append(vals, v)
		}
	}
	switch e.Name {
	case "COUNT":
		return int64(len(vals)), nil
	case "SUM", "AVG":
		if len(vals) == 0 {
			return nil, nil
		}
		var sum any = int64(0)
		for _, v := range vals {
			s, perr := arith("+", sum, v, e.Pos)
			if perr != nil {
				return nil, perr
			}
			sum = s
		}
		if e.Name == "SUM" {
			return sum, nil
		}
		f, _ := toFloat(sum)
		return f / float64(len(vals)), nil
	}
	if len(vals) == 0 {
		return nil, nil
	}
	best := vals[0]
	for _, v := range vals[1:] {
		if c := compareValues(v, best); (e.Name == "MIN" && c < 0) || (e.Name == "MAX" && c > 0) {
			best = v
		}
	}
	return best, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// arith applies + - * / to two non-NULL values. Integer operands give an
// integer result (division truncates); any float operand gives a float.
func arith(op string, l, r any, pos Pos) (any, *posErr) {
	if op != "+" && op != "-" && op != "*" && op != "/" {
		return nil, &posErr{pos, fmt.Sprintf("El operador '%s' no se admite aquí", op)}
	}
	li, lInt := l.(int64)
	ri, rInt := r.(int64)
	if lInt && rInt {
		overflow := &posErr{pos, "Desbordamiento de entero"}
		switch op {
		case "+":
			if s := li + ri; (s > li) == (ri > 0) {
				return s, nil
			}
			return nil, overflow
		case "-":
			if d := li - ri; (d < li) == (ri > 0) {
				return d, nil
			}
			return nil, overflow
		case "*":
			if li == 0 || ri == 0 {
				return int64(0), nil
			}
			p := li * ri
			if p/ri != li || (li == -1 && ri == math.MinInt64) || (ri == -1 && li == math.MinInt64) {
				return nil, overflow
			}
			return p, nil
		}
		if ri == 0 {
			return nil, &posErr{pos, "División por cero"}
		}
		if li == math.MinInt64 && ri == -1 {
			return nil, overflow
		}
		return li / ri, nil
	}
	lf, lok := toFloat(l)
	rf, rok := toFloat(r)
	if !lok || !rok {
		return nil, &posErr{pos, fmt.Sprintf("El operador '%s' requiere valores numéricos pero recibió %s y %s", op, kindOf(l), kindOf(r))}
	}
	var out float64
	switch op {
	case "+":
		out = lf + rf
	case "-":
		out = lf - rf
	case "*":
		out = lf * rf
	default:
		if rf == 0 {
			return nil, &posErr{pos, "División por cero"}
		}
		out = lf / rf
	}
	if math.IsInf(out, 0) || math.IsNaN(out) {
		return nil, &posErr{pos, "Desbordamiento de número de punto flotante"}
	}
	return out, nil
}

// --- tables and indexes ---

type colPred struct {
	column string
	pred
}

func flattenAnd(e Expr, out []Expr) []Expr {
	if b, ok := e.(*Binary); ok && b.Op == "AND" {
		return flattenAnd(b.Right, flattenAnd(b.Left, out))
	}
	if e != nil {
		out = append(out, e)
	}
	return out
}

var flipped = map[string]string{"=": "=", "<": ">", "<=": ">=", ">": "<", ">=": "<="}

// indexPreds collects the WHERE conjuncts of the form column <op> constant
// that apply to table i of the query; they decide whether an index can be used.
func (x *execCtx) indexPreds(tabs []boundTab, i int, conjuncts []Expr) []colPred {
	var out []colPred
	owns := func(c *ColumnRef) bool {
		if c.Qualifier != "" {
			return tabs[i].label == strings.ToLower(c.Qualifier) && tabs[i].table.columnIndex(c.Name) >= 0
		}
		count, mine := 0, false
		for j, bt := range tabs {
			if bt.table.columnIndex(c.Name) >= 0 {
				count++
				mine = mine || j == i
			}
		}
		return count == 1 && mine
	}
	for _, c := range conjuncts {
		if in, ok := c.(*InExpr); ok && !in.Not && in.Sub == nil {
			col, isCol := in.X.(*ColumnRef)
			if !isCol || !owns(col) {
				continue
			}
			ct := tabs[i].table.Columns[tabs[i].table.columnIndex(col.Name)].Type
			var vals []any
			usable := true
			for _, item := range in.List {
				v, perr := evalConst(item)
				if perr != nil || (v != nil && !indexableConst(ct, v)) {
					usable = false
					break
				}
				if v != nil { // x IN (1, NULL) can only be true for x = 1
					vals = append(vals, v)
				}
			}
			if usable {
				out = append(out, colPred{strings.ToLower(col.Name), pred{"IN", vals}})
			}
			continue
		}
		b, ok := c.(*Binary)
		if !ok {
			continue
		}
		op := b.Op
		if _, ok := flipped[op]; !ok {
			continue
		}
		col, isCol := b.Left.(*ColumnRef)
		other := b.Right
		if !isCol {
			col, isCol = b.Right.(*ColumnRef)
			other, op = b.Left, flipped[op]
		}
		if !isCol || !owns(col) {
			continue
		}
		v, perr := evalConst(other)
		if perr != nil || v == nil {
			continue
		}
		ct := tabs[i].table.Columns[tabs[i].table.columnIndex(col.Name)].Type
		if !indexableConst(ct, v) {
			continue
		}
		out = append(out, colPred{strings.ToLower(col.Name), pred{op, v}})
	}
	return out
}

func indexableConst(ct ColType, v any) bool {
	switch ct.Kind {
	case tInt, tFloat:
		_, ok := toFloat(v)
		return ok
	case tVarchar, tDate:
		_, ok := v.(string)
		return ok
	case tBool:
		_, ok := v.(bool)
		return ok
	}
	return false
}

func condText(column string, preds []pred) string {
	parts := make([]string, len(preds))
	for i, p := range preds {
		if p.op == "IN" {
			items := make([]string, len(p.v.([]any)))
			for j, v := range p.v.([]any) {
				items[j] = literalText(v)
			}
			parts[i] = column + " IN (" + strings.Join(items, ", ") + ")"
			continue
		}
		parts[i] = column + " " + p.op + " " + literalText(p.v)
	}
	return strings.Join(parts, " AND ")
}

func (x *execCtx) index(t *Table, ix *IndexDef) (*loadedIndex, error) {
	key := strings.ToLower(ix.Name)
	if li, ok := x.indexes[key]; ok {
		return li, nil
	}
	li, err := loadIndex(x.dir, t, ix)
	if err != nil {
		return nil, err
	}
	x.indexes[key] = li
	return li, nil
}

// chooseIndex picks the index to use for the predicates of one table,
// preferring a column that has an equality condition.
func (x *execCtx) chooseIndex(t *Table, preds []colPred) (*IndexDef, []pred) {
	if x.noIndex {
		return nil, nil
	}
	byCol := map[string][]pred{}
	var order []string
	for _, p := range preds {
		if _, ok := byCol[p.column]; !ok {
			order = append(order, p.column)
		}
		byCol[p.column] = append(byCol[p.column], p.pred)
	}
	var best *IndexDef
	var bestPreds []pred
	bestEq := false
	for _, col := range order {
		for _, key := range x.cat.idxOrd {
			ix := x.cat.Indexes[key]
			if !strings.EqualFold(ix.Table, t.Name) || !strings.EqualFold(ix.Column, col) {
				continue
			}
			eq := false
			for _, p := range byCol[col] {
				eq = eq || p.op == "=" || p.op == "IN"
			}
			if best == nil || (eq && !bestEq) {
				best, bestPreds, bestEq = ix, byCol[col], eq
			}
		}
	}
	return best, bestPreds
}

// candidateOffsets returns the offsets of the records an index selects, and
// the index used, or nil when no index applies.
func (x *execCtx) candidateOffsets(t *Table, preds []colPred) ([]int64, *loadedIndex, *IndexDef, []pred, error) {
	ix, ps := x.chooseIndex(t, preds)
	if ix == nil {
		return nil, nil, nil, nil, nil
	}
	li, err := x.index(t, ix)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return li.lookup(ps), li, ix, ps, nil
}

// rowsFor returns the rows of a table that can satisfy the predicates:
// only the index matches when an index applies, otherwise every record.
func (x *execCtx) rowsFor(t *Table, preds []colPred) ([][]any, error) {
	offs, li, ix, ps, err := x.candidateOffsets(t, preds)
	if err != nil {
		return nil, err
	}
	if ix != nil {
		x.note("Índice %q usado en %q por %s: %d de %d registros leídos", ix.Name, t.Name, condText(ix.Column, ps), len(offs), li.total)
		return readRowsAt(x.dir, t, offs)
	}
	key := strings.ToLower(t.Name)
	rows, ok := x.tables[key]
	if !ok {
		var err error
		if rows, err = readRows(x.dir, t); err != nil {
			return nil, err
		}
		x.tables[key] = rows
	}
	x.note("Recorrido completo de %q: %d registros", t.Name, len(rows))
	return rows, nil
}

// --- SELECT ---

func columnLabel(it SelectItem) string {
	if it.Alias != "" {
		return it.Alias
	}
	return exprText(it.Expr)
}

func exprText(e Expr) string {
	switch e := e.(type) {
	case *ColumnRef:
		if e.Qualifier != "" {
			return e.Qualifier + "." + e.Name
		}
		return e.Name
	case *FuncCall:
		if e.Star {
			return e.Name + "(*)"
		}
		return e.Name + "(" + exprText(e.Arg) + ")"
	case *Literal:
		if e.Kind == "STRING" {
			return literalText(e.Text)
		}
		return e.Text
	case *Binary:
		return exprText(e.Left) + " " + e.Op + " " + exprText(e.Right)
	case *Unary:
		if e.Op == "NOT" {
			return "NOT " + exprText(e.X)
		}
		return "-" + exprText(e.X)
	}
	return "expr"
}

func (x *execCtx) runSelect(s *Select, parent *env) (*queryResult, error) {
	refs := []TableRef{s.From}
	for _, j := range s.Joins {
		refs = append(refs, j.Table)
	}
	tabs := make([]boundTab, len(refs))
	for i, r := range refs {
		tabs[i] = boundTab{r.Label(), x.cat.Table(r.Name.Name)}
	}

	conjuncts := flattenAnd(s.Where, nil)
	perTable := make([][][]any, len(tabs))
	for i := range tabs {
		rows, err := x.rowsFor(tabs[i].table, x.indexPreds(tabs, i, conjuncts))
		if err != nil {
			return nil, err
		}
		perTable[i] = rows
	}

	cur := make([]jrow, len(perTable[0]))
	for i, r := range perTable[0] {
		cur[i] = jrow{r}
	}
	for j := 1; j < len(tabs); j++ {
		var next []jrow
		for _, left := range cur {
			for _, r := range perTable[j] {
				combined := append(append(jrow{}, left...), r)
				v, err := x.eval(s.Joins[j-1].On, &env{tabs: tabs[:j+1], row: combined, parent: parent})
				if err != nil {
					return nil, err
				}
				if isTrue(v) {
					next = append(next, combined)
				}
			}
		}
		cur = next
	}

	var filtered []jrow
	for _, jr := range cur {
		if s.Where != nil {
			v, err := x.eval(s.Where, &env{tabs: tabs, row: jr, parent: parent})
			if err != nil {
				return nil, err
			}
			if !isTrue(v) {
				continue
			}
		}
		filtered = append(filtered, jr)
	}

	// Output columns; * expands to every column of every table.
	var cols []string
	for _, it := range s.Items {
		if it.Star {
			for _, bt := range tabs {
				for _, c := range bt.table.Columns {
					cols = append(cols, c.Name)
				}
			}
			continue
		}
		cols = append(cols, columnLabel(it))
	}

	type outRow struct {
		vals []any
		keys []any
	}
	var out []outRow
	emit := func(en *env) error {
		var vals []any
		for _, it := range s.Items {
			if it.Star {
				for i := range tabs {
					if en.row == nil || en.row[i] == nil {
						vals = append(vals, make([]any, len(tabs[i].table.Columns))...)
					} else {
						vals = append(vals, en.row[i]...)
					}
				}
				continue
			}
			v, err := x.eval(it.Expr, en)
			if err != nil {
				return err
			}
			vals = append(vals, v)
		}
		keys := make([]any, len(s.OrderBy))
		for k, o := range s.OrderBy {
			if c, ok := o.Expr.(*ColumnRef); ok && c.Qualifier == "" {
				if idx := aliasIndex(s, c.Name); idx >= 0 {
					keys[k] = vals[idx]
					continue
				}
			}
			v, err := x.eval(o.Expr, en)
			if err != nil {
				return err
			}
			keys[k] = v
		}
		out = append(out, outRow{vals, keys})
		return nil
	}

	aggregate := len(s.GroupBy) > 0 || (s.Having != nil && containsAgg(s.Having))
	for _, it := range s.Items {
		aggregate = aggregate || (!it.Star && containsAgg(it.Expr))
	}
	for _, o := range s.OrderBy {
		aggregate = aggregate || containsAgg(o.Expr)
	}
	if !aggregate {
		for _, jr := range filtered {
			if err := emit(&env{tabs: tabs, row: jr, parent: parent}); err != nil {
				return nil, err
			}
		}
	} else {
		groups, err := x.groupRows(s, tabs, filtered, parent)
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			en := &env{tabs: tabs, parent: parent, group: g}
			if len(g) > 0 {
				en.row = g[0]
			}
			if s.Having != nil {
				v, err := x.eval(s.Having, en)
				if err != nil {
					return nil, err
				}
				if !isTrue(v) {
					continue
				}
			}
			if err := emit(en); err != nil {
				return nil, err
			}
		}
	}

	if len(s.OrderBy) > 0 {
		sort.SliceStable(out, func(a, b int) bool {
			for k, o := range s.OrderBy {
				c := compareNullsFirst(out[a].keys[k], out[b].keys[k])
				if o.Desc {
					c = -c
				}
				if c != 0 {
					return c < 0
				}
			}
			return false
		})
	}
	res := &queryResult{cols: cols, rows: make([][]any, len(out))}
	for i, o := range out {
		res.rows[i] = o.vals
	}
	return res, nil
}

func aliasIndex(s *Select, name string) int {
	idx := 0
	for _, it := range s.Items {
		if it.Star {
			continue
		}
		if it.Alias != "" && strings.EqualFold(it.Alias, name) {
			return idx
		}
		idx++
	}
	return -1
}

func compareNullsFirst(a, b any) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	}
	return compareValues(a, b)
}

// groupRows splits the filtered rows by the GROUP BY key, in order of first
// appearance. Without GROUP BY the whole input is one group, even if empty.
func (x *execCtx) groupRows(s *Select, tabs []boundTab, rows []jrow, parent *env) ([][]jrow, error) {
	if len(s.GroupBy) == 0 {
		return [][]jrow{append([]jrow{}, rows...)}, nil
	}
	index := map[string]int{}
	var groups [][]jrow
	for _, jr := range rows {
		en := &env{tabs: tabs, row: jr, parent: parent}
		var key strings.Builder
		for _, g := range s.GroupBy {
			v, err := x.eval(g, en)
			if err != nil {
				return nil, err
			}
			key.WriteString(fmt.Sprintf("%T:%v\x00", v, v))
		}
		i, ok := index[key.String()]
		if !ok {
			i = len(groups)
			index[key.String()] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], jr)
	}
	return groups, nil
}
