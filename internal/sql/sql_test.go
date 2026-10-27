package sql

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func types(toks []Token) []string {
	var out []string
	for _, t := range toks {
		out = append(out, t.Type)
	}
	return out
}

func TestLexTokensAndPositions(t *testing.T) {
	toks, diags := Lex("SELECT nombre,\n  edad >= 18 -- fin\nFROM t WHERE x <> 'a''b' AND y != 2.5;")
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	got := strings.Join(types(toks), " ")
	want := "SELECT IDENT , IDENT >= INT FROM IDENT WHERE IDENT <> STRING AND IDENT <> FLOAT ; EOF"
	if got != want {
		t.Fatalf("types:\n got %s\nwant %s", got, want)
	}
	if toks[3].Line != 2 || toks[3].Col != 3 {
		t.Errorf("edad position = %d:%d, want 2:3", toks[3].Line, toks[3].Col)
	}
	if toks[11].Text != "a'b" {
		t.Errorf("escaped quote text = %q", toks[11].Text)
	}
}

func TestLexKeywordsAreCaseInsensitive(t *testing.T) {
	toks, _ := Lex("select From wHeRe")
	if got := strings.Join(types(toks), " "); got != "SELECT FROM WHERE EOF" {
		t.Fatalf("got %s", got)
	}
}

func TestLexErrors(t *testing.T) {
	cases := map[string]string{
		"SELECT 'abc":    "Cadena de texto sin cerrar",
		"SELECT @":       "Carácter no reconocido",
		"SELECT 12abc":   "Número mal formado",
		"SELECT /* x":    "Comentario de bloque sin cerrar",
		"SELECT a ! b":   "Carácter no reconocido",
		"SELECT 1 # two": "Carácter no reconocido",
	}
	for src, msg := range cases {
		_, diags := Lex(src)
		if len(diags) != 1 || diags[0].Phase != "lexical" || !strings.Contains(diags[0].Message, msg) {
			t.Errorf("%q: diagnostics = %v, want %q", src, diags, msg)
		}
	}
	_, diags := Lex("SELECT\n  @")
	if diags[0].Line != 2 || diags[0].Col != 3 {
		t.Errorf("position = %d:%d, want 2:3", diags[0].Line, diags[0].Col)
	}
}

func parse(t *testing.T, src string) ([]Statement, *TreeNode, []Diagnostic) {
	t.Helper()
	toks, lex := Lex(src)
	if len(lex) != 0 {
		t.Fatalf("lex errors: %v", lex)
	}
	return Parse(toks)
}

func TestParseAllRequiredStatements(t *testing.T) {
	src := `
CREATE TABLE estudiantes (id INT PRIMARY KEY, nombre VARCHAR(50) NOT NULL, nota FLOAT, activo BOOLEAN, nacimiento DATE);
CREATE INDEX idx_nombre ON estudiantes (nombre);
DROP INDEX idx_nombre;
INSERT INTO estudiantes (id, nombre) VALUES (1, 'Ana'), (2, 'Luis');
UPDATE estudiantes SET nota = nota + 1, activo = TRUE WHERE id = 1 OR id = 2;
DELETE FROM estudiantes WHERE nota < 3 AND NOT activo;
SELECT e.nombre, COUNT(*) AS total, AVG(nota)
  FROM estudiantes e INNER JOIN inscripciones i ON i.estudiante_id = e.id
  WHERE e.id IN (SELECT estudiante_id FROM inscripciones WHERE curso_id = 3)
    AND EXISTS (SELECT id FROM cursos) AND nota > (SELECT MAX(nota) FROM estudiantes)
    AND e.id NOT IN (1, 2) AND nombre IS NOT NULL
  GROUP BY e.nombre HAVING SUM(nota) >= 10 ORDER BY total DESC, e.nombre;
DROP TABLE estudiantes;`
	stmts, tree, diags := parse(t, src)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(stmts) != 8 {
		t.Fatalf("got %d statements, want 8", len(stmts))
	}
	count := 0
	for _, c := range tree.Children {
		if c.Name == "statement" {
			count++
		}
	}
	if count != 8 {
		t.Fatalf("parse tree has %d statement nodes, want 8", count)
	}
	sel := stmts[6].(*Select)
	if len(sel.Joins) != 1 || len(sel.GroupBy) != 1 || sel.Having == nil || len(sel.OrderBy) != 2 || !sel.OrderBy[0].Desc {
		t.Errorf("select clauses not parsed: %+v", sel)
	}
	ast := ASTTree(stmts)
	if ast.Children[0].Name != "CreateTable" || ast.Children[6].Name != "Select" {
		t.Errorf("unexpected AST roots: %s, %s", ast.Children[0].Name, ast.Children[6].Name)
	}
}

