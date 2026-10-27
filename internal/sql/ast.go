package sql

import (
	"strconv"
	"strings"
)

// Pos is a 1-based source position.
type Pos struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// Ident is a name together with its position.
type Ident struct {
	Name string
	Pos
}

// Key is the case-insensitive catalog key of the identifier.
func (i Ident) Key() string { return strings.ToLower(i.Name) }

type DataType struct {
	Kind   string // INT, FLOAT, VARCHAR, BOOLEAN, DATE
	Length int    // VARCHAR only
	Pos
}

type Statement interface{ Position() Pos }
type Expr interface{ Position() Pos }

func (p Pos) Position() Pos { return p }

type ColumnDef struct {
	Pos
	Name       Ident
	Type       DataType
	PrimaryKey bool
	NotNull    bool
}

type CreateTable struct {
	Pos
	Name    Ident
	Columns []ColumnDef
}

type DropTable struct {
	Pos
	Name Ident
}

type CreateIndex struct {
	Pos
	Name, Table, Column Ident
}

type DropIndex struct {
	Pos
	Name Ident
}

type Insert struct {
	Pos
	Table   Ident
	Columns []Ident // empty means all columns in table order
	Rows    [][]Expr
}

type Assignment struct {
	Column Ident
	Value  Expr
}

type Update struct {
	Pos
	Table Ident
	Sets  []Assignment
	Where Expr
}

type Delete struct {
	Pos
	Table Ident
	Where Expr
}

type SelectItem struct {
	Pos
	Star  bool
	Expr  Expr
	Alias string
}

type TableRef struct {
	Pos
	Name  Ident
	Alias string
}

// Label is the name used to qualify columns of this table reference.
func (t TableRef) Label() string {
	if t.Alias != "" {
		return strings.ToLower(t.Alias)
	}
	return t.Name.Key()
}

type Join struct {
	Pos
	Table TableRef
	On    Expr
}

type OrderItem struct {
	Expr Expr
	Desc bool
}

type Select struct {
	Pos
	Items   []SelectItem
	From    TableRef
	Joins   []Join
	Where   Expr
	GroupBy []Expr
	Having  Expr
	OrderBy []OrderItem
}

// Expressions.

type Literal struct {
	Pos
	Kind string // INT, FLOAT, STRING, BOOLEAN, NULL
	Text string
}

type ColumnRef struct {
	Pos
	Qualifier string
	Name      string
}

type Binary struct {
	Pos
	Op          string
	Left, Right Expr
}

type Unary struct {
	Pos
	Op string // NOT or -
	X  Expr
}

type FuncCall struct {
	Pos
	Name string // COUNT, SUM, AVG, MIN, MAX
	Star bool
	Arg  Expr
}

type InExpr struct {
	Pos
	X    Expr
	List []Expr
	Sub  *Select
	Not  bool
}

type ExistsExpr struct {
	Pos
	Sub *Select
}

type SubqueryExpr struct {
	Pos
	Sub *Select
}

type IsNull struct {
	Pos
	X   Expr
	Not bool
}

// TreeNode is the generic tree used to show the parse tree and the AST.
type TreeNode struct {
	Name     string      `json:"name"`
	Value    string      `json:"value,omitempty"`
	Line     int         `json:"line,omitempty"`
	Col      int         `json:"col,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`
}

func tn(name, value string, p Pos, children ...*TreeNode) *TreeNode {
	return &TreeNode{Name: name, Value: value, Line: p.Line, Col: p.Col, Children: children}
}

// ASTTree renders the statements as a tree rooted at "Script".
func ASTTree(stmts []Statement) *TreeNode {
	root := &TreeNode{Name: "Script"}
	for _, s := range stmts {
		root.Children = append(root.Children, stmtTree(s))
	}
	return root
}

func identTree(role string, i Ident) *TreeNode { return tn(role, i.Name, i.Pos) }

func typeTree(t DataType) *TreeNode {
	v := t.Kind
	if t.Kind == "VARCHAR" {
		v = t.Kind + "(" + itoa(t.Length) + ")"
	}
	return tn("Type", v, t.Pos)
}

