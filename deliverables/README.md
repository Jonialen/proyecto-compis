# Deliverables

Versioned reading copies for Entrega 2 live in [entrega2](entrega2/). Start with the report, then use the run guide to reproduce the evidence. Editable sources remain in `docs/semestre2/entrega2`; do not maintain a second Markdown copy here.

| PDF | Editable source | Purpose |
|---|---|---|
| [Technical report](entrega2/report.pdf) | [report.md](../docs/semestre2/entrega2/report.md) | Requirement matrix, evidence provenance and pending human checks |
| [Architecture](entrega2/architecture.pdf) | [architecture.md](../docs/semestre2/entrega2/architecture.md) | Compiler pipeline, ownership and design decisions |
| [Run and review guide](entrega2/run-guide.pdf) | [README.md](../docs/semestre2/entrega2/README.md) | Prerequisites, CLI/IDE usage and verification entry points |
| [TAC reference](entrega2/tac-reference.pdf) | [tac-design.md](../docs/semestre2/entrega2/tac-design.md) | Instructions, translation examples and backend assumptions |

## Version policy

These four PDFs are exports, not additional implementation or browser-certification evidence. Regenerate the complete packet when its sources change and version the sources and PDFs together in the same commit. Git history retains earlier versions; this directory holds the current packet, not dated duplicates. Export tooling is temporary and is not a repository build-system feature.

Keep all four PDFs together: links between packet documents use sibling filenames and named destinations. Other references use repository branch/source-path URLs on `feat/compiscript-tac-closures-collections`; these are **not immutable commit permalinks**. They require publication of that branch and internet access. Offline checks validate the corresponding local paths, not remote availability. Source SHA-256 values can be compared with the export manifest retained during generation.

The [canonical assignment transcription](../docs/semestre2/entrega2/instructions.md) contains all four original pages, including the rubric. The human-provided `Proyecto_2__Compiladores.pdf` stays local and ignored; a fresh clone needs only the versioned transcription. It is not one of the four exported deliverables.
