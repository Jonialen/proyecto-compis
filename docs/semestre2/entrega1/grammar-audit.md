# Auditoría de la gramática de Compiscript

## Compatibilidad

La gramática original sigue siendo determinante. `forStatement` preserva la pertenencia
del inicializador y del separador de la condición, mientras que la evidencia oficial
`if (n < 60) continue;` justifica aceptar una sentencia existente en cada rama de `if`
sin restringir la sintaxis.

## Delta de flotantes

Las reglas oficiales de tipos justifican la palabra reservada `float`. `FloatLiteral` precede
a `IntegerLiteral`, requiere dígitos a ambos lados, preserva `42` y `floatValue`, y rechaza
`.5`, `5.`, `1..2` y `1.2.3` con diagnósticos.

## Evidencia de generación

El generador verifica ANTLR 4.13.2, prepara una salida segura respecto de argv, da formato
al código Go y reemplaza solo tras un resultado exitoso; las pruebas focalizadas cubren rutas
con espacios y cada fallo especificado. La generación se ejecuta desde el directorio de la
gramática con el nombre base de la gramática, de modo que los encabezados generados
contienen `Compiscript.g4` en lugar de una ruta absoluta de la copia local. Una prueba
focalizada regenera desde dos rutas temporales distintas de copia local y compara ambas
salidas con los artefactos confirmados, byte por byte.

## Evidencia de recuperación y de aceptación

`recovery_test.go` demuestra que una declaración mal formada se convierte en un único
`BadStmt` localizado mientras sus declaraciones hermanas válidas siguen mapeadas, y que un
operando de print mal formado se convierte en `BadExpr` sin reemplazar las sentencias print
adyacentes. También parsea dos ejemplos oficiales representativos de `Especificaciones.md`
sin diagnósticos ni sentencias erróneas de nivel superior. El manifiesto canónico fija este
alcance exacto y ordenado de ocho artefactos:
`(cd "$tree" && for f in Compiscript.interp Compiscript.tokens CompiscriptLexer.interp CompiscriptLexer.tokens compiscript_base_visitor.go compiscript_lexer.go compiscript_parser.go compiscript_visitor.go; do sha256sum "$f"; done) | sha256sum | awk '{print $1}'`.
Después de que la regeneración portable cambiara solo los encabezados generados específicos
de la copia local, la identidad confirmada actual es
`79d82b69a89a6ef6b5d21a1eaf9ec9b5699b9db3048c2deaa244bbdc344fe2bb`, que coincide
con la compuerta de `acceptance.md`.
