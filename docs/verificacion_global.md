# Verificación global del repositorio

**Estado: parcial.** La suite completa pasó; `go vet ./...` sigue fallando.
El commit de código probado fue `a533d50123e10630f1987c1b0bb3c87770d9cd48`.
Este informe no convierte el resultado de vet en una aprobación.

## Resultados observados

Las órdenes se ejecutaron una vez cada una, secuencialmente y en una copia
aislada del repositorio, con las dependencias ya disponibles y sin descargas.

| Orden | Código de salida | Resultado |
|---|---|---|
| `go test ./internal/compiscript/frontend -run '^(TestGeneratedArtifactsArePortableAndReproducible|TestGrammarCompatibilityAndFloatLiterals)$' -count=1` | 0 | Ambos tests antes bloqueados por ANTLR pasaron |
| `go test ./... -count=1` | 0 | 15 paquetes OK, 0 fallidos y 4 paquetes sin tests |
| `go vet ./...` | 1 | 46 diagnósticos `unreachable code` en el parser generado |

No se usó `-short`. Go y Node estaban disponibles; la salida no verbosa no
permite afirmar un conteo individual de tests omitidos.

## Entorno y ANTLR

- Go: `go1.27.1 darwin/arm64`; `go.mod` declara `go 1.26.1`.
- Java: OpenJDK `17.0.20.1`; Node: `24.21.0`.
- Herramienta ANTLR: `4.13.2`; runtime Go declarado: `v4.13.1`.
- JAR restaurado desde <https://www.antlr.org/download/antlr-4.13.2-complete.jar>.
- SHA-256 verificado contra `tools/antlr/antlr-4.13.2-complete.jar.sha256`:
  `eae2dfa119a64327444672aff63e9ec35a20180dc5b8090b7a6ab85125df4d76`.
- El JAR es caché local ignorada por Git; no se agrega el binario al repositorio.

## Hallazgo pendiente de vet

Los 46 diagnósticos se ubican en
`internal/compiscript/frontend/generated/compiscript_parser.go`, desde
`589:2` hasta `8662:2`. Por ejemplo, las líneas 588–589 contienen:

```go
return localctx
goto errorExit // Trick to prevent compiler error if the label is not used
```

El código generado conserva esos bytes en la revisión probada. No se editaron
sus saltos manualmente, no se cambiaron versiones y no se desactivó ningún
analizador para presentar un resultado verde. No se determinó si el diagnóstico
varía con otra versión de Go.

## Preservación y alcance

Los hashes de 243 archivos rastreados permanecieron iguales antes y después de
las pruebas, tanto en el checkout original como en la copia aislada. La
gramática y los ocho artefactos generados no tuvieron diferencias tras la
regeneración. Los archivos sin seguimiento preexistentes se preservaron.

El siguiente trabajo requiere una decisión explícita sobre la compatibilidad
del generador ANTLR y el análisis estático de Go. Este cierre restaura el
requisito de entorno y registra la evidencia; no modifica la gramática ni el
código del compilador para resolver vet.
