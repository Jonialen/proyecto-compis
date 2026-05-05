# Backlog técnico — Implementación Parte 2 (YAPar)

Este backlog traduce la planeación arquitectónica de YAPar en tareas ejecutables, ordenadas por dependencias y con criterio de terminado verificable.

---

## Vista general

### Objetivo
Implementar un generador de analizadores sintácticos en Go, integrado con la Parte 1, manteniendo separación estricta entre:

- parser de especificación `.yalp`,
- modelo de gramática,
- algoritmos LR,
- simulación,
- generación standalone.

### Orden maestro

1. Reestructuración mínima del proyecto
2. Parser de `.yalp`
3. Modelo de gramática
4. FIRST/FOLLOW
5. Items LR
6. Tabla SLR(1)
7. Simulador LR
8. Integración con lexer
9. Generador standalone
10. Documentación y pruebas finales

---

## Fase 0 — Reestructuración mínima

## Tarea 0.1 — Mover la CLI actual de YALex a `cmd/yalex`

### Objetivo
Separar el punto de entrada léxico del nuevo punto de entrada sintáctico.

### Archivos
- `main.go` → `cmd/yalex/main.go`

### Dependencias
- ninguna

### Criterio de terminado
- la CLI actual sigue funcionando igual,
- compila desde `cmd/yalex`.

### Pruebas
- `go test ./...`
- ejecución manual del flujo actual del lexer

---

## Tarea 0.2 — Crear `cmd/yapar/main.go`

### Objetivo
Preparar el nuevo orquestador para la Parte 2.

### Archivos
- `cmd/yapar/main.go`

### Dependencias
- Tarea 0.1

### Criterio de terminado
- existe una CLI mínima,
- acepta flags aunque todavía no implemente toda la lógica.

### Pruebas
- `go build ./cmd/yapar`

---

## Tarea 0.3 — Extraer contrato compartido de token

### Objetivo
Evitar acoplamiento entre parser y paquete interno del lexer.

### Archivos
- `internal/shared/token.go`
- archivos del lexer que hoy usan `Token`

### Dependencias
- Tarea 0.1

### Criterio de terminado
- el lexer y el parser dependen del mismo tipo `Token` compartido.

### Pruebas
- `go test ./...`

---

## Fase 1 — Parser de especificación `.yalp`

## Tarea 1.1 — Definir estructuras crudas de especificación

### Objetivo
Representar la salida del parser del archivo `.yalp`.

### Archivos
- `internal/yapar/parser.go`

### Dependencias
- Fase 0 completa

### Criterio de terminado
- existen tipos como `YaparSpec`, `RawProduction` e `IgnoreTokens`.

### Pruebas
- tests unitarios de construcción simple

---

## Tarea 1.2 — Eliminar comentarios y separar secciones

### Objetivo
Soportar comentarios `/* ... */` y división con `%%`.

### Archivos
- `internal/yapar/parser.go`

### Dependencias
- Tarea 1.1

### Criterio de terminado
- el parser separa correctamente encabezado léxico y bloque de producciones,
- ignora comentarios.

### Pruebas
- comentario simple
- comentario multilinea
- archivo sin `%%` debe fallar con error claro

---

## Tarea 1.3 — Parsear `%token` e `IGNORE`

### Objetivo
Interpretar correctamente terminales declarados y tokens ignorados.

### Archivos
- `internal/yapar/parser.go`

### Dependencias
- Tarea 1.2

### Criterio de terminado
- soporta múltiples tokens por línea,
- valida duplicados básicos,
- marca `IGNORE` correctamente.

### Pruebas
- una línea `%token A B C`
- múltiples líneas `%token`
- `IGNORE WS`

---

## Tarea 1.4 — Parsear producciones con `:`, `|`, `;`

### Objetivo
Construir las reglas crudas de la gramática.

### Archivos
- `internal/yapar/parser.go`

### Dependencias
- Tarea 1.3

### Criterio de terminado
- parsea alternativas,
- detecta símbolo inicial,
- soporta producciones vacías si el formato del laboratorio las permite.

### Pruebas
- producción simple
- producción con varias alternativas
- producción mal cerrada debe fallar

---

## Fase 2 — Modelo formal de gramática

## Tarea 2.1 — Definir `Symbol`, `Production` y `Grammar`

### Objetivo
Construir una representación formal, estable y reusable.

### Archivos
- `internal/yapar/grammar.go`

### Dependencias
- Fase 1 completa

### Criterio de terminado
- las producciones tienen ID,
- los símbolos distinguen terminal/no terminal.

### Pruebas
- creación de gramática desde un `YaparSpec` simple

---

## Tarea 2.2 — Clasificar terminales y no terminales

### Objetivo
Inferir correctamente el conjunto de símbolos de la gramática.

### Archivos
- `internal/yapar/grammar.go`

### Dependencias
- Tarea 2.1

