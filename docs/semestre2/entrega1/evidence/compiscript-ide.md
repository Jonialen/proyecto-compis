# API del IDE de Compiscript

El IDE analiza el código Compiscript proporcionado por el navegador sin cambiar el flujo `/api/process` de YALex/YAPar.

## Reproducción

Inicie el servidor y luego envíe el código fuente:

```bash
go run ./cmd/ide
curl --fail-with-body -sS \
  -H 'Content-Type: application/json' \
  --data '{"source":"let x: integer = 1;"}' \
  http://localhost:8080/api/compiscript/analyze
```

La respuesta `200` es la codificación JSON exacta de `model.AnalysisReport`: `ast`, `diagnostics` y `scopes`. Los diagnósticos, incluidos los de recuperación sintáctica, son datos del reporte y aun así devuelven `200`. El código fuente vacío es válido.

El cuerpo completo de la solicitud debe ser UTF-8 válido y no superar 1 MiB (`1,048,576` bytes). El objeto JSON debe contener exactamente un `source` con valor de tipo string; las llaves duplicadas se rechazan en lugar de resolverse con la regla de que gana el último valor. Se acepta `application/json; charset=utf-8`.

## Contrato HTTP

| Estado | Significado |
|---:|---|
| `200` | Se emitió el reporte de análisis completo. |
| `400` | El cuerpo es UTF-8 inválido; el JSON está mal formado o tiene contenido sobrante al final; `source` falta, está duplicado o no es un string; o el objeto tiene campos desconocidos. |
| `405` | El método de la solicitud no es `POST`; se devuelve `Allow: POST`. |
| `413` | El cuerpo completo de la solicitud supera 1 MiB; no se analiza ningún código fuente parcial. |
| `415` | `Content-Type` no es `application/json`. |

Los errores usan `{"error":"<stable message>"}`. El editor de Compiscript del navegador renderiza el código fuente enviado, el AST visual anidado, los diagnósticos y los entornos/símbolos anidados con nodos de texto del DOM en lugar de inyección de HTML.
