// Command sql runs a SQL script against a file-based database.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"genanalex/internal/sql"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sql", flag.ContinueOnError)
	fs.SetOutput(stderr)
	db := fs.String("db", sql.DefaultDir(), "directorio de la base de datos")
	analyze := fs.Bool("analyze", false, "solo analizar; no ejecutar ni modificar la base de datos")
	format := fs.String("format", "text", "formato de salida: text o json")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Uso: sql [-db dir] [-analyze] [-format text|json] [archivo.sql]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "formato desconocido %q\n", *format)
		return 2
	}

	var src []byte
	var err error
	name := "<stdin>"
	switch fs.NArg() {
	case 0:
		src, err = io.ReadAll(stdin)
	case 1:
		name = fs.Arg(0)
		src, err = os.ReadFile(name)
	default:
		fs.Usage()
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	rep := (&sql.Engine{Dir: *db}).Run(string(src), !*analyze)
	if *format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(rep)
	} else {
		for _, r := range rep.Results {
			fmt.Fprintf(stdout, "[%d] %s\n", r.Statement, r.Message)
		}
		for _, d := range rep.Diagnostics {
			fmt.Fprintf(stderr, "%s:%s\n", name, d)
		}
		if len(rep.Diagnostics) == 0 && !rep.Executed {
			fmt.Fprintln(stdout, "Análisis correcto: sin errores léxicos, sintácticos ni semánticos.")
		}
	}
	if len(rep.Diagnostics) > 0 {
		return 1
	}
	return 0
}
