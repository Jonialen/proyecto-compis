# Backlog extendido — Parte 2 real

## Avance contrastado con la implementación

La [guía de demos](guia_demos_yapar.md) contiene los archivos y comandos
reproducibles. La matriz `TestDemoCLI` ejecuta 16 escenarios sobre el binario
real y comprueba códigos de salida y mensajes. No se incluye guion de
presentación, por solicitud del usuario.

| Fase | Estado observado | Evidencia principal |
|---|---|---|
| 1. Arquitectura común | Implementada; clasificación de errores reforzada | `method.go`, `errors.go`, pruebas de gramática y contratos en `internal/yapar/` |
| 2. LL1 | Implementada; rechazo de recursión y conflictos probado | `ll1_backend_test.go`, casos `required`, `nullable` e `indirect` |
| 3. LALR | Implementada; diferencia frente a SLR probada | `lalr_backend_test.go`, demo `assignment` |
| 4. Comparación | Implementada; resultados y conflictos disponibles | `comparison_test.go`, CLI e IDE |
| 5. Visualización | Tablas y SVG LR(0) comprobados en flujos de demo | `visualizer_test.go`, validación Chrome sobre `d0cff74` |
| 6. CLI | Métodos y casos de demo ejecutados | `cmd/yapar/demo_test.go` y pruebas existentes de CLI |
| 7. IDE | Flujos de demo comprobados en Chrome real | Carga, edición, descarga, errores, vacío y ejecuciones consecutivas sobre `d0cff74` |
| 8. Cierre reproducible | Demos, guía y README preparados; video no verificado | [Guía](guia_demos_yapar.md); guion excluido del alcance |

Estos estados se limitan a las pruebas y escenarios indicados; no certifican
todas las gramáticas ni eliminan las limitaciones de standalone y conflictos.

Las listas originales de brechas son históricas: LL(1), LALR, comparación,
visualización e IDE ya tienen implementaciones. No deben reconstruirse por
aparecer en la sección «Falta»; se deben contrastar sus criterios con pruebas.

En la Fase 2, la política de recursión izquierda es **rechazar con error claro**,
sin transformación automática. El detector cubre recursión directa, indirecta y
prefijos nullable; conserva la recursión derecha válida. La regresión pública
se ejecuta con:

```sh
go test ./internal/yapar -run '^TestBuildParserLL1(RejectsLeftRecursiveCycles|AllowsRecursionAfterConsumedTerminal)$' -count=1
```

La Fase 2 también cuenta con una regresión del conflicto FIRST/FOLLOW de
`s : opt B ; opt : B | ;`, que debe rechazarse como `LL1ConflictError`.

Para el criterio comparativo de la Fase 3, la gramática
`s : L EQUAL R | R ; L : STAR R | ID ; R : L ;` produce conflicto
shift/reduce con SLR, pero LALR acepta referencias y asignaciones válidas y
rechaza entradas incompletas o asignaciones encadenadas:

```sh
go test ./internal/yapar -run '^(TestBuildParserLL1RejectsFirstFollowConflict|TestBuildParserLALRAcceptsGrammarThatSLRRejects)$' -count=1
```

Estas pruebas caracterizan comportamiento existente; no requieren cambiar la
política actual de conflictos LALR. Este avance no certifica todas las fases
ni los entregables de presentación.

Este backlog extiende la planeación anterior para alinearla con el enunciado oficial de la Parte 2, que exige no solo SLR(1), sino también **LL(1)**, **LALR**, visualización de estructuras y una **interfaz gráfica tipo IDE**.

---

## Resumen original de brecha (histórico)

### Ya implementado

- YALex funcional
- Parser `.yalp`
- Gramática formal
- `nullable`, `FIRST`, `FOLLOW`
- Autómata **LR(0)**
- Tabla **SLR(1)**
- Simulador LR
- Integración lexer → parser
- CLI `cmd/yalex` y `cmd/yapar`
- Generador standalone del parser

### Brecha registrada al crear este backlog (ya implementada)

- **LL(1)**
- **LALR(1)**
- Comparación entre métodos
- Visualización de **LR(0)**
- Visualización de tablas **LL(1)**, **SLR(1)** y **LALR**
- **IDE gráfica**

---

## Fase 1 — estabilizar arquitectura para múltiples métodos

### Objetivo
Crear una capa común para soportar múltiples estrategias de parsing sin acoplar todo a SLR(1).

### Tareas
1. Definir interfaz común para métodos de parsing.
2. Unificar tipos base para resultados, errores y visualización.
3. Separar mejor la capa de gramática de la capa de algoritmo.
4. Definir contratos exportables para tablas y autómatas.

