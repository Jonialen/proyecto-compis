# Demos reproducibles de YAPar

Los archivos de `testdata/yapar/demos/` permiten comprobar aceptación, rechazo,
entrada vacía, recursión izquierda y diferencias entre SLR y LALR. Esta guía
es una referencia de ejecución, no un guion de presentación.

## Comprobación rápida

Desde la raíz del repositorio, con Go compatible con `go.mod`:

```sh
go test ./cmd/yapar -run '^TestDemoCLI$' -count=1 -v
```

La prueba compila el ejecutable real y comprueba 16 escenarios, incluyendo
código de salida, mensajes de aceptación en stdout y diagnósticos en stderr.
No modifica los fixtures. `go test -short` omite esta prueba de integración.

## Archivos y resultados esperados

Las rutas siguientes son relativas a `testdata/yapar/demos/`. Para las
gramáticas de enteros se reutiliza `testdata/yapar/arithmetic.yal`.

| Gramática | Entrada | Método | Resultado |
|---|---|---|---|
| `required.yalp` | `valid.txt` (`7`) | LL1, SLR, LALR | Aceptada; salida 0 |
| `required.yalp` | `invalid.txt` (`+`) | LL1, SLR, LALR | Error sintáctico; salida 1 |
| `required.yalp` | `empty.txt` (archivo de cero bytes) | LL1, SLR, LALR | Error sintáctico; salida 1 |
| `nullable.yalp` | `empty.txt` | LL1, SLR, LALR | Aceptada; salida 0 |
| `indirect.yalp` | Sin entrada | LL1 | Error de recursión izquierda; salida 1 |
| `assignment.yalp` | Sin entrada | SLR | Conflicto shift/reduce; salida 1 |
| `assignment.yalp` | `assignment_valid.txt` (`x = * y`) | LALR | Aceptada; salida 0 |
| `assignment.yalp` | `assignment_invalid.txt` (`x =`) | LALR | Error sintáctico; salida 1 |

Las asignaciones usan su propio lexer `assignment.yal`. El caso `+` es un
token reconocido por el lexer aritmético, pero inválido para la gramática:
demuestra un rechazo sintáctico, no un error léxico.

## Ejecución desde CLI

Estas variables solo abrevían rutas; ejecutar los comandos desde la raíz:

```sh
D=testdata/yapar/demos
L=testdata/yapar/arithmetic.yal

# Repetir con -method slr o -method lalr para los otros métodos.
go run ./cmd/yapar -yalp "$D/required.yalp" -yal "$L" -src "$D/valid.txt" -method ll1
go run ./cmd/yapar -yalp "$D/required.yalp" -yal "$L" -src "$D/invalid.txt" -method ll1
go run ./cmd/yapar -yalp "$D/required.yalp" -yal "$L" -src "$D/empty.txt" -method ll1
go run ./cmd/yapar -yalp "$D/nullable.yalp" -yal "$L" -src "$D/empty.txt" -method ll1

# Diagnóstico de construcción: no necesita lexer ni entrada.
go run ./cmd/yapar -yalp "$D/indirect.yalp" -method ll1
go run ./cmd/yapar -yalp "$D/assignment.yalp" -method slr

# La misma gramática sí permite construir LALR y analizar una asignación.
go run ./cmd/yapar -yalp "$D/assignment.yalp" -yal "$D/assignment.yal" -src "$D/assignment_valid.txt" -method lalr
go run ./cmd/yapar -yalp "$D/assignment.yalp" -yal "$D/assignment.yal" -src "$D/assignment_invalid.txt" -method lalr
```

Los rechazos son resultados esperados: `go run` también imprime `exit status 1`.
No ejecutar todos los ejemplos con `&&`, porque el primer rechazo detendría
la secuencia. La matriz automática ejecuta y verifica cada caso por separado.

Para inspeccionar una tabla o el autómata LR(0), sin analizar entrada:

```sh
go run ./cmd/yapar -yalp "$D/required.yalp" -method ll1 -format text
go run ./cmd/yapar -yalp "$D/required.yalp" -method slr -format dot
```

## Reproducción en la IDE

1. Ejecutar `go run ./cmd/ide` y abrir `http://localhost:8080`.
2. Cargar el lexer correspondiente con **Load .yal**, la gramática con
   **Load .yalp** y la entrada con **Load Input**.
3. Seleccionar los métodos de la tabla anterior y presionar **Run**.
4. Revisar **Summary**, **Tokens**, las tablas por método y **Automaton**.
   El autómata visible corresponde a LR(0), no a la colección LR(1) canónica.
5. Para el caso vacío, cargar `empty.txt` conservando el lexer: debe producir
   una decisión de aceptación o rechazo. Sin lexer, el flujo es solo de tablas.
