# Backlog técnico detallado — Fase 1: arquitectura común multi-método

Esta fase prepara el backend para soportar de forma limpia y comparable:

- **LL(1)**
- **SLR(1)**
- **LALR(1)**

El objetivo NO es implementar todavía esos métodos completos, sino construir la base común correcta para que no queden acoplados entre sí.

---

## Objetivo de la fase

Extraer y formalizar una arquitectura común para múltiples estrategias de parsing, separando:

- gramática,
- construcción de estructuras de análisis,
- ejecución del parser,
- errores,
- visualización,
- resultados comparables.

---

## Resultado esperado al terminar la fase

Al terminar esta fase, el proyecto debe tener:

1. contratos comunes para métodos de parsing,
2. tipos compartidos para tablas, visualización y resultados,
3. separación clara entre backend común y algoritmos concretos,
4. estructura lista para empezar **LL(1)** sin contaminar lo ya hecho para **SLR(1)**.

---

## Fase 1.1 — modelar el concepto de “método de parsing”

## Tarea 1.1.1 — Definir tipo de método

### Objetivo
Representar explícitamente el algoritmo usado.

### Archivos sugeridos
- `internal/yapar/method.go`

### Propuesta

```go
type Method string

const (
	MethodLL1  Method = "ll1"
	MethodSLR1 Method = "slr1"
	MethodLALR Method = "lalr"
)
```

### Criterio de terminado
- existe un tipo común para identificar el método,
- puede reutilizarse desde CLI, backend y GUI.

### Pruebas
- parseo/validación básica del método si aplica

---

## Tarea 1.1.2 — Definir metadatos comunes del análisis

### Objetivo
Tener una estructura base que describa el artefacto construido para cada método.

### Archivos sugeridos
- `internal/yapar/analysis.go`

### Propuesta

```go
type Analysis struct {
	Method  Method
	Grammar *Grammar
}
```

### Criterio de terminado
- existe un contenedor mínimo que permita crecer sin acoplarse a SLR.

---

## Fase 1.2 — separar resultados de construcción y ejecución

## Tarea 1.2.1 — Refinar `ParseResult`

### Objetivo
Convertir `ParseResult` en un contrato común a todos los métodos.

### Archivos sugeridos
- `internal/yapar/simulator.go`
- o nuevo `internal/yapar/result.go`

### Propuesta

```go
type ParseResult struct {
	Accepted bool
	Method   Method
	Steps    int
}
```

### Criterio de terminado
- `ParseResult` ya no queda implícitamente ligado a SLR.

### Nota
No hace falta meter árbol ni traza completa todavía.

---

## Tarea 1.2.2 — Separar errores de construcción y errores de ejecución

### Objetivo
Diferenciar mejor:

- error de especificación,
- error de gramática,
- conflicto de tabla,
- error de parse runtime.

### Archivos sugeridos
- `internal/yapar/errors.go`

### Criterio de terminado
- los tipos de error quedan listos para reutilización por LL(1), SLR y LALR.

### Pruebas
- unitarias sobre `Error()` si aplica

---

## Fase 1.3 — definir contratos comunes para tablas

## Tarea 1.3.1 — Separar tabla abstracta de tablas concretas

### Objetivo
Evitar que todo el proyecto asuma solo `ACTION/GOTO` estilo LR.

### Archivos sugeridos
- `internal/yapar/table_model.go`

### Propuesta

```go
type TableKind string

const (
	TableKindLL1  TableKind = "ll1"
	TableKindLR   TableKind = "lr"
)

type TableView struct {
	Method Method
	Kind   TableKind
	Rows   []TableRow
}

type TableRow struct {
	State string
	Cells map[string]string
}
```

### Criterio de terminado
- existe una representación común exportable para mostrar tablas en CLI/GUI.

---

## Tarea 1.3.2 — Adaptar SLR actual a una vista común

### Objetivo
Permitir que la tabla SLR ya implementada se pueda exportar como `TableView`.

### Archivos sugeridos
- `internal/yapar/table.go`
- `internal/yapar/table_model.go`

### Criterio de terminado
- se puede convertir `ParsingTable` a una vista genérica.

### Pruebas
- verificar serialización/vista de tabla SLR

---

## Fase 1.4 — definir contratos comunes para visualización

## Tarea 1.4.1 — Definir modelo visual del autómata

### Objetivo
Preparar una estructura común para visualizar LR(0) y luego LALR.

### Archivos sugeridos
- `internal/yapar/visualization.go`

### Propuesta

```go
type GraphView struct {
	Name  string
	Nodes []GraphNode
	Edges []GraphEdge
}

type GraphNode struct {
	ID    string
	Label string
}

type GraphEdge struct {
	From  string
	To    string
	Label string
}
```

