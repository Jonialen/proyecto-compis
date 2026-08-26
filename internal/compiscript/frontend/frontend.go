package frontend

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend/generated"
	"genanalex/internal/compiscript/model"
	"github.com/antlr4-go/antlr/v4"
)

// Parse builds the generated parse tree and maps its statements with diagnostics.
func Parse(source []byte) (ast.Program, model.Diagnostics) {
	index := newSourceIndex(source)
	collector := newDiagnosticCollector(index)

	lexer := generated.NewCompiscriptLexer(antlr.NewInputStream(string(source)))
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(collector.listener(model.PhaseLexical))

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := generated.NewCompiscriptParser(tokens)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(collector.listener(model.PhaseSyntax))
	program := parser.Program()

	diagnostics := orderDiagnostics(collector.diagnostics)
	return recoverProgram(newStatementMapper(source).mapProgram(program), diagnostics), diagnostics
}
