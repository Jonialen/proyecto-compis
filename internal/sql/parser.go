package sql

import (
	"fmt"
	"strings"
)

type parser struct {
	toks  []Token
	pos   int
	diags []Diagnostic
	stack []*TreeNode // parse tree under construction; stack[0] is the script
	stmt  int
}

type syntaxError struct{ d Diagnostic }

// Parse builds the AST and the parse tree of a token stream. Lexical
// ILLEGAL tokens are skipped (they were already reported by Lex). After a
// syntax error the parser resynchronises at the next ';'.
func Parse(tokens []Token) ([]Statement, *TreeNode, []Diagnostic) {
	p := &parser{}
	for _, t := range tokens {
		if t.Type != "ILLEGAL" {
			p.toks = append(p.toks, t)
		}
	}
	if len(p.toks) == 0 || p.toks[len(p.toks)-1].Type != "EOF" {
		p.toks = append(p.toks, Token{Type: "EOF", Line: 1, Col: 1})
	}
	root := &TreeNode{Name: "script"}
	p.stack = []*TreeNode{root}

	var stmts []Statement
	for !p.at("EOF") {
		if p.at(";") {
			p.next()
			continue
		}
		p.stmt++
		s := p.parseStatementRecovering()
		if s != nil {
			stmts = append(stmts, s)
			if !p.at(";") && !p.at("EOF") {
				p.fail(p.peek(), "Se esperaba ';' o el fin de la entrada pero se encontró %s", describe(p.peek()))
				p.recover()
			}
		}
	}
	return stmts, root, p.diags
}

func (p *parser) parseStatementRecovering() (s Statement) {
	depth := len(p.stack)
	p.begin("statement")
	defer func() {
		if r := recover(); r != nil {
			se, ok := r.(syntaxError)
			if !ok {
				panic(r)
			}
			p.diags = append(p.diags, se.d)
			top := p.stack[depth]
			p.stack = p.stack[:depth]
			top.Children = append(top.Children, &TreeNode{Name: "error", Value: se.d.Message, Line: se.d.Line, Col: se.d.Col})
			p.recover()
			s = nil
		}
	}()
	s = p.parseStatement()
	p.end()
	return s
}

// recover skips tokens up to (not including) the next ';'.
func (p *parser) recover() {
	for !p.at(";") && !p.at("EOF") {
		p.pos++
	}
}

// --- token helpers ---

func (p *parser) peek() Token { return p.toks[p.pos] }

func (p *parser) peekAt(k int) Token {
	if p.pos+k < len(p.toks) {
		return p.toks[p.pos+k]
	}
	return p.toks[len(p.toks)-1]
}

func (p *parser) at(types ...string) bool {
	t := p.peek().Type
	for _, ty := range types {
		if t == ty {
			return true
		}
	}
	return false
}

func (p *parser) next() Token {
	t := p.toks[p.pos]
	if t.Type != "EOF" {
		p.pos++
	}
	top := p.stack[len(p.stack)-1]
	top.Children = append(top.Children, &TreeNode{Name: t.Type, Value: t.Text, Line: t.Line, Col: t.Col})
	return t
}

func (p *parser) accept(ty string) bool {
	if p.at(ty) {
		p.next()
		return true
	}
	return false
}

func describe(t Token) string {
	switch t.Type {
	case "EOF":
		return "el fin de la entrada"
	case "IDENT", "INT", "FLOAT":
		return fmt.Sprintf("'%s'", t.Text)
	case "STRING":
		return fmt.Sprintf("la cadena '%s'", t.Text)
	}
	if keywords[t.Type] {
		return fmt.Sprintf("la palabra reservada '%s'", t.Text)
	}
	return fmt.Sprintf("'%s'", t.Text)
}

func (p *parser) fail(t Token, format string, args ...any) {
	p.diags = append(p.diags, Diagnostic{Phase: "syntax", Message: fmt.Sprintf(format, args...), Line: t.Line, Col: t.Col, Statement: p.stmt})
}

