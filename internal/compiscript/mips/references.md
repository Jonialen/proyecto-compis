# Referencias y operaciones dinámicas en MIPS

`references.go` genera instrucciones MIPS reales para objetos, métodos y listas.
`scalars.go` selecciona operaciones y etiquetas a partir del valor efectivo cuando
TAC marca una operación como `Guarded`. No se evalúa el fuente durante la compilación.
La unidad está integrada con el núcleo y los handlers; las dos costuras detectadas por sus pruebas fueron corregidas y verificadas.

## Contrato físico

Todas las cajas empiezan alineadas a ocho bytes; conservan las etiquetas del
runtime compartido. Las referencias conservan su dirección al crear alias o
vistas tipadas.

| Caja | +0 | +4 | +8 | +12 | +16 en adelante |
|---|---|---|---|---|---|
| Objeto | 5 | descriptor de clase | vector de entornos retenidos | reservado | campos, cuatro bytes por puntero |
| Lista | 6 | descriptor del elemento de la asignación original | longitud | reservado | elementos, cuatro bytes por puntero |
| Método ligado | 7 | dirección de procedimiento | entorno retenido | receptor | — |

El vector de entornos del objeto sigue a sus campos en la misma asignación.
Se indexa por posición de clase en `Program.Classes`; solo se rellenan entradas
para la clase efectiva y sus ancestros. Cada entrada captura el entorno vigente
al ejecutar `NewObject`, no el vigente al seleccionar un método posteriormente.
Los descriptores de clase contienen el padre y el texto `<object ID>`.

Los descriptores de tipo son pares de palabras: etiqueta y descriptor hijo
(lista), descriptor nominal (objeto), o cero (escalar). Etiqueta cero significa
información desconocida de TAC, no un tipo nuevo del lenguaje.

## Reglas que conviene revisar

- Los campos heredados mantienen el índice del layout TAC. Cero en una celda de
  campo significa no inicializada; `rt_null` es un valor distinto y válido.
- `BindMethod.Callee` selecciona exactamente el inicializador indicado. Sin
  `Callee`, la selección usa `Member` y la clase efectiva, incluyendo overrides.
- La igualdad de métodos sigue siendo la tupla código/entorno/receptor del
  runtime existente. No se comparan las nuevas cajas por dirección.
- Un campo readonly solo puede inicializarse una vez desde el inicializador de
  su propietario; una escritura dinámica nunca obtiene ese privilegio.
- `checkValue` conserva todos los registros guardados, salvo el registro de
  resultado explícito. `TypeError` acepta cualquier etiqueta. La promoción de
  entero a float asigna otra caja: nunca cambia un literal ni un alias existente.
- Una vista de null no cambia su etiqueta, caja o identidad. Las comprobaciones
  nominales aceptan subclases y distinguen identidades bajo shadowing.
- Una lista conserva su contrato original incluso al escribirse mediante un
  acceso desconocido. Las vistas covariantes conocidas mantienen la política
  existente del frontend; una lectura posterior valida el elemento efectivo.
- Una comprobación de lista concreta compara el shape y la clase de su hoja.
  Si el shape original es genérico, comprueba los elementos sin copiar ni
  modificar la lista. Una lista genérica vacía conserva su identidad.
- El índice requiere etiqueta entera y un IEEE64 finito, integral y dentro de
  límites. La comprobación módulo uno precede a la conversión a entero MIPS;
  no se truncan índices enormes o fraccionarios para acceder a memoria.
- Las operaciones guardadas permiten suma de strings por longitud de bytes,
  promoción numérica y división flotante. La igualdad rechaza etiquetas
  incompatibles, exceptuando null y la combinación entero/float.

## Impresión y errores

Objetos y funciones muestran IDs de declaración estables, no direcciones de
heap. Las listas usan presupuesto de 64 valores y profundidad máxima de ocho,
con una cadena de ancestros en la pila para detectar ciclos. Los hermanos
compartidos no se confunden con ciclos. Los escalares delegan al runtime común;
los strings conservan los bytes UTF-8 y NUL embebidos.

Los errores de valores del lenguaje saltan a `rt_fault`, permitiendo que la
unidad de excepciones los entregue a handlers. Las pruebas independientes usan
fallos no capturados y verifican stdout exacto, stderr `"\n"` y salida 1.
Layouts y operandos inválidos se rechazan durante `Generate`, sin ensamblador
parcial. Hay límites explícitos de desplazamientos inmediatos para layouts y
literales de listas, además del límite de frame del núcleo.

## Verificación enfocada

```sh
go test ./internal/compiscript/mips -run '^TestMIPSReferences' -count=1 -v
go test ./internal/compiscript/mips -run '^TestMIPSDynamic' -count=1 -v
```

Las pruebas usan el compilador público y `execute`, que ejecuta el ensamblador
en MARS con los límites existentes. Comprueban herencia, entornos locales y base,
constructores, alias, shapes anidados, readonly, congelación de valores, bytes,
identidad de métodos, promociones sin mutación y fallos de tipo/índice.
La profundidad y el presupuesto de impresión tienen pruebas desde fuente;
la defensa de ciclos está implementada, pero no tiene fixture fuente aceptado.

## Costuras del núcleo verificadas

1. `TestMIPSDynamic/generic_identity_and_null` conserva el caso fuente aceptado
   `let saved; function keep(xs):integer[]{saved=xs;return xs;}`. El prólogo de
   llamadas lee todos los globals antes del cuerpo, incluso los de solo escritura,
   y producía `undefined binding`. El núcleo ahora distingue declaración activada
   mediante un sentinel privado: la dependencia no exige un valor previo, pero
   una lectura real sigue fallando si no hubo inicialización. El fixture se conserva.
2. `TestMIPSReferencesUnusedMetadata` exige rechazar una clase vacía no usada con
   inicializador inexistente. `validateClasses` ya valida ese contrato, pero las
   costuras iniciales solo la llamaban desde referencias o formales. `Generate`
   ahora la invoca incondicionalmente antes de emitir procedimientos, incluso
   cuando el programa no ejecuta instrucciones de referencia.

No se modificaron el frontend, TAC, el núcleo de llamadas, el runtime compartido
ni la unidad concurrente de excepciones. Los dos comandos enfocados y la suite completa del backend pasaron juntos tras
la corrección. La aceptación global de la implementación pasó después, incluidos 150 escenarios MIPS; ver la [evidencia](../../../docs/semestre2/mips/acceptance.md). El seguimiento posterior de T8 amplía documentación, no deja esa verificación del código pendiente.