### Entregable
Backend listo para enchufar:
- LL(1)
- SLR(1)
- LALR(1)

---

## Fase 2 — implementar LL(1)

### Objetivo
Agregar un parser predictivo funcional.

### Tareas
1. Implementar construcción de tabla **LL(1)**.
2. Detectar conflictos LL(1).
3. Definir política frente a recursión izquierda y factorización:
   - transformar, o
   - rechazar con error claro.
4. Implementar simulador LL(1).
5. Agregar pruebas unitarias y de integración.

### Entregable
- Tabla LL(1)
- Parser LL(1)
- Tests

---

## Fase 3 — implementar LALR(1)

### Objetivo
Agregar un parser LALR funcional.

### Tareas
1. Elegir estrategia:
   - LR(1) canónico + merge, o
   - construcción directa.
2. Implementar items con lookahead.
3. Construir colección LR(1).
4. Merge de estados compatibles.
5. Construir tabla LALR.
6. Detectar conflictos.
7. Reutilizar runtime LR actual donde sea posible.
8. Agregar pruebas sobre gramáticas donde SLR falle y LALR pase.

### Entregable
- Tabla LALR
- Parser LALR
- Tests comparativos

---

## Fase 4 — capa comparativa de métodos

### Objetivo
Poder correr el mismo input con:
- LL(1)
- SLR(1)
- LALR(1)

### Tareas
1. Crear selector de método.
2. Unificar formato de resultado.
3. Unificar formato de error.
4. Permitir comparar aceptación, conflictos y tablas.

### Entregable
Backend comparativo reutilizable por CLI y GUI.

---

## Fase 5 — visualización

### Objetivo
Mostrar estructuras requeridas por el enunciado.

### Tareas
1. Exportar autómata **LR(0)**.
2. Exportar tabla **LL(1)**.
3. Exportar tabla **SLR(1)**.
4. Exportar tabla **LALR**.
5. Definir formato consumible por interfaz.

### Entregable
Visualización de autómata y tablas.

---

## Fase 6 — CLI extendida

### Objetivo
Poder probar todo desde línea de comandos antes de construir la GUI.

### Tareas
1. Extender `cmd/yapar` con selector de método.
2. Soporte para imprimir/exportar tablas.
3. Soporte para exportar visualización LR(0).
4. Ejecutar parsing con el método elegido.

### Entregable
CLI comparativa y utilizable para validación.

---

## Fase 7 — IDE gráfica

### Objetivo
Construir la interfaz tipo IDE requerida por el curso.

### Funcionalidades mínimas
1. Cargar archivos `.yal`, `.yalp` e input.
2. Editor integrado.
3. Guardado de cambios.
4. Ejecución del análisis.
5. Selector de método:
   - LL(1)
   - SLR(1)
   - LALR
6. Visualización de:
   - LR(0)
   - tablas
   - resultados
   - errores

### Entregable
IDE funcional para demo y evaluación.

---

## Fase 8 — pulido de entrega

### Objetivo
Preparar la presentación y validación final.

### Tareas
1. Casos de demo válidos e inválidos: disponibles en `testdata/yapar/demos/`, con pruebas CLI.
2. Guion técnico de presentación: excluido por solicitud del usuario; no se genera.
3. Revisión de README y documentación: guía reproducible enlazada desde el README.
4. Revisión de video/demo: flujos de IDE validados; video no creado ni verificado.

### Entregable
Proyecto listo para evaluación.

---

## Orden original de implementación

1. Fase 1 — arquitectura común
2. Fase 2 — LL(1)
3. Fase 3 — LALR
4. Fase 4 — backend comparativo
5. Fase 5 — visualización
6. Fase 6 — CLI extendida
7. Fase 7 — IDE
8. Fase 8 — pulido final

---

## Riesgos fuertes

1. **LALR** es el bloque algorítmico más delicado.
2. **LL(1)** puede requerir transformación o rechazo explícito de gramáticas.
3. **GUI** puede consumir demasiado tiempo si se empieza antes de cerrar el backend.
4. Mezclar visualización con lógica de parsing demasiado temprano puede ensuciar el diseño.

---

## Recomendación inmediata

Ejecutar la matriz de la [guía de demos](guia_demos_yapar.md) antes de cambiar
algoritmos o la IDE. No reconstruir LL1, LALR ni los contratos comunes por
aparecer en las listas históricas. Las extensiones de standalone, una política
general de precedencia y la verificación de entregables externos requieren
trabajo separado; no forman parte de este cierre de demos.
