// Package lexer proporciona el motor de simulacion utilizado para identificar
// tokens en un archivo fuente usando uno o mas Automatas Finitos Deterministas (DFAs).
//
// Este archivo implementa el algoritmo de tokenizacion "Maximal Munch" (coincidencia
// mas larga), que ejecuta multiples DFAs en paralelo sobre la entrada para encontrar
// el lexema mas largo posible en cada posicion. La desambiguacion entre DFAs que
// coinciden en la misma longitud se resuelve por prioridad (orden de definicion
// en el archivo .yal).
package lexer

import (
	"fmt"

	"genanalex/internal/dfa"
)

// Token representa una unidad lexica individual identificada por el analizador lexico.
//
// Campos:
//   - Type:   tipo del token (por ejemplo, "ID", "NUMBER", "OPERATOR").
//     Corresponde al nombre de la regla que lo reconocio.
//   - Lexeme: la cadena exacta del codigo fuente que coincidio con el patron.
//   - Line:   numero de linea (base 1) donde se encontro el token.
type Token struct {
	Type   string
	Lexeme string
	Line   int
}

// DFAEntry agrupa un DFA con los metadatos necesarios para el proceso de simulacion.
// Cada regla del archivo .yal se convierte en un DFAEntry independiente.
//
// Campos:
//   - DFA:       puntero al DFA minimizado usado para el reconocimiento de patrones.
//   - TokenName: nombre/etiqueta que se asigna al token si este DFA logra una coincidencia.
//     El valor especial "skip" indica que el lexema debe descartarse (ej: espacios en blanco).
//   - Priority:  numero de prioridad para desambiguacion. Numeros menores indican mayor prioridad.
//     La prioridad se asigna segun el orden de aparicion de la regla en el archivo .yal.
type DFAEntry struct {
	DFA       *dfa.DFA
	TokenName string
	Priority  int
}

