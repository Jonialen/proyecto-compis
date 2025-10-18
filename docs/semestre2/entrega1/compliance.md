# Cumplimiento actual y trabajo pendiente de la Entrega 1

Esta auditoría describe el repositorio en `789da4b`, actualizada después de que la autoridad del curso (el equipo de dos integrantes) resolviera las dos contradicciones de las fuentes que antes se registraban aquí. Por lo demás, este documento no modifica la implementación ni la gramática generada. Los requisitos proporcionados siguen siendo determinantes: [`instrucciones.md`](instrucciones.md), [`Especificaciones.md`](Especificaciones.md) y [`Compiscript.g4`](Compiscript.g4).

## Resultado

El frontend ANTLR, el recorrido con el Visitor generado, el AST del proyecto, las verificaciones semánticas, los entornos de símbolos, la CLI, el IDE en el navegador, la generación portable y la evidencia de pruebas están presentes. Ambas contradicciones de las fuentes están resueltas (ver más abajo); el equipo confirmó que tiene dos integrantes en lugar de tres, por lo que el requisito de conformación del grupo se acepta como una brecha conocida y no técnica, y no como algo que la implementación pueda cerrar.

## Matriz de requisitos

| Requisito | Estado | Evidencia actual | Pendiente |
|---|---|---|---|
| Parser ANTLR basado en la gramática proporcionada; entrada `.cps` | Cumplido | `Compiscript.g4`; `internal/compiscript/frontend/frontend.go`; `cmd/compiscript`; pruebas de gramática y de CLI | Ninguno |
| Recorrer el árbol de ANTLR mediante un Listener o un Visitor | Cumplido | El despacho del `CompiscriptVisitor` generado y las llamadas a `Accept` se usan en `mapper_statement.go` y `mapper_expression.go`; `frontend_test.go` impide volver al despacho por type-switch sobre el CST | Ninguno |
| Construir y representar visualmente un árbol sintáctico real | Cumplido | AST del proyecto en `internal/compiscript/ast`; `internal/compiscript/view.go`; salida JSON `ast`; renderizado del árbol en el navegador | Ninguno |
| Diagnósticos léxicos, sintácticos y semánticos ubicados | Cumplido | `frontend/diagnostics.go`; `semantic/semantic.go`; `AnalysisReport.diagnostics`; evidencia de UTF-8 y de recuperación en [`acceptance.md`](acceptance.md) | Ninguno |
| Verificación de tipos aritméticos, lógicos, de comparación, de asignación, de constantes y de expresiones semánticas | Cumplido | `semantic.go`; los archivos de prueba (fixtures) de operadores, tipos, nombres y aceptación cubren la polaridad válida e inválida | Ninguno |
| Listas homogéneas, listas anidadas, índices enteros y límites demostrables | Cumplido | `semantic/collections_test.go`; `collections_valid.cps`; `collections_invalid.cps` | Ninguno |
| Resolución local/global, nombres no declarados, duplicados en el mismo ámbito, acceso anidado y sombreado | Cumplido | `semantic_test.go`; `valid.cps`; `invalid.cps` | Ninguno |
| Entornos de símbolos global, de función, de clase, de bloque y de catch | Cumplido | Salida ordenada de `ScopeSnapshot`; construcción semántica de ámbitos; la CLI y el IDE exponen `scopes` | Ninguno |
| Argumentos posicionales de funciones, tipos de retorno, recursión, funciones anidadas, closures y declaraciones duplicadas | Cumplido | `functions_flow_test.go`; `functions_valid.cps`; `functions_invalid.cps`; los símbolos capturados se marcan | Ninguno |
| Condiciones booleanas en `if`, `while`, `do-while` y `for` | Cumplido | Verificaciones de condiciones en `semantic.go`; archivos de prueba de flujo | Ninguno |
| `continue` solo en ciclos; `return` solo en funciones; detección de código inalcanzable | Cumplido | Verificaciones de profundidad de transferencia y de terminación en `semantic.go`; archivos de prueba de flujo y de foreach | Ninguno |
| Clases, miembros, constructores, herencia y `this` | Cumplido | `classes_exceptions_test.go`; `classes_valid.cps`; `classes_invalid.cps` | Ninguno |
| Pruebas exitosas y fallidas para las reglas semánticas | Cumplido | Pruebas semánticas focalizadas, más el manifiesto de aceptación fijado por hash y la compuerta de evidencia de requisitos documentados en [`acceptance.md`](acceptance.md) | Ninguno |
| El IDE puede escribir/cargar y compilar Compiscript y mostrar el árbol, los diagnósticos y los entornos | Cumplido | `web/`; `POST /api/compiscript/analyze`; manejador del IDE y pruebas de aceptación; [`evidence/compiscript-ide.md`](evidence/compiscript-ide.md) | Ninguno |
| Documentación de ejecución y de arquitectura | Cumplido | El pipeline y los comandos de Compiscript en el [README del repositorio](../../../README.md); evidencia de la CLI, el IDE, la aceptación, la auditoría de la gramática y las contribuciones en este directorio | Ninguno |
| Semántica de la condición de `switch` | Cumplido (decidido) | Decisión del equipo: rige `instrucciones.md`; `switch` requiere un discriminante booleano, igual que `if`/`while`/`do-while`/`for`. El ejemplo entero `switch (x)` de `Especificaciones.md` queda sin efecto para este proyecto. `semantic.go` ahora emite `SEM_CONDITION` ante un valor de `switch` no booleano; los valores de case deben seguir siendo compatibles en tipo con ese discriminante booleano. `flow_valid.cps`/`flow_invalid.cps` y sus pruebas se actualizaron en consecuencia. | Ninguno |
| Ubicación de `break` | Cumplido (decidido) | Decisión del equipo: `break` es válido dentro de ciclos o de `switch`, en concordancia con la gramática proporcionada y con `Especificaciones.md`. No se necesitó ningún cambio de implementación; `semantic.go` ya lo permitía. | Ninguno |
| Grupos de tres con commits individuales de cada integrante | Brecha aceptada, no aplica | El equipo confirmó que tiene dos integrantes, no tres. [`evidence/contributions.md`](evidence/contributions.md) registra solo dos identidades de contribuyentes observables, lo cual es consistente con la conformación real. | Ninguno: es una restricción a nivel del programa, no un defecto; se debe informar a la autoridad del curso que el grupo tiene dos integrantes en lugar de inventar un tercer contribuyente. |
| Funcionamiento correcto durante la presentación | Aún no verificable | Las compuertas automatizadas actuales están documentadas y pasaron anteriormente. | Debe demostrarse en el entorno de evaluación el día de la presentación. |

