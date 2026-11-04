# Enunciado de la fase 3: generación de código

Transcripción de `Proyecto_3__Comiladores.pdf`, proporcionado por el equipo. El texto de cada página se conserva en bloques; los saltos y espacios provienen de la extracción del PDF. Esta fuente no reemplaza una aclaración posterior del docente ni acredita por sí sola el cumplimiento.

## Página 1

```text
Construcción de Compiladores
Octubre, 2026




                                  Proyecto Fase 3
                                Generación de Código



Descripción
    Esta actividad corresponde a la fase final en la construcción de un compilador para Com-
piscript, un lenguaje basado en un subconjunto de TypeScript: la generación de código de
máquina o lenguaje ensamblador. A partir del análisis semántico y del código intermedio (TAC),
se debe traducir esta representación intermedia a un código de bajo nivel. Para ello se utilizará
MIPS, un lenguaje ensamblador utilizado convencionalmente de forma educativa para aterrizar los
conceptos de compilación.

     Secuencias de Llamada y Retorno: Traducción de las llamadas y retornos de procedi-
     mientos, preservando el estado de la memoria mediante el manejo del stack pointer.

     Asignación de Registros: Algoritmo getReg() para asignar registros de MIPS, o uso de la
     pila, durante la traducción del código intermedio.

     Generación de Código MIPS: Código ensamblador ejecutable en un simulador de MIPS.

   Modalidad: Grupos de 3 integrantes.


Objetivos
Generales:
     Implementar un generador de código MIPS funcional a partir del código intermedio de Com-
     piscript.

     Aplicar los conceptos de secuencias de llamada, registros de activación y asignación de registros
     sobre una arquitectura real.

     Validar el código generado mediante su ejecución en un simulador de MIPS.

Especı́ficos:
     Implementar un algoritmo de generación de secuencias de llamada y de retorno de procedi-
     mientos.

     Implementar un algoritmo de asignación de registros (getReg()) o de uso de pila.

     Traducir el código de tres direcciones a código ensamblador MIPS.
```

## Página 2

```text
Construcción de Compiladores
Octubre, 2026


     Ejecutar el código generado en un simulador de MIPS y validar su correcto funcionamiento.

     Implementar una baterı́a de pruebas que valide casos exitosos y fallidos de la generación de
     código.

     Desarrollar un IDE que permita escribir y compilar código Compiscript.


Especificaciones
   El generador de código debe cumplir, como mı́nimo, con lo siguiente:

     Secuencias de Llamada y Retorno:

          • Traducción del código de tres direcciones a código MIPS capaz de realizar saltos hacia y
            desde procedimientos (funciones).
          • Preservación del estado general de la memoria al invocar y retornar de un procedimiento,
            mediante el manejo del stack pointer ($sp).
          • Paso de parámetros, almacenamiento de la dirección de retorno y del valor de retorno,
            conforme al diseño de registros de activación.

     Asignación de Registros:

          • Implementación de la función getReg() para asignar registros libres durante la traduc-
            ción del código intermedio.
          • Uso apropiado de los distintos tipos de registros de MIPS, tales como los temporales ($t)
            y los guardados ($s).
          • Opción de utilizar la pila para almacenar los valores de las operaciones cuando no existan
            registros disponibles.

     Generación de Código MIPS:

          • Generación de código ensamblador MIPS a partir del código intermedio.
          • Ejecución del código generado por medio de un simulador de MIPS, validando que se
            ejecute correctamente.

Entrada:
     Archivo fuente de Compiscript con extensión .cps.

Salida:
     Código ensamblador MIPS generado a partir del programa fuente.

     Reporte de errores encontrados durante la compilación.
```

## Página 3

```text
Construcción de Compiladores
Octubre, 2026


Funcionamiento del Programa
  1. El usuario carga o escribe un archivo .cps a través del IDE.

  2. El analizador léxico y sintáctico (ANTLR) construye el árbol sintáctico.

  3. El analizador semántico recorre el árbol validando las reglas del lenguaje y construyendo la
     tabla de sı́mbolos.

  4. Se genera el código intermedio (TAC) a partir del árbol sintáctico.

  5. El generador de código traduce el código intermedio a MIPS, aplicando las secuencias de
     llamada y retorno y la asignación de registros.

  6. El código MIPS generado se ejecuta en un simulador de MIPS para validar su funcionamiento.

  7. El usuario puede ejecutar la baterı́a de pruebas para validar casos exitosos y fallidos.


Entregables
     Un repositorio de GitHub, con commits individuales que evidencien claramente la contribución
     de cada integrante (no se permite compartir commits en conjunto).

     Baterı́a de pruebas que valide casos exitosos y fallidos de la generación de código, presente y
     funcional al momento de la evaluación.

     Documentación de la arquitectura de la implementación y documentación de cómo ejecutar
     el compilador.

     IDE funcional que permita escribir y compilar código Compiscript.
```

## Página 4

```text
Construcción de Compiladores
Octubre, 2026


Evaluación
Requisitos para Calificación
   Para poder optar a calificación, el programa debe funcionar correctamente el dı́a
de la presentación. El proyecto debe estar correctamente documentado y organizado.

 Componente                 Descripción                                                  Puntos
 Llamadas y Retornos        Algoritmo de generación de secuencias de llamada y retorno    20 pts
                            de procedimientos
 Asignación de Registros   Algoritmo de asignación de registros (getReg()) o uso de      20 pts
                            pila
 Código MIPS               Generación de código MIPS funcional y ejecutado en un        60 pts
                            simulador de MIPS, con baterı́a de pruebas
                                                                                 Total    100 pts
```