// Tokenize procesa un archivo fuente y convierte su contenido en un flujo de tokens.
// Implementa el principio "Maximal Munch" (coincidencia maxima): siempre intenta
// encontrar la coincidencia mas larga posible desde la posicion actual.
//
// Si multiples DFAs coinciden con el mismo lexema mas largo, se selecciona el que
// tiene mayor prioridad (menor valor numerico de Priority), determinada por el orden
// de definicion en el archivo .yal.
//
// Parametros:
//   - dfas: slice de DFAEntry con todos los automatas a ejecutar en paralelo.
//     Cada uno corresponde a una regla del archivo .yal.
//   - src:  archivo fuente previamente leido y normalizado con ReadSource.
//
// Retorna:
//   - []Token:  slice de tokens identificados en orden de aparicion.
//   - []string: slice de mensajes de error para caracteres no reconocidos.
//
// Algoritmo general:
//  1. Convertir el contenido a slice de runas (soporte Unicode completo).
//  2. Para cada posicion en la entrada:
//     a. Inicializar todos los DFAs en su estado inicial.
//     b. Avanzar caracter por caracter, transicionando todos los DFAs activos.
//     c. Registrar la posicion mas lejana donde algun DFA llega a estado de aceptacion.
//     d. Cuando ningun DFA puede avanzar, usar la ultima posicion de aceptacion registrada.
//     e. Si no hubo aceptacion, reportar error lexico y avanzar un caracter.
//  3. Desambiguar por prioridad cuando multiples DFAs aceptan el mismo lexema.
//  4. Descartar tokens marcados como "skip" (espacios en blanco, comentarios, etc.).
func Tokenize(dfas []DFAEntry, src *SourceFile) ([]Token, []string) {
	var tokens []Token
	var errors []string

	// Convertir a runas para manejar correctamente caracteres Unicode multi-byte.
	runes := []rune(src.Content)
	i := 0    // Puntero a la posicion actual en el arreglo de runas.
	line := 1 // Contador de linea actual (base 1) para reportes de error.

	// Ciclo principal: procesar toda la entrada caracter por caracter.
	for i < len(runes) {
		// dfaStatus rastrea el estado actual de simulacion de un DFA individual.
		// Se crea como tipo local porque solo se usa dentro de este ciclo.
		type dfaStatus struct {
			entry   DFAEntry // El DFAEntry original con el automata y metadatos.
			current int      // Estado actual del DFA durante la simulacion.
			active  bool     // Indica si el DFA aun puede consumir caracteres.
		}

		// Inicializar todos los DFAs en sus respectivos estados iniciales.
		// Cada DFA comienza activo y en su estado Start.
		states := make([]dfaStatus, len(dfas))
		for k, entry := range dfas {
			states[k] = dfaStatus{
				entry:   entry,
				current: entry.DFA.Start,
				active:  true,
			}
		}

		lastOKPos := -1              // Posicion final de la coincidencia mas larga encontrada hasta ahora. -1 significa "sin coincidencia".
		var lastOKMatches []DFAEntry // DFAs que aceptaron en la posicion lastOKPos.

		// Ciclo de lookahead (Maximal Munch): avanza caracter por caracter
		// intentando extender la coincidencia lo mas posible.
		j := i
		for j < len(runes) {
			c := runes[j]
			anyActive := false

			// Intentar transicionar cada DFA activo con el simbolo 'c'.
			// Los DFAs que no tienen transicion para 'c' se desactivan.
			for k := range states {
				if !states[k].active {
					continue
				}
				s := &states[k]
				transitions := s.entry.DFA.Transitions[s.current]
				if nextState, ok := transitions[c]; ok {
					// Transicion exitosa: el DFA avanza al siguiente estado.
					s.current = nextState
					anyActive = true
				} else {
					// Sin transicion disponible: este DFA ya no puede coincidir mas.
					s.active = false
				}
			}

			// Si ningun DFA puede consumir el caracter actual, terminamos el lookahead.
			// La coincidencia mas larga ya fue registrada (si existio).
			if !anyActive {
				break
			}

			j++

			// Verificar cuales de los DFAs que acaban de transicionar estan
			// en un estado de aceptacion. Esto nos indica una coincidencia parcial valida.
			var currentAccepting []DFAEntry
			for k := range states {
				if states[k].active && states[k].entry.DFA.Accepting[states[k].current] {
					currentAccepting = append(currentAccepting, states[k].entry)
				}
			}

			// Si hay al menos un DFA aceptando, registrar esta posicion como la
			// coincidencia valida mas reciente. Esto implementa el "Maximal Munch":
			// seguimos avanzando para buscar una coincidencia aun mas larga,
			// pero recordamos esta posicion por si no encontramos nada mejor.
			if len(currentAccepting) > 0 {
				lastOKPos = j
				lastOKMatches = currentAccepting
			}
		}

		// Si no se encontro ninguna coincidencia (ningun DFA llego a estado de aceptacion),
		// el caracter actual es un error lexico. Se reporta y se avanza un caracter
		// para intentar recuperarse y seguir tokenizando el resto de la entrada.
		if lastOKPos == -1 {
			errors = append(errors, fmt.Sprintf("line %d: unrecognized character %q", line, runes[i]))
			if runes[i] == '\n' {
				line++
			}
			i++
			continue
		}

		// === Desambiguacion por prioridad ===
		// Extraer el lexema correspondiente a la coincidencia mas larga.
		lexeme := string(runes[i:lastOKPos])

		// Seleccionar el DFA con la prioridad mas alta (valor numerico mas bajo)
		// entre todos los que aceptaron en la misma posicion.
		// Esto garantiza que las reglas definidas primero en el archivo .yal
		// tengan precedencia sobre las definidas despues.
		bestMatch := lastOKMatches[0]
		for _, m := range lastOKMatches[1:] {
			if m.Priority < bestMatch.Priority {
				bestMatch = m
			}
		}

		// "skip" es una accion especial que descarta el lexema reconocido.
		// Se usa tipicamente para espacios en blanco, tabuladores y saltos de linea
		// que no deben generar tokens en la salida.
		if bestMatch.TokenName != "skip" {
			tokens = append(tokens, Token{
				Type:   bestMatch.TokenName,
				Lexeme: lexeme,
				Line:   line,
			})
		}

		// Actualizar el contador de linea recorriendo el lexema reconocido.
		// Esto es importante para tokens que pueden abarcar multiples lineas
		// (por ejemplo, cadenas multilinea o comentarios de bloque).
		for _, r := range lexeme {
			if r == '\n' {
				line++
			}
		}

		// Avanzar el puntero principal hasta el final del lexema reconocido
		// para continuar la tokenizacion desde la siguiente posicion.
		i = lastOKPos
	}

	return tokens, errors
}