func TestParseOperatorPrecedence(t *testing.T) {
	stmts, _, diags := parse(t, "SELECT a FROM t WHERE a = 1 OR b = 2 AND c = 3;")
	if len(diags) != 0 {
		t.Fatal(diags)
	}
	w := stmts[0].(*Select).Where.(*Binary)
	if w.Op != "OR" || w.Right.(*Binary).Op != "AND" {
		t.Errorf("AND must bind tighter than OR: %+v", w)
	}
}

func TestParseSyntaxErrorsAreLocatedAndRecovered(t *testing.T) {
	src := "SELECT FROM t;\nINSERT INTO t VALUES (1;\nDROP TABLE t;"
	stmts, tree, diags := parse(t, src)
	if len(diags) != 2 {
		t.Fatalf("diagnostics = %v, want 2", diags)
	}
	if diags[0].Line != 1 || diags[0].Col != 8 || !strings.Contains(diags[0].Message, "Se esperaba una expresión") {
		t.Errorf("first error = %v", diags[0])
	}
	if diags[1].Line != 2 || !strings.Contains(diags[1].Message, "')' o ','") {
		t.Errorf("second error = %v", diags[1])
	}
	if len(stmts) != 1 {
		t.Errorf("statements after recovery = %d, want 1 (the DROP)", len(stmts))
	}
	first := tree.Children[0]
	if first.Children[len(first.Children)-1].Name != "error" {
		t.Errorf("failed statement must end with an error node in the parse tree")
	}
}

func TestParseMissingSemicolonBetweenStatements(t *testing.T) {
	_, _, diags := parse(t, "DROP TABLE a DROP TABLE b")
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "';'") {
		t.Fatalf("diagnostics = %v", diags)
	}
}

func TestParseRejectsKeywordAsIdentifierAndStarInSum(t *testing.T) {
	for src, msg := range map[string]string{
		"CREATE TABLE select (a INT);": "palabra reservada 'select'",
		"SELECT SUM(*) FROM t;":        "'*' solo es válido dentro de COUNT",
		"CREATE TABLE t (a TEXT);":     "tipo de dato",
		"CREATE TABLE t (a VARCHAR);":  "'(' después de VARCHAR",
	} {
		_, _, diags := parse(t, src)
		if len(diags) == 0 || !strings.Contains(diags[0].Message, msg) {
			t.Errorf("%q: diagnostics = %v, want %q", src, diags, msg)
		}
	}
}

// --- semantic analysis and execution through the engine ---

func newEngine(t *testing.T) *Engine {
	t.Helper()
	return &Engine{Dir: filepath.Join(t.TempDir(), "db")}
}

const schema = `
CREATE TABLE estudiantes (id INT PRIMARY KEY, nombre VARCHAR(10) NOT NULL, nota FLOAT, activo BOOLEAN, nacimiento DATE);
CREATE TABLE cursos (id INT PRIMARY KEY, nombre VARCHAR(20));
CREATE TABLE inscripciones (estudiante_id INT, curso_id INT);
`

func analyzeMessages(t *testing.T, e *Engine, src string) []string {
	t.Helper()
	rep := e.Run(src, false)
	var out []string
	for _, d := range rep.Diagnostics {
		out = append(out, d.Phase+": "+d.Message)
	}
	return out
}