func (p *parser) abort(t Token, format string, args ...any) {
	panic(syntaxError{Diagnostic{Phase: "syntax", Message: fmt.Sprintf(format, args...), Line: t.Line, Col: t.Col, Statement: p.stmt}})
}

func (p *parser) expect(ty, what string) Token {
	if !p.at(ty) {
		p.abort(p.peek(), "Se esperaba %s pero se encontró %s", what, describe(p.peek()))
	}
	return p.next()
}

func (p *parser) expectIdent(what string) Ident {
	t := p.expect("IDENT", what)
	return Ident{Name: t.Text, Pos: Pos{t.Line, t.Col}}
}

func posOf(t Token) Pos { return Pos{t.Line, t.Col} }

// --- parse tree helpers ---

func (p *parser) begin(rule string) {
	n := &TreeNode{Name: rule}
	top := p.stack[len(p.stack)-1]
	top.Children = append(top.Children, n)
	p.stack = append(p.stack, n)
	n.Line, n.Col = p.peek().Line, p.peek().Col
}

func (p *parser) end() { p.stack = p.stack[:len(p.stack)-1] }

func (p *parser) mark() int { return len(p.stack[len(p.stack)-1].Children) }

// wrap groups the nodes added since mark under a new rule node.
func (p *parser) wrap(mark int, rule string) {
	top := p.stack[len(p.stack)-1]
	moved := append([]*TreeNode(nil), top.Children[mark:]...)
	n := &TreeNode{Name: rule, Children: moved}
	if len(moved) > 0 {
		n.Line, n.Col = moved[0].Line, moved[0].Col
	}
	top.Children = append(top.Children[:mark], n)
}

// --- statements ---

func (p *parser) parseStatement() Statement {
	t := p.peek()
	switch t.Type {
	case "CREATE":
		return p.parseCreate()
	case "DROP":
		return p.parseDrop()
	case "INSERT":
		return p.parseInsert()
	case "UPDATE":
		return p.parseUpdate()
	case "DELETE":
		return p.parseDelete()
	case "SELECT":
		return p.parseSelect()
	}
	p.abort(t, "Se esperaba una sentencia (CREATE, DROP, INSERT, UPDATE, DELETE o SELECT) pero se encontró %s", describe(t))
	return nil
}

func (p *parser) parseCreate() Statement {
	start := p.next()
	switch {
	case p.at("TABLE"):
		p.next()
		name := p.expectIdent("el nombre de la tabla")
		p.expect("(", "'('")
		ct := &CreateTable{Pos: posOf(start), Name: name}
		for {
			ct.Columns = append(ct.Columns, p.parseColumnDef())
			if !p.accept(",") {
				break
			}
		}
		p.expect(")", "')' o ','")
		return ct
	case p.at("INDEX"):
		p.next()
		name := p.expectIdent("el nombre del índice")
		p.expect("ON", "'ON'")
		table := p.expectIdent("el nombre de la tabla")
		p.expect("(", "'('")
		col := p.expectIdent("el nombre de la columna")
		p.expect(")", "')'")
		return &CreateIndex{Pos: posOf(start), Name: name, Table: table, Column: col}
	}
	p.abort(p.peek(), "Se esperaba TABLE o INDEX después de CREATE pero se encontró %s", describe(p.peek()))
	return nil
}

func (p *parser) parseColumnDef() ColumnDef {
	p.begin("column_def")
	defer p.end()
	name := p.expectIdent("el nombre de la columna")
	cd := ColumnDef{Pos: name.Pos, Name: name, Type: p.parseType()}
	for {
		switch {
		case p.at("PRIMARY"):
			p.next()
			p.expect("KEY", "'KEY' después de PRIMARY")
			cd.PrimaryKey = true
		case p.at("NOT"):
			p.next()
			p.expect("NULL", "'NULL' después de NOT")
			cd.NotNull = true
		default:
			return cd
		}
	}
}