## Brechas de auditoría cerradas

- **Visitor:** cerrada por `c8df850`; el mapeo de CST a AST ahora usa el despacho `Accept`/Visitor generado por ANTLR.
- **Generación portable:** cerrada por `92914dc`; la generación se ejecuta desde el directorio de la gramática, los encabezados confirmados no contienen ninguna ruta absoluta específica de una copia local, y dos rutas temporales distintas de copia reproducen bytes idénticos.
- **README:** cerrada por `97c394b`; el README raíz ahora explica el pipeline de Compiscript, la CLI, el IDE, los comandos de verificación y los enlaces a la evidencia.
- **Contradicción de las fuentes sobre `switch`/`break`:** cerrada tras la decisión del equipo (ver la matriz anterior); `semantic.go` y los archivos de prueba/pruebas de flujo se actualizaron para requerir un discriminante booleano en `switch`, manteniendo `break` válido dentro de `switch`.
- **Sobrescritura de métodos y herencia de constructores:** esta fila se había registrado antes como `Met` (Cumplido) por error. El analizador semántico rechazaba el ejemplo canónico de herencia de [`Especificaciones.md`](Especificaciones.md) por dos motivos: `SEM_INHERITED_MEMBER` marcaba toda redeclaración de un miembro heredado, lo que prohibía la sobrescritura de métodos, y `new` resolvía `constructor` solo en la propia clase, por lo que una subclase sin constructor propio no podía instanciarse con los parámetros heredados. `semantic.go` ahora acepta una sobrescritura cuya firma coincide con la del método heredado, sigue rechazando cualquier otro conflicto de nombres heredados y reporta `SEM_OVERRIDE` ante una sobrescritura con firma incompatible; `lookupMember` resuelve `constructor` a través de la cadena de herencia. `inheritance_valid.cps` e `inheritance_invalid.cps` cubren ambas direcciones y están fijados en el manifiesto de aceptación. Los archivos de prueba anteriores `classes_valid.cps` y `classes_invalid.cps` habían evitado ambos caminos, razón por la cual la brecha sobrevivió a la auditoría previa.

## Trabajo pendiente, en orden de prioridad

1. Volver a ejecutar las compuertas documentadas en el entorno de la presentación. Esta es una verificación operativa, no un cambio de documentación ni de implementación.

En esta auditoría no se encontró ninguna otra brecha de requisitos de la Entrega 1 visible en el repositorio. El punto del grupo de tres integrantes es una restricción del programa ya informada y no se registra aquí como trabajo pendiente.
