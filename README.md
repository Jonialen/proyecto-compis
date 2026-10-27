# Compiscript: compilador e IDE

Proyecto académico de Construcción de Compiladores. Compiscript transforma archivos `.cps` mediante un frontend ANTLR y análisis semántico en código intermedio de tres direcciones (TAC), con diagnósticos ubicados y símbolos extendidos. La CLI y el IDE web Furlantran comparten el mismo núcleo. **Compilar genera código; no ejecuta el programa fuente ni el TAC.**

[Repositorio en GitHub](https://github.com/Jonialen/proyecto-compis) · [Guía de Entrega 2](docs/semestre2/entrega2/README.md) · [Entregables PDF](deliverables/README.md)

## Empezar desde la raíz

Se necesita **Go 1.26.1 o posterior**, según [go.mod](go.mod), y acceso a las dependencias del módulo (o una caché existente). El módulo conserva el nombre `genanalex`. No se necesita Java para estos comandos si se usan los archivos ANTLR ya generados.

```sh
go run ./cmd/compiscript-tac -h
go run ./cmd/compiscript-tac testdata/compiscript/tac/numeric.cps
go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/acceptance.cps
```

El primer ejemplo genera TAC de una expresión numérica; el segundo devuelve el reporte JSON con `tac`, `diagnostics`, `scopes` y, cuando corresponde, `codegen` en unidades `abstract-cell`. Las opciones preceden al archivo. Los valores de ejecución descritos en los informes pertenecen al oráculo acotado de pruebas, no a la salida de la CLI.

| Interfaz | Comando desde la raíz | Resultado |
|---|---|---|
| Compilación | `go run ./cmd/compiscript-tac -format=json testdata/compiscript/tac/acceptance.cps` | TAC, diagnósticos y símbolos/entornos de generación |
| Análisis anterior | `go run ./cmd/compiscript testdata/compiscript/tac/acceptance.cps` | AST, diagnósticos y ámbitos; sin TAC ni `codegen` |
| IDE local | `go run ./cmd/ide` | Interfaz web en `http://localhost:8080` |

Para el IDE, ejecutar el servidor desde la raíz porque sirve la carpeta relativa `web`. Seleccionar Compiscript, cargar o editar un `.cps` y elegir **Compile .cps**; **Analyze .cps (AST)** conserva el análisis anterior. El servidor escucha en `:8080`, no exclusivamente en loopback: usarlo en un entorno local confiable y detenerlo con Ctrl+C. Los detalles de paneles y errores están en el [manual del IDE](docs/semestre2/entrega2/ide.md).

## Entrega 3: intérprete SQL

El mismo repositorio incluye un intérprete SQL con almacenamiento en archivos (lexer, parser, AST, análisis semántico, `CREATE`/`DROP`/`INSERT`, `SELECT` con `JOIN`, agregaciones y subconsultas, `UPDATE`/`DELETE` e índices en archivos). Detalles, formato de almacenamiento y límites en la [guía de Entrega 3](docs/semestre2/entrega3/README.md).

```sh
go run ./cmd/sql -db /tmp/mibd testdata/sql/ok/enunciado.sql
go run ./cmd/ide   # y abrir http://localhost:8080/sql.html
```

## Informes, instrucciones y entregables

Los Markdown son las fuentes editables. Los cuatro PDF versionados están separados en `deliverables/entrega2/`; conservarlos juntos para los enlaces internos del paquete.

| Documento | Leer o editar | PDF para entregar |
|---|---|---|
| Informe técnico y matriz de requisitos/evidencia | [Informe](docs/semestre2/entrega2/report.md) | [report.pdf](deliverables/entrega2/report.pdf) |
| Arquitectura y decisiones | [Arquitectura](docs/semestre2/entrega2/architecture.md) | [architecture.pdf](deliverables/entrega2/architecture.pdf) |
| Ejecución y revisión | [Guía](docs/semestre2/entrega2/README.md) | [run-guide.pdf](deliverables/entrega2/run-guide.pdf) |
| Lenguaje intermedio, ejemplos y supuestos | [Referencia TAC](docs/semestre2/entrega2/tac-design.md) | [tac-reference.pdf](deliverables/entrega2/tac-reference.pdf) |

- [Instrucciones originales completas](docs/semestre2/entrega2/instructions.md): transcripción en español de las cuatro páginas, flujo de seis pasos y rúbrica de 25/65/10 puntos. El PDF docente original se conserva localmente, ignorado; no es necesario para una copia nueva del repositorio.
- [Batería de aceptación](docs/semestre2/entrega2/acceptance.md): comandos, resultados esperados y límites de la evidencia.
- [Índice y política de versiones de entregables](deliverables/README.md): regenerar los PDF al cambiar sus fuentes y versionarlos juntos. Los enlaces externos de los PDF apuntan a rutas de una rama, no a permalinks inmutables.

## Pruebas reproducibles

La [política de reproducción](docs/semestre2/entrega2/acceptance.md#pruebas-reproducibles) describe las dependencias y la separación entre pruebas focalizadas y verificaciones completas. Ejemplos desde la raíz:

```sh
go test ./internal/compiscript -run '^TestComplexSourceFilesCLI$' -count=1 -v
go test ./internal/compiscript ./cmd/ide -run '^Test(ValuesEffects|DynamicExceptions|IndependentT9|IndependentT10|MixedAcceptanceAPI)' -count=1
node --test web/*.test.cjs
```

La primera prueba cubre siete fuentes complejas mediante la CLI construida; las otras ejercitan contratos del compilador/API y conexiones VM/DOM de la interfaz. Node.js es necesario para las pruebas web integradas; no requieren `npm install`. Las pruebas no sustituyen una revisión visual del navegador.

Para reproducir sin red se requieren módulos Go previamente almacenados, `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local` y HOME/TMPDIR/GOCACHE externos. La suite completa puede regenerar archivos ANTLR ignorados: ejecutarla en una copia aislada. La regeneración requiere Java y el jar ANTLR 4.13.2 en `tools/antlr/`; el runtime Go es v4.13.1. Una caché existente no demuestra que una instalación nueva funcione sin red. Véase también la [verificación global](docs/verificacion_global.md).

## Mapa del repositorio

| Ruta | Contenido |
|---|---|
| `cmd/compiscript-tac`, `cmd/compiscript`, `cmd/ide` | CLI de compilación, CLI de análisis y servidor del IDE |
| `internal/compiscript/` | Frontend, análisis semántico, TAC y pruebas del núcleo |
| `testdata/compiscript/` | Fuentes de ejemplo, aceptación y casos complejos |
| `web/` | Interfaz Furlantran y pruebas de conexiones |
| `docs/semestre2/entrega2/` | Fuentes de documentación actual e instrucciones canónicas |
| `deliverables/entrega2/` | Cuatro exportaciones PDF versionadas |
| `docs/semestre2/entrega1/`, `docs/semestre1/` | Entregas anteriores e historia técnica |
| `cmd/yalex`, `cmd/yapar`, otros paquetes de `internal/` | Herramientas históricas de lexer/parser y soporte compartido |
| `scripts/`, `tools/` | Utilidades y herramientas de generación existentes |
| `odd/`, `openspec/` | Contexto y registros del desarrollo |

Las especificaciones de texto `lex` y `yapar`, el ejemplo `input.txt`, el material `ins`, `lr0.dot`, `lr0.png` y las imágenes históricas se conservan como entradas o evidencia, no como basura de compilación.

## Alcance y comprobaciones pendientes

El informe documenta traducción de construcciones válidas del AST, funciones/closures, objetos/listas, valores dinámicos y try/catch, junto con las guardas estáticas y los fallos de ejecución previstos por el IR. Esto no certifica todos los programas, terminación ni una calificación. Los desplazamientos usan celdas abstractas, no una ABI física; no se entrega un ensamblador ni un intérprete de producto.

Antes de presentar: ejecutar la batería en el entorno de evaluación, revisar carga/edición/guardado y paneles en un navegador real, confirmar disponibilidad de los recursos CDN CodeMirror/Viz.js, acceso a GitHub y contribuciones individuales reales. La disponibilidad sin conexión del navegador no está garantizada. Consultar los [controles humanos pendientes](docs/semestre2/entrega2/report.md#brechas-de-requisitos-pendientes-y-controles-de-presentación).

## Herramientas históricas: YALex y YAPar

El trabajo del primer semestre sigue disponible; no es el punto de entrada principal de Entrega 2. YALex procesa especificaciones léxicas y YAPar construye/compara analizadores. Para comandos, métodos y generación standalone, consultar la [documentación técnica del semestre 1](docs/semestre1/documentacion_tecnica.md) y las [demostraciones reproducibles de YAPar](docs/semestre1/parte2/guia_demos_yapar.md), en lugar de duplicar aquí sus manuales.

La Entrega 1 de Compiscript conserva su [matriz histórica](docs/semestre2/entrega1/compliance.md), [aceptación](docs/semestre2/entrega1/acceptance.md) y [evidencia de contribuciones](docs/semestre2/entrega1/evidence/contributions.md). Estos registros no reemplazan el informe actual ni se reinterpretan como evidencia nueva.