func (p *parser) parseType() DataType {
	p.begin("type")
	defer p.end()
	t := p.peek()
	switch t.Type {
	case "INT", "FLOAT", "BOOLEAN", "DATE":
		p.next()
		return DataType{Kind: t.Type, Pos: posOf(t)}
	case "VARCHAR":
		p.next()
		p.expect("(", "'(' después de VARCHAR")
		n := p.expect("INT", "la longitud de VARCHAR")
		p.expect(")", "')'")
		length := 0
		fmt.Sscanf(n.Text, "%d", &length)
		return DataType{Kind: "VARCHAR", Length: length, Pos: posOf(t)}
	}
	p.abort(t, "Se esperaba un tipo de dato (INT, FLOAT, VARCHAR(n), BOOLEAN o DATE) pero se encontró %s", describe(t))
	return DataType{}
}

func (p *parser) parseDrop() Statement {
	start := p.next()
	switch {
	case p.at("TABLE"):
		p.next()
		return &DropTable{Pos: posOf(start), Name: p.expectIdent("el nombre de la tabla")}
	case p.at("INDEX"):
		p.next()
		return &DropIndex{Pos: posOf(start), Name: p.expectIdent("el nombre del índice")}
	}
	p.abort(p.peek(), "Se esperaba TABLE o INDEX después de DROP pero se encontró %s", describe(p.peek()))
	return nil
}

func (p *parser) parseInsert() Statement {
	start := p.next()
	p.expect("INTO", "'INTO'")
	ins := &Insert{Pos: posOf(start), Table: p.expectIdent("el nombre de la tabla")}
	if p.accept("(") {
		for {
			ins.Columns = append(ins.Columns, p.expectIdent("el nombre de la columna"))
			if !p.accept(",") {
				break
			}
		}
		p.expect(")", "')' o ','")
	}
	p.expect("VALUES", "'VALUES'")
	for {
		p.begin("row")
		p.expect("(", "'('")
		var row []Expr
		for {
			row = append(row, p.parseExpr())
			if !p.accept(",") {
				break
			}
		}
		p.expect(")", "')' o ','")
		p.end()
		ins.Rows = append(ins.Rows, row)
		if !p.accept(",") {
			break
		}
	}
	return ins
}

func (p *parser) parseUpdate() Statement {
	start := p.next()
	u := &Update{Pos: posOf(start), Table: p.expectIdent("el nombre de la tabla")}
	p.expect("SET", "'SET'")
	for {
		p.begin("assignment")
		col := p.expectIdent("el nombre de la columna")
		p.expect("=", "'='")
		u.Sets = append(u.Sets, Assignment{Column: col, Value: p.parseExpr()})
		p.end()
		if !p.accept(",") {
			break
		}
	}
	u.Where = p.parseOptionalWhere()
	return u
}

func (p *parser) parseDelete() Statement {
	start := p.next()
	p.expect("FROM", "'FROM'")
	d := &Delete{Pos: posOf(start), Table: p.expectIdent("el nombre de la tabla")}
	d.Where = p.parseOptionalWhere()
	return d
}

func (p *parser) parseOptionalWhere() Expr {
	if !p.at("WHERE") {
		return nil
	}
	p.begin("where_clause")
	defer p.end()
	p.next()
	return p.parseExpr()
}

