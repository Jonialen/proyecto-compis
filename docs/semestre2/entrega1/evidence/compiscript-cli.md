# CLI de Compiscript

La CLI analiza un archivo fuente `.cps` y escribe como JSON el reporte de análisis completo de la fachada.

## Ejecución

Código fuente válido:

```bash
go run ./cmd/compiscript testdata/compiscript/valid/types.cps
```

Código fuente con diagnósticos:

```bash
go run ./cmd/compiscript testdata/compiscript/invalid/types.cps
```

Ambos comandos terminan con `0`. Los diagnósticos son datos del análisis, no fallos del proceso.

## Contrato de salida

`stdout` contiene únicamente el JSON de `model.AnalysisReport`, con sangría de dos espacios y un salto de línea final. El orden de los campos de nivel superior es `ast`, `diagnostics` y luego `scopes`; el orden de los campos anidados y los nombres JSON provienen directamente del modelo. Las colecciones vacías se mantienen como `[]`, nunca `null`.

`stderr` está vacío cuando se emite un reporte. Los errores de invocación, de extensión de archivo, de lectura, de codificación del reporte o de salida se escriben de forma legible para humanos en `stderr`; los mensajes legibles para humanos nunca se mezclan en `stdout`.

El UTF-8 inválido se rechaza antes del análisis con salida `1`, `stdout` vacío y un `stderr` que comienza con `compiscript: source is not valid UTF-8`.

| Salida | Significado |
|---:|---|
| `0` | Se emitió un reporte, con o sin diagnósticos. |
| `1` | El código fuente no es UTF-8 válido, no pudo leerse, o el reporte no pudo codificarse o escribirse. |
| `2` | El uso es inválido o la ruta de entrada no termina en `.cps`. |

La invocación exacta es `compiscript <file.cps>`; los argumentos extra o faltantes imprimen `usage: compiscript <file.cps>` en `stderr`.
