# Backlog extendido — Parte 2 real

Este backlog extiende la planeación anterior para alinearla con el enunciado oficial de la Parte 2, que exige no solo SLR(1), sino también **LL(1)**, **LALR**, visualización de estructuras y una **interfaz gráfica tipo IDE**.

---

## Resumen de brecha

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

### Falta para cumplir el enunciado oficial

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
1. Casos de demo válidos e inválidos.
2. Guion técnico de presentación.
3. Revisión de README y documentación.
4. Revisión de video/demo.

### Entregable
Proyecto listo para evaluación.

---

## Prioridad recomendada

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

El siguiente paso correcto es comenzar por:

- **Fase 1 — arquitectura común**
- **Fase 2 — LL(1)**

Esto ordena el backend comparativo antes de meterse a LALR o a la interfaz.