### Criterio de terminado
- existe un formato consumible por CLI, JSON o GUI.

---

## Tarea 1.4.2 — Exportar LR(0) actual a `GraphView`

### Objetivo
No esperar hasta la GUI para poder mostrar el autómata.

### Archivos sugeridos
- `internal/yapar/items.go`
- `internal/yapar/visualization.go`

### Criterio de terminado
- la colección canónica LR(0) se exporta a una representación visual común.

### Pruebas
- número de nodos/aristas esperados en gramáticas pequeñas

---

## Fase 1.5 — extraer interfaz común del runtime

## Tarea 1.5.1 — Definir interfaz común de parser ejecutable

### Objetivo
Preparar un contrato uniforme para LL(1), SLR(1) y LALR.

### Archivos sugeridos
- `internal/yapar/runtime.go`

### Propuesta

```go
type ExecutableParser interface {
	Method() Method
	Parse(tokens []shared.Token) (*ParseResult, error)
	TableView() *TableView
}
```

### Criterio de terminado
- existe un contrato común de ejecución.

---

## Tarea 1.5.2 — Adaptar SLR actual al contrato común

### Objetivo
Usar el backend ya implementado como primera instancia del diseño nuevo.

### Archivos sugeridos
- `internal/yapar/simulator.go`
- nuevo wrapper tipo `internal/yapar/slr_runtime.go`

### Criterio de terminado
- el parser SLR actual puede exponerse como `ExecutableParser`.

### Pruebas
- una prueba que construya el runtime y ejecute parse vía la interfaz

---

## Fase 1.6 — preparar capa de construcción por método

## Tarea 1.6.1 — Crear builder común por método

### Objetivo
Centralizar la construcción del backend según el método elegido.

### Archivos sugeridos
- `internal/yapar/builder.go`

### Propuesta

```go
type BuildOptions struct {
	Method Method
}

func BuildParser(spec *YaparSpec, opts BuildOptions) (ExecutableParser, error)
```

### Criterio de terminado
- existe un punto único de entrada para construir el parser por método.

### Nota
En esta fase puede soportar solo `slr1` y devolver “not implemented” para `ll1`/`lalr`.

---

## Tarea 1.6.2 — Definir comportamiento de métodos no implementados

### Objetivo
Evitar caos cuando la CLI o GUI pidan LL(1)/LALR antes de que existan.

### Archivos sugeridos
- `internal/yapar/errors.go`
- `internal/yapar/builder.go`

### Criterio de terminado
- el sistema devuelve errores explícitos y útiles para métodos pendientes.

---

## Fase 1.7 — extender CLI mínimamente sin romper alcance

## Tarea 1.7.1 — agregar selector de método a `cmd/yapar`

### Objetivo
Dejar preparada la CLI para el backend comparativo.

### Archivos sugeridos
- `cmd/yapar/main.go`

### Criterio de terminado
- existe `-method`,
- `slr1` funciona,
- `ll1` y `lalr` responden “not implemented” de forma clara.

### Pruebas
- test de parsing de flags
- test de error por método no implementado

---

## Fase 1.8 — pruebas de arquitectura

## Tarea 1.8.1 — pruebas del builder común

### Objetivo
Verificar que la nueva arquitectura no rompa el camino SLR.

### Archivos sugeridos
- `internal/yapar/builder_test.go`

### Criterio de terminado
- construir parser `slr1` funciona,
- pedir `ll1` o `lalr` devuelve error controlado.

---

## Tarea 1.8.2 — pruebas de exportación común

### Objetivo
Verificar que las vistas comunes sirvan para futuras visualizaciones.

### Archivos sugeridos
- tests de `TableView`
- tests de `GraphView`

### Criterio de terminado
- SLR exporta tabla y LR(0) exporta grafo sin acoplar la GUI.

---

## Checklist de terminado de la fase

- [ ] Existe `Method` común
- [ ] Existe `ExecutableParser`
- [ ] Existe `BuildParser(...)`
- [ ] El camino SLR actual sigue funcionando sobre la nueva arquitectura
- [ ] Existen `TableView` y `GraphView`
- [ ] `cmd/yapar` acepta selector de método
- [ ] LL(1) y LALR reportan “not implemented” claramente
- [ ] `go test ./...` sigue pasando

---

## Siguiente fase después de esta

Cuando esta fase termine, el siguiente paso correcto es:

## **Fase 2 — implementación de LL(1)**

Con esta arquitectura, LL(1) podrá entrar como un método nuevo sin contaminar el runtime LR ya construido.
