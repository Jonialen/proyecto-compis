package sql

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const metadataFile = "metadata.txt"

// ColType is the declared type of a column.
type ColType struct {
	Kind   string `json:"kind"` // INT, FLOAT, VARCHAR, BOOLEAN, DATE
	Length int    `json:"length,omitempty"`
}

func (t ColType) String() string {
	if t.Kind == "VARCHAR" {
		return "VARCHAR(" + strconv.Itoa(t.Length) + ")"
	}
	return t.Kind
}

type Column struct {
	Name       string  `json:"name"`
	Type       ColType `json:"type"`
	PrimaryKey bool    `json:"primaryKey"`
	NotNull    bool    `json:"notNull"`
}

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

func (t *Table) columnIndex(name string) int {
	for i, c := range t.Columns {
		if strings.EqualFold(c.Name, name) {
			return i
		}
	}
	return -1
}

type IndexDef struct {
	Name   string `json:"name"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

// Catalog is the schema of a database. It is persisted in metadata.txt.
type Catalog struct {
	Tables  map[string]*Table
	order   []string
	Indexes map[string]*IndexDef
	idxOrd  []string
}

func NewCatalog() *Catalog {
	return &Catalog{Tables: map[string]*Table{}, Indexes: map[string]*IndexDef{}}
}

func (c *Catalog) Table(name string) *Table { return c.Tables[strings.ToLower(name)] }

func (c *Catalog) AddTable(t *Table) {
	key := strings.ToLower(t.Name)
	c.Tables[key] = t
	c.order = append(c.order, key)
}

func (c *Catalog) DropTable(name string) {
	key := strings.ToLower(name)
	delete(c.Tables, key)
	c.order = removeKey(c.order, key)
	for ik, ix := range c.Indexes {
		if strings.EqualFold(ix.Table, name) {
			delete(c.Indexes, ik)
			c.idxOrd = removeKey(c.idxOrd, ik)
		}
	}
}

func (c *Catalog) AddIndex(ix *IndexDef) {
	key := strings.ToLower(ix.Name)
	c.Indexes[key] = ix
	c.idxOrd = append(c.idxOrd, key)
}

func (c *Catalog) DropIndex(name string) {
	key := strings.ToLower(name)
	delete(c.Indexes, key)
	c.idxOrd = removeKey(c.idxOrd, key)
}

// TableList returns the tables in creation order.
func (c *Catalog) TableList() []*Table {
	out := make([]*Table, 0, len(c.order))
	for _, k := range c.order {
		out = append(out, c.Tables[k])
	}
	return out
}

// Clone copies the schema so analysis can apply DDL without touching disk.
func (c *Catalog) Clone() *Catalog {
	n := NewCatalog()
	for _, t := range c.TableList() {
		cp := *t
		cp.Columns = append([]Column(nil), t.Columns...)
		n.AddTable(&cp)
	}
	for _, k := range c.idxOrd {
		ix := *c.Indexes[k]
		n.AddIndex(&ix)
	}
	return n
}

func removeKey(keys []string, key string) []string {
	out := keys[:0:0]
	for _, k := range keys {
		if k != key {
			out = append(out, k)
		}
	}
	return out
}

// Metadata format (metadata.txt):
//
//	TABLE estudiantes
//	COLUMN id INT PRIMARY_KEY NOT_NULL
//	COLUMN nombre VARCHAR(50)
//	END
//	INDEX idx_nombre ON estudiantes(nombre)

func (c *Catalog) metadataText() string {
	var b strings.Builder
	b.WriteString("# Compiscript SQL metadata v1\n")
	for _, t := range c.TableList() {
		fmt.Fprintf(&b, "TABLE %s\n", t.Name)
		for _, col := range t.Columns {
			fmt.Fprintf(&b, "COLUMN %s %s", col.Name, col.Type)
			if col.PrimaryKey {
				b.WriteString(" PRIMARY_KEY")
			}
			if col.NotNull {
				b.WriteString(" NOT_NULL")
			}
			b.WriteString("\n")
		}
		b.WriteString("END\n")
	}
	for _, k := range c.idxOrd {
		ix := c.Indexes[k]
		fmt.Fprintf(&b, "INDEX %s ON %s(%s)\n", ix.Name, ix.Table, ix.Column)
	}
	return b.String()
}

// LoadCatalog reads dir/metadata.txt; a missing file is an empty database.
func LoadCatalog(dir string) (*Catalog, error) {
	cat := NewCatalog()
	f, err := os.Open(filepath.Join(dir, metadataFile))
	if os.IsNotExist(err) {
		return cat, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cur *Table
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		bad := func(msg string) error { return fmt.Errorf("%s línea %d: %s", metadataFile, n, msg) }
		switch fields[0] {
		case "TABLE":
			if cur != nil || len(fields) != 2 {
				return nil, bad("TABLE inválido")
			}
			cur = &Table{Name: fields[1]}
		case "COLUMN":
			if cur == nil || len(fields) < 3 {
				return nil, bad("COLUMN inválido")
			}
			ct, err := parseColType(fields[2])
			if err != nil {
				return nil, bad(err.Error())
			}
			col := Column{Name: fields[1], Type: ct}
			for _, flag := range fields[3:] {
				switch flag {
				case "PRIMARY_KEY":
					col.PrimaryKey = true
				case "NOT_NULL":
					col.NotNull = true
				default:
					return nil, bad("restricción desconocida " + flag)
				}
			}
			cur.Columns = append(cur.Columns, col)
		case "END":
			if cur == nil {
				return nil, bad("END sin TABLE")
			}
			cat.AddTable(cur)
			cur = nil
		case "INDEX":
			var name, table, col string
			rest := strings.TrimSpace(strings.TrimPrefix(line, "INDEX"))
			parts := strings.SplitN(rest, " ON ", 2)
			if len(parts) != 2 || !strings.HasSuffix(parts[1], ")") || !strings.Contains(parts[1], "(") {
				return nil, bad("INDEX inválido")
			}
			name = strings.TrimSpace(parts[0])
			tc := strings.TrimSuffix(parts[1], ")")
			i := strings.Index(tc, "(")
			table, col = tc[:i], tc[i+1:]
			cat.AddIndex(&IndexDef{Name: name, Table: table, Column: col})
		default:
			return nil, bad("directiva desconocida " + fields[0])
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if cur != nil {
		return nil, fmt.Errorf("%s: falta END para la tabla %s", metadataFile, cur.Name)
	}
	return cat, nil
}

func parseColType(s string) (ColType, error) {
	switch s {
	case "INT", "FLOAT", "BOOLEAN", "DATE":
		return ColType{Kind: s}, nil
	}
	if strings.HasPrefix(s, "VARCHAR(") && strings.HasSuffix(s, ")") {
		n, err := strconv.Atoi(s[len("VARCHAR(") : len(s)-1])
		if err == nil && n > 0 {
			return ColType{Kind: "VARCHAR", Length: n}, nil
		}
	}
	return ColType{}, fmt.Errorf("tipo desconocido %q", s)
}

// Save writes metadata.txt atomically (temp file + rename).
func (c *Catalog) Save(dir string) error {
	return writeFileAtomic(filepath.Join(dir, metadataFile), []byte(c.metadataText()))
}

func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// validDate reports whether s is a real calendar date in YYYY-MM-DD form.
func validDate(s string) bool {
	t, err := time.Parse("2006-01-02", s)
	return err == nil && t.Format("2006-01-02") == s
}