func TestSemanticErrors(t *testing.T) {
	e := newEngine(t)
	e.Run(schema, true)
	cases := []struct{ sql, want string }{
		{"DROP TABLE nada;", `La tabla "nada" no existe`},
		{"CREATE TABLE cursos (a INT);", `La tabla "cursos" ya existe`},
		{"CREATE TABLE x (a INT, A FLOAT);", "duplicada"},
		{"CREATE TABLE x (a INT PRIMARY KEY, b INT PRIMARY KEY);", "más de una PRIMARY KEY"},
		{"CREATE TABLE x (a VARCHAR(0));", "mayor que 0"},
		{"INSERT INTO nada VALUES (1);", `La tabla "nada" no existe`},
		{"INSERT INTO cursos (id, zzz) VALUES (1, 'a');", `La columna "zzz" no existe`},
		{"INSERT INTO cursos (id, id) VALUES (1, 2);", "se repite"},
		{"INSERT INTO cursos VALUES (1);", "tiene 1 valores pero se esperaban 2"},
		{"INSERT INTO cursos VALUES ('x', 'a');", "Tipos incompatibles"},
		{"INSERT INTO estudiantes (id) VALUES (1);", "NOT NULL"},
		{"INSERT INTO estudiantes VALUES (1, NULL, 1, TRUE, '2000-01-01');", "no admite NULL"},
		{"INSERT INTO estudiantes VALUES (1, 'nombre muy largo', 1, TRUE, '2000-01-01');", "excede la longitud"},
		{"INSERT INTO estudiantes VALUES (1, 'a', 1, TRUE, '2000-02-31');", "Fecha inválida"},
		{"INSERT INTO estudiantes VALUES (1, 'a', 1.5, 1, '2000-01-01');", "Tipos incompatibles"},
		{"INSERT INTO cursos VALUES (id, 'a');", "Referencia inválida"},
		{"INSERT INTO cursos VALUES (1 / 0, 'a');", "División por cero"},
		{"SELECT x FROM estudiantes;", `La columna "x" no existe`},
		{"SELECT id FROM estudiantes e INNER JOIN cursos c ON e.id = c.id;", "Referencia ambigua"},
		{"SELECT z.id FROM estudiantes;", "no es una tabla ni un alias"},
		{"SELECT nombre FROM estudiantes WHERE nota = 'a';", "Tipos incompatibles"},
		{"SELECT nombre FROM estudiantes WHERE nacimiento > 'ayer';", "Tipos incompatibles"},
		{"SELECT nombre FROM estudiantes WHERE nota;", "debe ser de tipo BOOLEAN"},
		{"SELECT nombre FROM estudiantes WHERE COUNT(*) > 1;", "no se permite en esta cláusula"},
		{"SELECT nombre, COUNT(*) FROM estudiantes;", "debe aparecer en GROUP BY"},
		{"SELECT SUM(nombre) FROM estudiantes;", "SUM requiere una columna numérica"},
		{"SELECT nombre FROM estudiantes WHERE id IN (SELECT id, nombre FROM cursos);", "exactamente una columna"},
		{"SELECT nombre FROM estudiantes WHERE nota = (SELECT id, nombre FROM cursos);", "Subconsulta escalar inválida"},
		{"SELECT nombre FROM estudiantes WHERE id IN (SELECT nombre FROM cursos);", "Tipos incompatibles"},
		{"UPDATE estudiantes SET zzz = 1;", `La columna "zzz" no existe`},
		{"UPDATE estudiantes SET nombre = 5;", "Tipos incompatibles"},
		{"DELETE FROM estudiantes WHERE zzz = 1;", `La columna "zzz" no existe`},
		{"CREATE INDEX i ON nada (a);", `La tabla "nada" no existe`},
		{"CREATE INDEX i ON cursos (zzz);", `La columna "zzz" no existe`},
		{"DROP INDEX i;", `El índice "i" no existe`},
		{"CREATE INDEX i ON cursos (id); CREATE INDEX i ON cursos (nombre);", "ya existe"},
		{"CREATE INDEX i ON cursos (id); CREATE INDEX j ON cursos (id);", "Índice duplicado"},
	}
	for _, c := range cases {
		msgs := analyzeMessages(t, e, c.sql)
		if len(msgs) == 0 || !strings.Contains(strings.Join(msgs, "|"), c.want) {
			t.Errorf("%s\n  got %v\n  want substring %q", c.sql, msgs, c.want)
		}
	}
}