func stmtTree(s Statement) *TreeNode {
	switch s := s.(type) {
	case *CreateTable:
		n := tn("CreateTable", s.Name.Name, s.Pos)
		for _, c := range s.Columns {
			flags := ""
			if c.PrimaryKey {
				flags += " PRIMARY KEY"
			}
			if c.NotNull {
				flags += " NOT NULL"
			}
			n.Children = append(n.Children, tn("Column", c.Name.Name+flags, c.Pos, typeTree(c.Type)))
		}
		return n
	case *DropTable:
		return tn("DropTable", s.Name.Name, s.Pos)
	case *CreateIndex:
		return tn("CreateIndex", s.Name.Name, s.Pos, identTree("Table", s.Table), identTree("Column", s.Column))
	case *DropIndex:
		return tn("DropIndex", s.Name.Name, s.Pos)
	case *Insert:
		n := tn("Insert", s.Table.Name, s.Pos)
		if len(s.Columns) > 0 {
			cols := tn("Columns", "", s.Columns[0].Pos)
			for _, c := range s.Columns {
				cols.Children = append(cols.Children, identTree("Column", c))
			}
			n.Children = append(n.Children, cols)
		}
		for _, row := range s.Rows {
			r := tn("Row", "", row[0].Position())
			for _, e := range row {
				r.Children = append(r.Children, exprTree(e))
			}
			n.Children = append(n.Children, r)
		}
		return n
	case *Update:
		n := tn("Update", s.Table.Name, s.Pos)
		for _, a := range s.Sets {
			n.Children = append(n.Children, tn("Set", a.Column.Name, a.Column.Pos, exprTree(a.Value)))
		}
		if s.Where != nil {
			n.Children = append(n.Children, tn("Where", "", s.Where.Position(), exprTree(s.Where)))
		}
		return n
	case *Delete:
		n := tn("Delete", s.Table.Name, s.Pos)
		if s.Where != nil {
			n.Children = append(n.Children, tn("Where", "", s.Where.Position(), exprTree(s.Where)))
		}
		return n
	case *Select:
		return selectTree(s)
	}
	return tn("Unknown", "", s.Position())
}

func tableRefTree(role string, t TableRef) *TreeNode {
	v := t.Name.Name
	if t.Alias != "" {
		v += " AS " + t.Alias
	}
	return tn(role, v, t.Pos)
}

func selectTree(s *Select) *TreeNode {
	n := tn("Select", "", s.Pos)
	items := tn("Items", "", s.Pos)
	for _, it := range s.Items {
		if it.Star {
			items.Children = append(items.Children, tn("Star", "*", it.Pos))
			continue
		}
		c := tn("Item", it.Alias, it.Pos, exprTree(it.Expr))
		items.Children = append(items.Children, c)
	}
	n.Children = append(n.Children, items, tableRefTree("From", s.From))
	for _, j := range s.Joins {
		n.Children = append(n.Children, tn("InnerJoin", "", j.Pos, tableRefTree("Table", j.Table), tn("On", "", j.On.Position(), exprTree(j.On))))
	}
	if s.Where != nil {
		n.Children = append(n.Children, tn("Where", "", s.Where.Position(), exprTree(s.Where)))
	}
	if len(s.GroupBy) > 0 {
		g := tn("GroupBy", "", s.GroupBy[0].Position())
		for _, e := range s.GroupBy {
			g.Children = append(g.Children, exprTree(e))
		}
		n.Children = append(n.Children, g)
	}
	if s.Having != nil {
		n.Children = append(n.Children, tn("Having", "", s.Having.Position(), exprTree(s.Having)))
	}
	if len(s.OrderBy) > 0 {
		o := tn("OrderBy", "", s.OrderBy[0].Expr.Position())
		for _, it := range s.OrderBy {
			dir := "ASC"
			if it.Desc {
				dir = "DESC"
			}
			o.Children = append(o.Children, tn("Key", dir, it.Expr.Position(), exprTree(it.Expr)))
		}
		n.Children = append(n.Children, o)
	}
	return n
}

func itoa(n int) string { return strconv.Itoa(n) }

func exprTree(e Expr) *TreeNode {
	switch e := e.(type) {
	case *Literal:
		return tn("Literal", e.Kind+" "+e.Text, e.Pos)
	case *ColumnRef:
		name := e.Name
		if e.Qualifier != "" {
			name = e.Qualifier + "." + e.Name
		}
		return tn("Column", name, e.Pos)
	case *Binary:
		return tn("Binary", e.Op, e.Pos, exprTree(e.Left), exprTree(e.Right))
	case *Unary:
		return tn("Unary", e.Op, e.Pos, exprTree(e.X))
	case *FuncCall:
		if e.Star {
			return tn("Aggregate", e.Name, e.Pos, tn("Star", "*", e.Pos))
		}
		return tn("Aggregate", e.Name, e.Pos, exprTree(e.Arg))
	case *InExpr:
		name := "In"
		if e.Not {
			name = "NotIn"
		}
		n := tn(name, "", e.Pos, exprTree(e.X))
		if e.Sub != nil {
			n.Children = append(n.Children, tn("Subquery", "", e.Sub.Pos, selectTree(e.Sub)))
		}
		for _, x := range e.List {
			n.Children = append(n.Children, exprTree(x))
		}
		return n
	case *ExistsExpr:
		return tn("Exists", "", e.Pos, selectTree(e.Sub))
	case *SubqueryExpr:
		return tn("ScalarSubquery", "", e.Pos, selectTree(e.Sub))
	case *IsNull:
		name := "IsNull"
		if e.Not {
			name = "IsNotNull"
		}
		return tn(name, "", e.Pos, exprTree(e.X))
	}
	return tn("Unknown", "", e.Position())
}