6. Cambiar de entrada inválida a válida y ejecutar de nuevo: deben desaparecer
   los errores previos y actualizarse las tablas y tarjetas.
7. Editar un archivo y usar **Save**: descarga el contenido del editor activo;
   no sobrescribe directamente el archivo del repositorio.

Para el ejemplo de asignaciones, seleccionar solo SLR y LALR. SLR debe
mostrar su conflicto; LALR debe aceptar la entrada válida. Si se omite el
lexer, **Table Ready** solo acredita construcción, no análisis de entrada.

## Parser standalone LALR

La generación LALR utiliza la misma plantilla LR que SLR, pero serializa la
tabla LALR. Los archivos `assignment_valid.tokens.json` e
`assignment_invalid.tokens.json` representan las dos entradas de asignación
de esta guía; no es necesario preparar tokens a mano para repetir el ejemplo.

Desde la raíz, crear la salida en un directorio temporal:

```sh
D=testdata/yapar/demos
WORK=$(mktemp -d)
go run ./cmd/yapar -yalp "$D/assignment.yalp" -method lalr -out "$WORK/parser.go"
go build -o "$WORK/parser" "$WORK/parser.go"
"$WORK/parser" -tokens "$D/assignment_valid.tokens.json"
"$WORK/parser" -tokens "$D/assignment_invalid.tokens.json"
```

La entrada válida termina con código 0 e imprime `Input accepted`. La inválida
termina con código 1 y escribe un error sintáctico de línea 1 en stderr. Ese
rechazo es esperado. El binario es autónomo: no importa paquetes del proyecto
ni lee la gramática o el lexer al ejecutarse. Sigue consumiendo tokens JSON;
no se agrega un adaptador automático para la salida del lexer standalone.

La prueba reproduce generación, compilación, aceptación y rechazo, comprueba
que dos generaciones producen los mismos bytes y mantiene el rechazo de
gramáticas LL1 recursivas por la izquierda y de `-compare` con `-out`:

```sh
go test ./cmd/yapar -run '^(TestRunGeneratesStandaloneParsers|TestStandaloneRejectionsPreserveDestination|TestRunRejectsLeftRecursiveLL1StandaloneGeneration|TestRunCompareRejectsStandaloneGeneration)$' -count=1
```

## Parser standalone LL1

LL1 genera un runtime predictivo separado. La gramática `predictive.yalp`
reconoce un entero seguido de cero o más sumas; usa una producción vacía y
recursión derecha. El JSON válido incluye un token `WS` para comprobar que
el parser aplica `IGNORE WS`.

```sh
D=testdata/yapar/demos
WORK=$(mktemp -d)
go run ./cmd/yapar -yalp "$D/predictive.yalp" -method ll1 -out "$WORK/parser.go"
go build -o "$WORK/parser" "$WORK/parser.go"
"$WORK/parser" -tokens "$D/predictive_valid.tokens.json"
"$WORK/parser" -tokens "$D/predictive_invalid.tokens.json"
```

La entrada válida termina con código 0. La inválida (`7 +`) termina con código
1 y un diagnóstico que espera `INT` en la línea 1. Los tokens extra y el JSON
malformado también se rechazan. Para aceptar una entrada completamente vacía,
generar con `nullable.yalp` y usar `[]` como archivo de tokens JSON.

La CLI reutiliza `BuildLL1Table` para construir la tabla que serializa; no
convierte acciones LR en predicciones ni transforma automáticamente gramáticas
incompatibles. Los binarios LR anteriores conservan su plantilla y formato.

La regresión de equivalencia compara stdout, stderr y código de salida con el
parser LL1 integrado para prefijos nullable, tokens ignorados y diagnósticos
de EOF, líneas sin ubicación y tokens sobrantes:

```sh
go test ./internal/generator -run '^TestLL1StandaloneDiagnosticParity$' -count=1
```

## Evidencia y límites

- `cmd/yapar/demo_test.go` conserva la matriz CLI de estos archivos.
- `cmd/ide/yapar_process_test.go` verifica el contrato HTTP, incluidos vacío y
  espacios omitidos; `web/app.test.cjs` verifica estados y encabezados de error.
- El flujo equivalente de carga/edición/descarga, tablas, SVG, errores y
  ejecuciones consecutivas fue validado en Chrome headless sobre `d0cff74`.
  Esa ejecución usó fixtures temporales equivalentes, no estos archivos nuevos.
- No se certifican todas las gramáticas, tamaños de pantalla ni una entrega
  académica completa. No se creó ni revisó un video.
- `lr0` y `lr1` siguen como placeholders ejecutables; la generación standalone
  desde CLI admite los tres métodos implementados: LL1, SLR y LALR.

La suite global de Compiscript requiere además el JAR de ANTLR documentado en
su configuración. Las pruebas de estas demos YAPar no lo necesitan.