### Criterio de terminado
- terminales = `%token`,
- no terminales = cabezas de producciones,
- se validan referencias inconsistentes.

### Pruebas
- gramática válida
- símbolo no declarado

---

## Tarea 2.3 — Augmentar la gramática

### Objetivo
Crear `S' -> S` como base del análisis LR.

### Archivos
- `internal/yapar/grammar.go`

### Dependencias
- Tarea 2.2

### Criterio de terminado
- la gramática aumentada existe y se identifica claramente.

### Pruebas
- validación del nuevo símbolo inicial aumentado

---

## Fase 3 — FIRST, FOLLOW y nullable

## Tarea 3.1 — Implementar sets reutilizables

### Objetivo
Evitar duplicar lógica de unión, comparación y copia.

### Archivos
- `internal/yapar/first_follow.go`

### Dependencias
- Fase 2 completa

### Criterio de terminado
- existen helpers de sets confiables.

### Pruebas
- unión
- igualdad
- inserción sin duplicados

---

## Tarea 3.2 — Calcular `nullable`

### Objetivo
Resolver qué no terminales pueden derivar epsilon.

### Archivos
- `internal/yapar/first_follow.go`

### Dependencias
- Tarea 3.1

### Criterio de terminado
- `nullable` converge por punto fijo.

### Pruebas
- gramática con epsilon directo
- gramática con epsilon transitivo

---

## Tarea 3.3 — Calcular `FIRST`

### Objetivo
Obtener los conjuntos FIRST para símbolos y secuencias.

### Archivos
- `internal/yapar/first_follow.go`

### Dependencias
- Tarea 3.2

### Criterio de terminado
- `FIRST` funciona para terminales, no terminales y secuencias.

### Pruebas
- gramática aritmética
- gramática con epsilon

---

## Tarea 3.4 — Calcular `FOLLOW`

### Objetivo
Obtener los conjuntos FOLLOW para construir la tabla SLR.

### Archivos
- `internal/yapar/first_follow.go`

### Dependencias
- Tarea 3.3

### Criterio de terminado
- incluye `$` en el símbolo inicial,
- propaga correctamente FOLLOW entre producciones.

### Pruebas
- gramática aritmética
- casos con sufijos anulables

---

## Fase 4 — Items LR y colección canónica

## Tarea 4.1 — Definir `Item` y `State`

### Objetivo
Modelar estados LR de forma estable.

### Archivos
- `internal/yapar/items.go`

### Dependencias
- Fase 3 completa

### Criterio de terminado
- los items son comparables y serializables para deduplicación.

### Pruebas
- igualdad lógica entre items

---

## Tarea 4.2 — Implementar `closure`

### Objetivo
Expandir correctamente items con punto antes de un no terminal.

### Archivos
- `internal/yapar/items.go`

### Dependencias
- Tarea 4.1

### Criterio de terminado
- `closure` agrega todos los items necesarios sin duplicados.

### Pruebas
- cierre de gramática pequeña

---

## Tarea 4.3 — Implementar `goto`

### Objetivo
Desplazar el punto sobre un símbolo y construir el siguiente conjunto.

### Archivos
- `internal/yapar/items.go`

### Dependencias
- Tarea 4.2

### Criterio de terminado
- `goto` genera estados válidos o vacío cuando corresponde.

### Pruebas
- desplazamiento sobre terminal y no terminal

---

## Tarea 4.4 — Construir colección canónica de estados

### Objetivo
Generar el autómata LR(0).

### Archivos
- `internal/yapar/items.go`

### Dependencias
- Tarea 4.3

### Criterio de terminado
- todos los estados quedan enumerados,
- no se crean duplicados equivalentes.

### Pruebas
- gramática pequeña con cantidad conocida de estados

---

## Fase 5 — Tabla SLR(1)

## Tarea 5.1 — Definir `Action`, `ActionKind` y `ParsingTable`

### Objetivo
Modelar explícitamente la tabla de parsing.

### Archivos
- `internal/yapar/table.go`

### Dependencias
- Fase 4 completa

### Criterio de terminado
- estructura lista para acciones y gotos.

### Pruebas
- construcción básica de tabla vacía

---

## Tarea 5.2 — Construir transiciones `shift` y `goto`

### Objetivo
Poblar las entradas directas del autómata.

### Archivos
- `internal/yapar/table.go`

### Dependencias
- Tarea 5.1

### Criterio de terminado
- acciones `shift` correctas,
- gotos correctos.

### Pruebas
- tabla parcial sobre gramática simple

---

## Tarea 5.3 — Construir reducciones usando `FOLLOW`

### Objetivo
Completar la lógica SLR(1).

### Archivos
- `internal/yapar/table.go`

### Dependencias
- Tarea 5.2
- Fase 3 completa

### Criterio de terminado
- cada item completo genera reducciones correctas en los símbolos de FOLLOW.

### Pruebas
- producciones reducidas esperadas por estado