func TestSemanticReportsLocation(t *testing.T) {
	e := newEngine(t)
	e.Run(schema, true)
	rep := e.Run("INSERT INTO cursos\nVALUES (1, 5);", false)
	if len(rep.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %v", rep.Diagnostics)
	}
	d := rep.Diagnostics[0]
	if d.Phase != "semantic" || d.Line != 2 || d.Col != 12 || d.Statement != 1 {
		t.Errorf("diagnostic = %+v, want semantic at 2:12 in statement 1", d)
	}
}

func TestSemanticAcceptsValidQueries(t *testing.T) {
	e := newEngine(t)
	e.Run(schema, true)
	valid := []string{
		"SELECT nombre FROM estudiantes WHERE nota >= 3 AND nacimiento >= '2000-01-01';",
		"SELECT e.nombre, c.nombre FROM estudiantes e INNER JOIN inscripciones i ON i.estudiante_id = e.id INNER JOIN cursos c ON c.id = i.curso_id;",
		"SELECT nombre FROM estudiantes WHERE id IN (SELECT estudiante_id FROM inscripciones WHERE curso_id = 3);",
		"SELECT nombre FROM estudiantes e WHERE EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = e.id);",
		"SELECT nombre FROM estudiantes WHERE nota > (SELECT AVG(nota) FROM estudiantes);",
		"SELECT activo, COUNT(*), AVG(nota), MAX(nombre) FROM estudiantes GROUP BY activo HAVING COUNT(*) > 1 ORDER BY activo;",
		"SELECT nombre AS n FROM estudiantes ORDER BY n DESC;",
		"UPDATE estudiantes SET nota = nota * 2, nacimiento = '2001-05-06' WHERE id = 1;",
		"DELETE FROM estudiantes WHERE activo = FALSE OR nota IS NULL;",
		"CREATE TABLE a (x INT); INSERT INTO a VALUES (1); DROP TABLE a;",
		"CREATE INDEX i ON cursos (id); DROP INDEX i;",
	}
	for _, q := range valid {
		if msgs := analyzeMessages(t, e, q); len(msgs) != 0 {
			t.Errorf("%s\n  unexpected: %v", q, msgs)
		}
	}
}

