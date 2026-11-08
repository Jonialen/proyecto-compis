# Paquete de lectura de la fase MIPS

Empezá por `report.pdf`; después usá `run-guide.pdf` para reproducir la demo. Los Markdown de `docs/semestre2/mips/` son las fuentes editables: no se mantiene una segunda copia de sus contenidos aquí.

| PDF | Fuente | Contenido |
|---|---|---|
| [Informe técnico](report.pdf) | [report.md](../../docs/semestre2/mips/report.md) | Objetivo, requisitos, decisiones, resultados y límites |
| [Arquitectura](architecture.pdf) | [architecture.md](../../docs/semestre2/mips/architecture.md) | Pipeline, memoria, frames, registros, entornos y handlers |
| [Guía de ejecución](run-guide.pdf) | [README.md](../../docs/semestre2/mips/README.md) | Preparación, CLI, API, IDE, pruebas y presentación |
| [Referencia TAC a MIPS](mips-reference.pdf) | [mips-reference.md](../../docs/semestre2/mips/mips-reference.md) | Instrucciones, contratos y ejemplos de traducción |

## Versionado y alcance

Mantené juntos los cuatro PDF: sus enlaces entre documentos usan nombres hermanos. Las referencias a fuentes, ejemplos y evidencia que no se exportan apuntan a rutas del repositorio en la rama `feat/compiscript-mips`; no son permalinks de commits ni se afirma que la rama esté publicada. La lectura del contenido del paquete no necesita esos enlaces externos; consultar el código remoto sí requiere que exista la rama publicada y acceso a GitHub.

El [manifiesto de exportación](manifest.json) registra hashes de cada fuente/PDF, páginas, herramientas y opciones. Actualizá las cuatro exportaciones y el manifiesto cuando cambie cualquiera de sus fuentes. Generar un PDF no agrega evidencia de funcionamiento ni sustituye las comprobaciones de navegador, entorno del evaluador o contribuciones individuales.

La exportación usa Pandoc y XeLaTeX disponibles localmente; no son dependencias del compilador. Las herramientas auxiliares de maquetación son temporales, como en la entrega anterior. Al reexportar, respetá el mapa de nombres del manifiesto, remapeá los enlaces `.md` entre estos cuatro documentos a sus `.pdf` hermanos y comprobá texto, páginas y enlaces; una conversión directa sin ese remapeo conservaría enlaces a los Markdown.

Este paquete no modifica ni reemplaza los PDF de `deliverables/entrega2/`. La [transcripción del enunciado](../../docs/semestre2/mips/instructions.md) y la [evidencia detallada](../../docs/semestre2/mips/acceptance.md) siguen siendo fuentes adicionales versionadas.
