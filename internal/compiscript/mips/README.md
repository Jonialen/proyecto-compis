# Backend MIPS de Compiscript — en desarrollo

`Generate(*tac.Program) (string, error)` traduce el IR estructurado, no su representación textual. Devuelve ensamblador MARS o un error sin ensamblador parcial. No ejecuta el programa durante la compilación.

## Verificar el corte escalar

Desde la raíz, con MARS instalado según [`tools/mars/README.md`](../../../tools/mars/README.md):

```sh
go test ./internal/compiscript/mips -count=1 -v
```

Las pruebas generan ensamblador desde `.cps` mediante el compilador existente y lo ejecutan realmente en MARS, con límites de tiempo e instrucciones; comparan stdout, stderr y código de salida. `-short` omite las ejecuciones externas, no la validación pura del IR.

## Representación física

- Cada valor es un puntero a una caja alineada de 16 bytes. Etiqueta a offset 0; booleanos y punteros de texto a offset 4; números IEEE-754 binary64 a offset 8. Las cadenas guardan además su longitud UTF-8 a offset 8: no se truncan en NUL y se escriben por syscall 15. Esto conserva los valores `float64` del oráculo TAC existente, incluso para enteros mayores de 32 bits.
- Las etiquetas distinguen entero, flotante, booleano, cadena, objeto, lista, función, null y excepción. Las ranuras temporales contienen punteros y pueden cambiar de tipo.
- Los valores tienen una ubicación canónica en memoria. Los resultados se guardan inmediatamente; las llamadas y las transferencias de control no pueden perder temporales vivos.
- El encabezado de activación reserva 48 bytes y preserva `$ra`, `$fp` y `$s0`–`$s7`; las ranuras comienzan a offset 48. Los operandos sobreviven a helpers en registros `$s`; `getReg` presta scratch `$t` por instrucción.
- El heap es monotónico durante una ejecución: permite conservar referencias después de liberar una activación. No hay recolección de basura en este corte.
- El resto numérico usa reducción binaria sobre doubles, sin truncar el cociente a un entero de 32 bits.
- La salida numérica usa impresión entera cuando el valor es integral y cabe en 32 bits; el resto usa la impresión double de MARS. No se promete formato decimal idéntico a Go para valores extremos.

## Estado, no entrega final

Este corte cubre valores escalares, operaciones, cadenas, impresión y control. Las funciones, entornos, objetos, listas, guardas dinámicas y excepciones se implementan en los siguientes bloques de [`odd/tasks/compiscript-mips.md`](../../../odd/tasks/compiscript-mips.md). Una operación aún no implementada se rechaza; no se omite ni se finge su ejecución. Todavía no está conectado a CLI/API/IDE.