---

## Tarea 5.4 — Detectar conflictos

### Objetivo
Reportar gramáticas no aptas para la tabla construida.

### Archivos
- `internal/yapar/table.go`
- `internal/yapar/errors.go`

### Dependencias
- Tarea 5.3

### Criterio de terminado
- conflictos `shift/reduce` y `reduce/reduce` producen error descriptivo.

### Pruebas
- gramática ambigua controlada

---

## Fase 6 — Simulador LR

## Tarea 6.1 — Definir errores sintácticos de ejecución

### Objetivo
Dar diagnósticos útiles en vez de errores genéricos.

### Archivos
- `internal/yapar/errors.go`

### Dependencias
- Fase 5 completa

### Criterio de terminado
- existe un `SyntaxError` con línea, token recibido y esperados.

### Pruebas
- formateo del mensaje

---

## Tarea 6.2 — Implementar stack LR y algoritmo `shift/reduce`

### Objetivo
Ejecutar el parse de tokens contra la tabla.

### Archivos
- `internal/yapar/simulator.go`

### Dependencias
- Tarea 6.1

### Criterio de terminado
- procesa `shift`, `reduce`, `accept` y `error` correctamente.

### Pruebas
- entrada aceptada
- entrada inválida
- reducción múltiple

---

## Tarea 6.3 — Filtrar tokens `IGNORE`

### Objetivo
Integrar correctamente el comportamiento sintáctico con tokens irrelevantes como whitespace.

### Archivos
- `internal/yapar/simulator.go`

### Dependencias
- Tarea 6.2

### Criterio de terminado
- el parser ignora tokens declarados en `IGNORE` antes de analizar.

### Pruebas
- `WS` presente entre tokens válidos

---

## Fase 7 — Integración con la Parte 1

## Tarea 7.1 — Adaptar el lexer para entregar tokens al parser

### Objetivo
Conectar YAPar al resultado de YALex sin acoplamiento indebido.

### Archivos
- `cmd/yapar/main.go`
- archivos necesarios en `internal/lexer`

### Dependencias
- Fase 6 completa

### Criterio de terminado
- `cmd/yapar` puede tokenizar con `.yal` y pasar los tokens al simulador.

### Pruebas
- integración `.yal + .yalp + input`

---

## Tarea 7.2 — Ejecutar modo simulador completo

### Objetivo
Permitir análisis sintáctico directo sin generación de código.

### Archivos
- `cmd/yapar/main.go`

### Dependencias
- Tarea 7.1

### Criterio de terminado
- el comando puede aceptar o rechazar una entrada real.

### Pruebas
- caso válido
- caso inválido

---

## Fase 8 — Generación standalone

## Tarea 8.1 — Diseñar el template del parser generado

### Objetivo
Definir la forma del parser standalone antes de serializar datos.

### Archivos
- `internal/generator/parser_gen.go`

### Dependencias
- Fase 7 completa

### Criterio de terminado
- existe un template claro con estructuras mínimas del runtime.

### Pruebas
- generación de archivo compilable de prueba

---

## Tarea 8.2 — Serializar gramática y tabla LR

### Objetivo
Emitir el parser completo como código Go autónomo.

### Archivos
- `internal/generator/parser_gen.go`

### Dependencias
- Tarea 8.1

### Criterio de terminado
- el parser generado compila,
- puede analizar una entrada simple.

### Pruebas
- `go build` del parser generado
- ejecución sobre entrada válida

---

## Fase 9 — End-to-end, documentación y cierre

## Tarea 9.1 — Crear suite de integración para YAPar

### Objetivo
Verificar el pipeline completo.

### Archivos
- tests de integración nuevos
- `testdata/` adicional si hace falta

### Dependencias
- Fase 8 completa

### Criterio de terminado
- hay pruebas que cubren lexer + parser + errores.

### Pruebas
- aritmética válida
- aritmética inválida
- ignorar whitespace

---

## Tarea 9.2 — Actualizar documentación técnica

### Objetivo
Dejar trazabilidad formal de la arquitectura de la Parte 2.

### Archivos
- `docs/documentacion_tecnica.md`
- `docs/parte2/planeacion_tecnica_yapar.md`

### Dependencias
- Fase 9.1

### Criterio de terminado
- la arquitectura y pipeline de YAPar están documentados.

### Pruebas
- revisión manual de consistencia

---

## Checklist de terminado global

- [ ] YALex sigue funcionando después de la reestructuración
- [ ] YAPar parsea `.yalp`
- [ ] La gramática aumentada se construye bien
- [ ] FIRST/FOLLOW son correctos
- [ ] La colección LR(0) es estable
- [ ] La tabla SLR(1) se genera o reporta conflicto correctamente
- [ ] El simulador acepta y rechaza entradas bien
- [ ] La integración con el lexer funciona
- [ ] El parser standalone se genera y compila
- [ ] La documentación quedó actualizada
