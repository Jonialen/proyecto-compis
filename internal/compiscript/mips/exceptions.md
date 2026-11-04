# Excepciones ejecutables en MARS

Los fallos del lenguaje llegan a `rt_raise` mediante `rt_fault`. El handler dinámico más reciente recibe una caja de excepción nueva; sin handler, se conserva la salida anterior: mensaje y salto de línea en stdout, código de salida 1.

## Verificación rápida

```sh
go test ./internal/compiscript/mips -run '^TestMIPSExceptions' -count=1 -v
```

Las pruebas usan fuente pública → compilador → `mips.Generate` → MARS real. Comparan stdout, stderr y salida exactos. No interpretan TAC ni evalúan fuente durante la generación. Requieren MARS instalado; `-short` omite la ejecución externa.

## Estado y desenrollado

| Elemento | Representación |
|---|---|
| Handler | Registro heap de 56 bytes: anterior, destino, `$sp`, `$fp`, `$ra`, `$s0`…`$s7`, excepción pendiente anterior |
| Cadena dinámica | `rt_handler`, compartida entre llamadas y recursión; no pertenece al entorno de una clausura |
| Excepción | Caja alineada de 16 bytes, tag 9, puntero de texto en 4, longitud en bytes en 8; identidad por dirección |
| Captura | `CatchException` guarda la caja con `g.store`; los bindings retenidos por clausuras sobreviven al frame |

`PushHandler` resuelve el destino con `g.jumpLabel`. `rt_raise` retira el handler seleccionado **antes** de saltar al catch y restaura su stack, frame y registros guardados. Descarta así vectores de argumentos incompletos, stacks auxiliares y llamadas intermedias. Los valores TAC viven en sus homes canónicos, no en registros temporales.

Restaurar `$s0` y `$s1` recupera las entradas visibles al instalar el handler. No se revierten mutaciones de celdas existentes ni se liberan entornos heap: una clausura escapada desde una llamada abandonada conserva sus capturas y efectos previos al fallo.

`PopHandler` respeta exclusivamente los opcodes de limpieza TAC, incluidos return, break y continue; no reconstruye ámbitos léxicos. Recupera el estado pendiente anterior sin cambiar entornos ni homes. `CatchException` consume la excepción pendiente después de almacenarla.

## Errores de infraestructura y límites

La validación de flujo comprueba profundidad y estado pendiente en saltos, destinos catch y salidas alcanzables. Rechaza underflow y joins incompatibles antes de entregar assembly. Los guardas defensivos de ejecución para pop/catch inválidos terminan directamente, sin convertirse en fallos capturables.

El texto recibido por `rt_raise` debe ser ASCII NUL-terminado estable, según el ABI de `rt_fault`; la caja retiene ese puntero. Los handlers y las cajas usan el allocator monotónico existente, sin recolector.

Las pruebas independientes cubren división/resto, identidad, catches anidados, recursión, homes del caller, capturas retenidas, fallos con stack auxiliar y limpieza normal/escapante. Los fallos específicos de objetos, listas y verificaciones dinámicas corresponden a la unidad de referencias; su integración con este handler requiere verificación conjunta por el padre. No se modifica el frontend ni el TAC estático.
