# Backend MIPS de Compiscript

`Generate(*tac.Program) (string, error)` traduce el IR estructurado, no su representación textual. Devuelve ensamblador MARS o un error sin ensamblador parcial. No ejecuta el programa durante la compilación.

## Verificar el backend

Desde la raíz, con MARS instalado según [`tools/mars/README.md`](../../../tools/mars/README.md):

```sh
go test ./internal/compiscript/mips -count=1 -v
```

Las pruebas generan ensamblador desde `.cps` mediante el compilador existente y lo ejecutan realmente en MARS, con límites de tiempo e instrucciones; comparan stdout, stderr y código de salida. `-short` omite las ejecuciones externas, no la validación pura del IR.

## Representación física

- Cada valor es un puntero a una caja alineada de al menos 16 bytes. Etiqueta a offset 0; booleanos y punteros de texto a offset 4; números IEEE-754 binary64 a offset 8. Las cadenas guardan además su longitud UTF-8 a offset 8: no se truncan en NUL y se escriben por syscall 15. Esto conserva los valores `float64` del oráculo TAC existente, incluso para enteros mayores de 32 bits.
- Las etiquetas distinguen entero, flotante, booleano, cadena, objeto, lista, función, null y excepción. Las ranuras temporales contienen punteros y pueden cambiar de tipo.
- Los valores tienen una ubicación canónica en memoria. Los resultados se guardan inmediatamente; las llamadas y las transferencias de control no pueden perder temporales vivos.
- Los argumentos se evalúan antes de llamar y se pasan en un vector de pila del caller, sin límite especial de cuatro argumentos. El callee los copia a sus propias ranuras. El resultado vuelve por `$v0`; `$ra` conserva la continuación y `$fp` separa activaciones recursivas.
- El encabezado de activación reserva 48 bytes y preserva `$ra`, `$fp` y `$s0`–`$s7`; las ranuras comienzan a offset 48. Los operandos sobreviven a helpers en registros `$s`; `getReg` presta scratch `$t` por instrucción.
- Cada entorno retenido guarda ID de ámbito, padre léxico, entrada anterior de la activación y celdas de bindings. `$s0` apunta al entorno capturado y `$s1` a las instancias propias más recientes; una entrada de bloque siempre crea una instancia nueva. Las cajas de funciones guardan código (offset 4), entorno (8) y receptor (12).
- El heap es monotónico durante una ejecución: permite conservar referencias después de liberar una activación. No hay recolección de basura en este corte.
- El resto numérico usa reducción binaria sobre doubles, sin truncar el cociente a un entero de 32 bits.
- La salida numérica usa impresión entera cuando el valor es integral y cabe en 32 bits; el resto usa la impresión double de MARS. No se promete formato decimal idéntico a Go para valores extremos.

## Cobertura y verificación

Este corte cubre valores escalares, operaciones, cadenas, impresión, control, funciones directas, parámetros, retornos y recursión. Las pruebas incluyen Fibonacci, recursión mutua, aislamiento de locales, widening y diez argumentos con llamadas anidadas. También cubre entornos retenidos, capturas mutables, reentrada de bloques, closures y llamadas indirectas con validación de firma; las funciones se comparan dinámicamente por código, entorno y receptor. El runtime agrega [objetos, herencia, métodos ligados, listas y valores dinámicos](references.md), junto con [excepciones y desenrollado](exceptions.md). Ambos bloques y sus regresiones pasaron en MARS. Un sentinel privado distingue un binding declarado sin valor de uno aún no declarado; leer cualquiera sin inicialización falla. La suite también rechaza metadata inválida aunque no se use. La integración CLI/API/IDE y la aceptación global pasaron; la [guía](../../../docs/semestre2/mips/README.md) y la [evidencia final](../../../docs/semestre2/mips/acceptance.md) describen comandos, 150 escenarios, límites y controles humanos pendientes. El [registro de trabajo](../../../odd/tasks/compiscript-mips.md) conserva decisiones y commits.
