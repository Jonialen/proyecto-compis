# Guía de ejecución de aceptación de la Entrega 1

Ejecute desde la raíz del repositorio. La compuerta determinante construye la CLI actual e invoca el manejador actual del IDE mediante `httptest`; nunca depende del puerto `8080` ni de otro proceso.

Para el estado de los requisitos, las contradicciones de las fuentes y el trabajo pendiente, consulte el [cumplimiento actual](compliance.md).

## Ruta rápida

Prerrequisitos: la versión de Go indicada en `go.mod`; Java solo para la identidad del generador; Node para la sintaxis del navegador.

```bash
go test ./cmd/ide -run 'Test(CompiscriptAcceptanceCorpus|CompiscriptRecoveryPreservesValidSiblings|AcceptanceManifestRejectsUnsafeInput|AcceptanceManifestRejectsSymlinkEscape|AcceptanceRequirementEvidence)' -count=1
go test ./internal/compiscript/... ./cmd/compiscript ./cmd/ide -count=1
go test ./internal/compiscript/frontend -run 'Test(LocalizedRecovery|OfficialExamples)' -count=1
```

Esperado: todos terminan con `0`. `TestOfficialExamples` cubre dos programas combinados representativos del parser; no se afirma que los fragmentos dependientes del documento fuente sean programas independientes.

## Matriz de evidencia

`testdata/compiscript/acceptance/manifest.tsv` está ordenado, confinado a rutas y fijado por hash para detectar desviaciones. Los hashes no prueban la corrección: las pruebas verifican diagnósticos/ubicaciones exactos, la recuperación, la forma del árbol/arreglo, los entornos ordenados y la igualdad entre fachada/CLI/IDE.

| Regla | Evidencia `.cps` válida / inválida ubicada |
|---|---|
| Parseo, árbol visual, recuperación, UTF-8 | `types-valid`; `recoverable:2`; `multibyte:2`; `lexical-invalid:1`; `constant-initializer-invalid:1` |
| Nombres, ámbitos anidados, sombreado, duplicados | `names-valid:1`; `names-invalid:1` |
| Aritmética, lógica, comparación, asignación, constantes | `operators-valid:1`; `operators-invalid:1`; `names-valid:1`; `names-invalid:1` |
| Null y centinela de recuperación | `null-valid:1`; `null-invalid:1`; `names-invalid:1` |
| Funciones, recursión, closures, argumentos, retornos | `functions-valid:1-11`; `functions-invalid:1-9` |
| Condiciones, `for`, `foreach`, switch, transferencia, código muerto | `flow-valid:1-11`; `flow-invalid:1-23`; `foreach-valid:1`; `foreach-invalid:1` |
| Listas homogéneas/irregulares, índices, límites demostrables | `collections-valid:1-17`; `collections-invalid:1-25` |
| Clases, constructores, `this`, herencia | `classes-valid:1-17`; `classes-invalid:1-18` |
| Enlace de excepción solo en catch | `catch-valid:1`; `catch-invalid:1-2` |
| IDE (15), semántica/árbol (60), entornos (25) | Cada fila atraviesa todos los consumidores; las pruebas focalizadas demuestran la polaridad |

Los operadores cubren `%`, `||`, `!` unario, `!=`, `>` y `>=`; null cubre destinos válidos string/list/class y destinos inválidos integer/boolean. `ErrorType` es un centinela de recuperación cuya polaridad válida es la supresión de diagnósticos dependientes, no un tipo del código fuente. La falta de inicialización de una constante es evidencia inválida exclusiva del parser, porque la gramática requiere `=` y una expresión.

## Compuertas completa, de candidato y del generador

```bash
go test ./internal/yalex ./internal/yapar ./cmd/yapar -count=1
go test . -run 'TestIntegration_' -count=1
go test ./... -count=1
go build ./...
go test . -run TestDependencyTreeHasNoExternalOrGUIPackages -count=1
node --check web/app.js

tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
export GIT_INDEX_FILE="$tmp/index"
git read-tree HEAD; git add -A; git diff --cached --check
test -z "$(git diff --cached --name-only -z --diff-filter=ACM -- '*.go' | xargs -0 -r gofmt -l)"
bad=$(git diff --cached --name-only | grep -Ev '^(cmd/ide/acceptance_test.go|docs/semestre2/entrega1/acceptance.md|openspec/changes/compiscript-semantic-analysis/tasks.md|testdata/compiscript/acceptance/.*)$' || true)
test -z "$bad"
rm -rf "$tmp"; unset GIT_INDEX_FILE; trap - EXIT

tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
./scripts/generate-compiscript.sh --output "$tmp/generated"
diff -qr internal/compiscript/frontend/generated "$tmp/generated"
identity=$(cd "$tmp/generated" && for f in Compiscript.interp Compiscript.tokens CompiscriptLexer.interp CompiscriptLexer.tokens compiscript_base_visitor.go compiscript_lexer.go compiscript_parser.go compiscript_visitor.go; do sha256sum "$f"; done | sha256sum | cut -d' ' -f1)
test "$identity" = 79d82b69a89a6ef6b5d21a1eaf9ec9b5699b9db3048c2deaa244bbdc344fe2bb
```

El índice temporal cubre los archivos no rastreados sin modificar el índice real. Su lista de rutas permitidas, junto con la verificación de dependencias, excluye paquetes de producción, imports, dependencias y mecanismos de extensión. El orden explícito del generador es independiente de la configuración regional y nunca reemplaza la salida confirmada en el repositorio.

## Solución de problemas y reversión

- Discrepancia de hash: inspeccione la salida; actualice solo ante un cambio de contrato aprobado.
- Discrepancia del generador: verifique el checksum fijado del JAR y el conjunto exacto de artefactos.
- Reversión: elimine la prueba de aceptación, la guía de ejecución y el corpus, y restaure solo la línea 4.1 del registro de tareas; ningún comportamiento de producción pertenece a esta unidad.