func (p *parser) parseSelect() *Select {
	p.begin("select_stmt")
	defer p.end()
	start := p.next()
	s := &Select{Pos: posOf(start)}

	p.begin("select_list")
	for {
		s.Items = append(s.Items, p.parseSelectItem())
		if !p.accept(",") {
			break
		}
	}
	p.end()

	p.begin("from_clause")
	p.expect("FROM", "'FROM'")
	s.From = p.parseTableRef()
	p.end()

	for p.at("INNER", "JOIN") {
		p.begin("join_clause")
		jt := p.peek()
		p.accept("INNER")
		p.expect("JOIN", "'JOIN'")
		tr := p.parseTableRef()
		p.expect("ON", "'ON'")
		s.Joins = append(s.Joins, Join{Pos: posOf(jt), Table: tr, On: p.parseExpr()})
		p.end()
	}
	s.Where = p.parseOptionalWhere()
	if p.at("GROUP") {
		p.begin("group_by_clause")
		p.next()
		p.expect("BY", "'BY' después de GROUP")
		for {
			s.GroupBy = append(s.GroupBy, p.parseExpr())
			if !p.accept(",") {
				break
			}
		}
		p.end()
	}
	if p.at("HAVING") {
		p.begin("having_clause")
		p.next()
		s.Having = p.parseExpr()
		p.end()
	}
	if p.at("ORDER") {
		p.begin("order_by_clause")
		p.next()
		p.expect("BY", "'BY' después de ORDER")
		for {
			it := OrderItem{Expr: p.parseExpr()}
			if p.accept("DESC") {
				it.Desc = true
			} else {
				p.accept("ASC")
			}
			s.OrderBy = append(s.OrderBy, it)
			if !p.accept(",") {
				break
			}
		}
		p.end()
	}
	return s
}

func (p *parser) parseSelectItem() SelectItem {
	p.begin("select_item")
	defer p.end()
	t := p.peek()
	if p.accept("*") {
		return SelectItem{Pos: posOf(t), Star: true}
	}
	it := SelectItem{Pos: posOf(t), Expr: p.parseExpr()}
	if p.accept("AS") {
		it.Alias = p.expectIdent("el alias").Name
	} else if p.at("IDENT") {
		it.Alias = p.next().Text
	}
	return it
}

func (p *parser) parseTableRef() TableRef {
	p.begin("table_ref")
	defer p.end()
	name := p.expectIdent("el nombre de la tabla")
	tr := TableRef{Pos: name.Pos, Name: name}
	if p.accept("AS") {
		tr.Alias = p.expectIdent("el alias de la tabla").Name
	} else if p.at("IDENT") {
		tr.Alias = p.next().Text
	}
	return tr
}

// --- expressions (OR < AND < NOT < comparison < additive < multiplicative) ---

func (p *parser) parseExpr() Expr { return p.parseOr() }

func (p *parser) parseOr() Expr {
	m := p.mark()
	left := p.parseAnd()
	for p.at("OR") {
		op := p.next()
		right := p.parseAnd()
		left = &Binary{Pos: posOf(op), Op: "OR", Left: left, Right: right}
		p.wrap(m, "or_expr")
	}
	return left
}

func (p *parser) parseAnd() Expr {
	m := p.mark()
	left := p.parseNot()
	for p.at("AND") {
		op := p.next()
		right := p.parseNot()
		left = &Binary{Pos: posOf(op), Op: "AND", Left: left, Right: right}
		p.wrap(m, "and_expr")
	}
	return left
}

