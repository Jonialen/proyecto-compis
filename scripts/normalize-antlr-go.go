package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const workaroundComment = "// Trick to prevent compiler error if the label is not used"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: normalize-antlr-go <generated-directory>")
	}
	return filepath.WalkDir(args[0], func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".go") {
			return normalizeFile(path)
		}
		return nil
	})
}

// normalizeFile removes only ANTLR 4.13.2's annotated dead return/goto tail.
// Real recovery branches and their labels are preserved; other input is not rewritten.
func normalizeFile(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	header := bytes.SplitN(source, []byte("\n"), 2)[0]
	if !bytes.HasPrefix(header, []byte("// Code generated from ")) || !bytes.Contains(header, []byte(" by ANTLR 4.13.2. DO NOT EDIT.")) {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse generated Go: %w", err)
	}
	markers := make(map[int]*ast.CommentGroup)
	for _, group := range file.Comments {
		if len(group.List) == 1 && group.List[0].Text == workaroundComment {
			markers[fset.PositionFor(group.Pos(), false).Line] = group
		}
	}
	removed := make(map[*ast.CommentGroup]bool)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil || len(function.Body.List) < 2 {
			continue
		}
		statements := function.Body.List
		jump, ok := statements[len(statements)-1].(*ast.BranchStmt)
		if !ok {
			continue
		}
		marker := markers[fset.PositionFor(jump.Pos(), false).Line]
		if marker == nil {
			continue
		}
		result, ok := statements[len(statements)-2].(*ast.ReturnStmt)
		if !ok || len(result.Results) != 1 || jump.Tok != token.GOTO || jump.Label == nil || jump.Label.Name != "errorExit" {
			return fmt.Errorf("%s: unsupported ANTLR workaround in %s", path, function.Name.Name)
		}
		context, ok := result.Results[0].(*ast.Ident)
		if !ok || context.Name != "localctx" {
			return fmt.Errorf("%s: unsupported ANTLR return in %s", path, function.Name.Name)
		}
		labelIndex := -1
		for i, statement := range statements {
			if label, ok := statement.(*ast.LabeledStmt); ok && label.Label.Name == "errorExit" {
				labelIndex = i
			}
		}
		if labelIndex < 0 {
			return fmt.Errorf("%s: missing ANTLR recovery label in %s", path, function.Name.Name)
		}
		function.Body.List = statements[:len(statements)-1]
		used := false
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false // Labels inside nested functions have their own scope.
			}
			if branch, ok := node.(*ast.BranchStmt); ok && branch.Label != nil && branch.Label.Name == "errorExit" {
				used = true
			}
			return true
		})
		if !used {
			function.Body.List[labelIndex] = statements[labelIndex].(*ast.LabeledStmt).Stmt
		}
		removed[marker] = true
	}
	if len(removed) != len(markers) {
		return fmt.Errorf("%s: unrecognized ANTLR workaround location", path)
	}
	if len(removed) == 0 {
		return nil
	}
	comments := file.Comments[:0]
	for _, group := range file.Comments {
		if !removed[group] {
			comments = append(comments, group)
		}
	}
	file.Comments = comments
	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return fmt.Errorf("format normalized Go: %w", err)
	}
	return os.WriteFile(path, output.Bytes(), 0o644)
}
