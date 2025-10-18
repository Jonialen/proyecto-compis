# Entregables

Las copias de lectura versionadas de la Entrega 2 están en [entrega2](entrega2/). Empiece por el informe y luego use la guía de ejecución para reproducir la evidencia. Las fuentes editables permanecen en `docs/semestre2/entrega2`; no mantenga aquí una segunda copia en Markdown.

| PDF | Fuente editable | Propósito |
|---|---|---|
| [Informe técnico](entrega2/report.pdf) | [report.md](../docs/semestre2/entrega2/report.md) | Matriz de requisitos, procedencia de la evidencia y comprobaciones humanas pendientes |
| [Arquitectura](entrega2/architecture.pdf) | [architecture.md](../docs/semestre2/entrega2/architecture.md) | Flujo del compilador, responsabilidades y decisiones de diseño |
| [Guía de ejecución y revisión](entrega2/run-guide.pdf) | [README.md](../docs/semestre2/entrega2/README.md) | Requisitos previos, uso de la CLI y el IDE, y puntos de entrada de verificación |
| [Referencia TAC](entrega2/tac-reference.pdf) | [tac-design.md](../docs/semestre2/entrega2/tac-design.md) | Instrucciones, ejemplos de traducción y supuestos del backend |

## Política de versiones

Estos cuatro PDF son exportaciones, no evidencia adicional de implementación ni de certificación en navegador. Regenere el paquete completo cuando cambien sus fuentes y versione las fuentes y los PDF juntos en el mismo commit. El historial de Git conserva las versiones anteriores; este directorio contiene el paquete actual, no duplicados fechados. Las herramientas de exportación son temporales y no forman parte del sistema de compilación del repositorio.

Mantenga juntos los cuatro PDF: los enlaces entre documentos del paquete usan nombres de archivo hermanos y destinos con nombre. Las demás referencias usan URL de rutas de rama/fuente del repositorio en `feat/compiscript-tac-closures-collections`; **no son permalinks inmutables de commits**. Requieren que esa rama esté publicada y acceso a internet. Las comprobaciones sin conexión validan las rutas locales correspondientes, no la disponibilidad remota. Los valores SHA-256 de las fuentes pueden compararse con el manifiesto de exportación conservado durante la generación.

La [transcripción canónica del enunciado](../docs/semestre2/entrega2/instructions.md) contiene las cuatro páginas originales, incluida la rúbrica. El `Proyecto_2__Compiladores.pdf` proporcionado por el equipo se mantiene local e ignorado; una copia nueva solo necesita la transcripción versionada. No es uno de los cuatro entregables exportados.