func (p *parser) parseNot() Expr {
	if p.at("NOT") {
		m := p.mark()
		op := p.next()
		x := p.parseNot()
		p.wrap(m, "not_expr")
		return &Unary{Pos: posOf(op), Op: "NOT", X: x}
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() Expr {
	m := p.mark()
	left := p.parseAdditive()
	t := p.peek()
	switch {
	case p.at("=", "<>", "<", "<=", ">", ">="):
		p.next()
		right := p.parseAdditive()
		p.wrap(m, "comparison")
		return &Binary{Pos: posOf(t), Op: t.Type, Left: left, Right: right}
	case p.at("IS"):
		p.next()
		not := p.accept("NOT")
		p.expect("NULL", "'NULL'")
		p.wrap(m, "is_null")
		return &IsNull{Pos: posOf(t), X: left, Not: not}
	case p.at("IN") || (p.at("NOT") && p.peekAt(1).Type == "IN"):
		not := p.accept("NOT")
		p.next() // IN
		p.expect("(", "'(' después de IN")
		in := &InExpr{Pos: posOf(t), X: left, Not: not}
		if p.at("SELECT") {
			in.Sub = p.parseSelect()
		} else {
			for {
				in.List = append(in.List, p.parseExpr())
				if !p.accept(",") {
					break
				}
			}
		}
		p.expect(")", "')'")
		p.wrap(m, "in_expr")
		return in
	}
	return left
}

func (p *parser) parseAdditive() Expr {
	m := p.mark()
	left := p.parseMultiplicative()
	for p.at("+", "-") {
		op := p.next()
		right := p.parseMultiplicative()
		left = &Binary{Pos: posOf(op), Op: op.Type, Left: left, Right: right}
		p.wrap(m, "additive_expr")
	}
	return left
}

func (p *parser) parseMultiplicative() Expr {
	m := p.mark()
	left := p.parseUnary()
	for p.at("*", "/") {
		op := p.next()
		right := p.parseUnary()
		left = &Binary{Pos: posOf(op), Op: op.Type, Left: left, Right: right}
		p.wrap(m, "multiplicative_expr")
	}
	return left
}

func (p *parser) parseUnary() Expr {
	if p.at("-") {
		m := p.mark()
		op := p.next()
		x := p.parseUnary()
		p.wrap(m, "unary_expr")
		return &Unary{Pos: posOf(op), Op: "-", X: x}
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() Expr {
	t := p.peek()
	switch t.Type {
	case "INT", "FLOAT", "STRING":
		p.next()
		return &Literal{Pos: posOf(t), Kind: t.Type, Text: t.Text}
	case "TRUE", "FALSE":
		p.next()
		return &Literal{Pos: posOf(t), Kind: "BOOLEAN", Text: strings.ToUpper(t.Text)}
	case "NULL":
		p.next()
		return &Literal{Pos: posOf(t), Kind: "NULL", Text: "NULL"}
	case "(":
		m := p.mark()
		p.next()
		if p.at("SELECT") {
			sub := p.parseSelect()
			p.expect(")", "')'")
			p.wrap(m, "scalar_subquery")
			return &SubqueryExpr{Pos: posOf(t), Sub: sub}
		}
		e := p.parseExpr()
		p.expect(")", "')'")
		p.wrap(m, "paren_expr")
		return e
	case "EXISTS":
		m := p.mark()
		p.next()
		p.expect("(", "'(' después de EXISTS")
		sub := p.parseSelect()
		p.expect(")", "')'")
		p.wrap(m, "exists_expr")
		return &ExistsExpr{Pos: posOf(t), Sub: sub}
	case "COUNT", "SUM", "AVG", "MIN", "MAX":
		m := p.mark()
		p.next()
		p.expect("(", "'(' después de la función de agregación")
		fc := &FuncCall{Pos: posOf(t), Name: t.Type}
		if p.at("*") {
			if t.Type != "COUNT" {
				p.abort(p.peek(), "'*' solo es válido dentro de COUNT")
			}
			p.next()
			fc.Star = true
		} else {
			fc.Arg = p.parseExpr()
		}
		p.expect(")", "')'")
		p.wrap(m, "aggregate")
		return fc
	case "IDENT":
		m := p.mark()
		p.next()
		if p.at(".") {
			p.next()
			col := p.expect("IDENT", "el nombre de la columna después de '.'")
			p.wrap(m, "qualified_column")
			return &ColumnRef{Pos: posOf(t), Qualifier: t.Text, Name: col.Text}
		}
		return &ColumnRef{Pos: posOf(t), Name: t.Text}
	}
	p.abort(t, "Se esperaba una expresión pero se encontró %s", describe(t))
	return nil
}