func TestAnalyzeOnlyNeverTouchesDisk(t *testing.T) {
	e := newEngine(t)
	rep := e.Run("CREATE TABLE a (x INT); INSERT INTO a VALUES (1);", false)
	if len(rep.Diagnostics) != 0 || rep.Executed {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := os.Stat(e.Dir); !os.IsNotExist(err) {
		t.Errorf("analysis created the database directory: %v", err)
	}
}

func TestExecuteCreateInsertDropPersistsInFiles(t *testing.T) {
	e := newEngine(t)
	rep := e.Run(schema, true)
	if len(rep.Diagnostics) != 0 || len(rep.Results) != 3 {
		t.Fatalf("report = %+v", rep)
	}
	rep = e.Run(`INSERT INTO estudiantes VALUES (1, 'Ana', 9, TRUE, '2001-03-04'), (2, 'Luis', NULL, FALSE, NULL);
INSERT INTO estudiantes (id, nombre, nota) VALUES (3, '\path', 1 + 2 * 3);`, true)
	if len(rep.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %v", rep.Diagnostics)
	}
	if rep.Results[0].RowsAffected != 2 || rep.Results[1].RowsAffected != 1 {
		t.Errorf("rows affected = %+v", rep.Results)
	}

	meta, err := os.ReadFile(filepath.Join(e.Dir, "metadata.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TABLE estudiantes", "COLUMN id INT PRIMARY_KEY NOT_NULL", "COLUMN nombre VARCHAR(10) NOT_NULL", "COLUMN nacimiento DATE", "END"} {
		if !strings.Contains(string(meta), want) {
			t.Errorf("metadata.txt missing %q:\n%s", want, meta)
		}
	}
	csvData, err := os.ReadFile(filepath.Join(e.Dir, "estudiantes.csv"))
	if err != nil {
		t.Fatal(err)
	}
	want := "id,nombre,nota,activo,nacimiento\n1,Ana,9,TRUE,2001-03-04\n2,Luis,\\N,FALSE,\\N\n3,\\\\path,7,\\N,\\N\n"
	if string(csvData) != want {
		t.Errorf("estudiantes.csv:\n got %q\nwant %q", csvData, want)
	}

	cat, err := LoadCatalog(e.Dir)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := readRows(e.Dir, cat.Table("estudiantes"))
	if err != nil || len(rows) != 3 || rows[2][1] != `\path` || rows[1][2] != nil || rows[0][2] != 9.0 {
		t.Errorf("round trip rows = %v, err = %v", rows, err)
	}

	rep = e.Run("DROP TABLE cursos;", true)
	if len(rep.Diagnostics) != 0 {
		t.Fatal(rep.Diagnostics)
	}
	if _, err := os.Stat(filepath.Join(e.Dir, "cursos.csv")); !os.IsNotExist(err) {
		t.Errorf("cursos.csv should be deleted: %v", err)
	}
	if got := analyzeMessages(t, e, "SELECT id FROM cursos;"); len(got) == 0 {
		t.Error("dropped table must be unknown afterwards")
	}
	if len(rep.Schema) != 2 {
		t.Errorf("schema after drop = %+v", rep.Schema)
	}
}

func TestExecutionRefusesScriptsWithErrorsAndRunsNothing(t *testing.T) {
	e := newEngine(t)
	rep := e.Run("CREATE TABLE a (x INT); INSERT INTO a VALUES ('mal');", true)
	if len(rep.Diagnostics) != 1 || rep.Executed || len(rep.Results) != 0 {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := os.Stat(e.Dir); !os.IsNotExist(err) {
		t.Errorf("a rejected script must not create files: %v", err)
	}
}

func TestPrimaryKeyViolationIsAtomic(t *testing.T) {
	e := newEngine(t)
	e.Run("CREATE TABLE a (id INT PRIMARY KEY, v INT);", true)
	e.Run("INSERT INTO a VALUES (1, 10);", true)
	rep := e.Run("INSERT INTO a VALUES (2, 20), (1, 30);", true)
	if len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != "execution" || !strings.Contains(rep.Diagnostics[0].Message, "PRIMARY KEY") {
		t.Fatalf("diagnostics = %v", rep.Diagnostics)
	}
	rep = e.Run("INSERT INTO a VALUES (5, 1), (5, 2);", true)
	if len(rep.Diagnostics) != 1 {
		t.Fatalf("duplicate inside one INSERT must fail: %v", rep.Diagnostics)
	}
	if rep.Schema[0].Rows != 1 {
		t.Errorf("rows = %d, want 1 (failed inserts are all-or-nothing)", rep.Schema[0].Rows)
	}
}

func TestStatementsPastImplementedStageFailAtExecutionAfterEarlierOnes(t *testing.T) {
	e := newEngine(t)
	rep := e.Run("CREATE TABLE a (x INT); SELECT x FROM a;", true)
	if len(rep.Results) != 1 || len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != "execution" || rep.Diagnostics[0].Statement != 2 {
		t.Fatalf("report = %+v", rep)
	}
	if !strings.Contains(rep.Diagnostics[0].Message, "aún no se ejecuta") {
		t.Errorf("message = %q", rep.Diagnostics[0].Message)
	}
}

func TestCorruptMetadataIsReported(t *testing.T) {
	e := newEngine(t)
	os.MkdirAll(e.Dir, 0o755)
	os.WriteFile(filepath.Join(e.Dir, "metadata.txt"), []byte("TABLE a\nCOLUMN x BLOB\nEND\n"), 0o644)
	rep := e.Run("DROP TABLE a;", true)
	if len(rep.Diagnostics) != 1 || !strings.Contains(rep.Diagnostics[0].Message, "metadata.txt línea 2") {
		t.Fatalf("diagnostics = %v", rep.Diagnostics)
	}
}

func TestMetadataRoundTripWithIndexes(t *testing.T) {
	cat := NewCatalog()
	cat.AddTable(&Table{Name: "t", Columns: []Column{{Name: "a", Type: ColType{Kind: "VARCHAR", Length: 5}, NotNull: true}, {Name: "b", Type: ColType{Kind: "DATE"}}}})
	cat.AddIndex(&IndexDef{Name: "ix", Table: "t", Column: "a"})
	dir := t.TempDir()
	if err := cat.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.metadataText() != cat.metadataText() {
		t.Errorf("round trip changed metadata:\n%s\nvs\n%s", got.metadataText(), cat.metadataText())
	}
}
