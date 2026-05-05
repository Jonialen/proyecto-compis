// builder.go construye un AFD (Automata Finito Determinista) directamente a partir
// del arbol sintactico de una expresion regular, sin pasar por un AFN intermedio.
// Implementa el algoritmo de construccion directa del Dragon Book (Seccion 3.9.5).
//
// El algoritmo utiliza una lista de trabajo (worklist) con estrategia BFS (busqueda en
// anchura) para explorar y crear estados. Cada estado del AFD corresponde a un conjunto
// de posiciones del arbol sintactico, y las transiciones se calculan usando la tabla
// FollowPos.
package dfa

import "genanalex/internal/regex"

// DFA representa un automata finito determinista.
//
// Campos:
//   - States:      mapea cada ID de estado a su conjunto de posiciones del arbol sintactico.
//     Cada estado es un subconjunto de posiciones que el AFD puede "estar mirando".
//   - Transitions: mapea cada ID de estado a sus transiciones, donde cada transicion asocia
//     un simbolo de entrada (rune) con el ID del estado destino.
//   - Start:       ID del estado inicial del AFD.
//   - Accepting:   conjunto de IDs de estados de aceptacion (estados finales).
//   - StateToken:  mapea cada estado de aceptacion al nombre del token que reconoce.
//     Esto permite que el analizador lexico sepa que tipo de token se encontro.
type DFA struct {
	States      map[int]map[int]bool // ID de estado -> conjunto de posiciones
	Transitions map[int]map[rune]int // ID de estado -> (simbolo -> ID de estado destino)
	Start       int
	Accepting   map[int]bool
	StateToken  map[int]string // ID de estado -> nombre del token (para estados de aceptacion)
}

// BuildDFA construye un AFD directamente a partir del arbol sintactico usando el
// algoritmo de construccion directa (Dragon Book, Seccion 3.9.5).
//
// Parametros:
//   - root: raiz del arbol sintactico (debe estar aumentado con el marcador de fin #)
//   - posToSymbol: mapeo de numero de posicion a su caracter/simbolo, generado por BuildTree
//   - tokenName: nombre del token que este AFD reconoce (e.g. "ID", "NUMBER", "IF")
//
// Retorna:
//   - *DFA: el automata finito determinista construido
//
// Algoritmo paso a paso:
//  1. Calcular la tabla FollowPos para todo el arbol
//  2. Encontrar la posicion del marcador de fin (#) en posToSymbol
//  3. Crear el estado inicial como FirstPos(raiz)
//  4. Usar una lista de trabajo (worklist/BFS) para explorar estados:
//     a. Para cada estado no procesado, encontrar todos los simbolos presentes
//     b. Para cada simbolo, calcular el estado destino como la union de FollowPos
//     de todas las posiciones en el estado actual que tienen ese simbolo
//     c. Si el estado destino es nuevo, agregarlo a la lista de trabajo
//     d. Registrar la transicion
//  5. Un estado es de aceptacion si contiene la posicion del marcador de fin (#)
func BuildDFA(root *Node, posToSymbol map[int]rune, tokenName string) *DFA {
	// Paso 1: Calcular la tabla FollowPos para todo el arbol sintactico.
	followPos := ComputeFollowPos(root)

	// Paso 2: Encontrar la posicion del marcador de fin (#) en el mapeo de simbolos.
	// Esta posicion se usa para determinar si un estado es de aceptacion.
	endPos := -1
	for pos, sym := range posToSymbol {
		if sym == regex.EndMarker {
			endPos = pos
		}
	}

	// Inicializar la estructura del AFD con mapas vacios.
	dfa := &DFA{
		States:      make(map[int]map[int]bool),
		Transitions: make(map[int]map[rune]int),
		Start:       0,
		Accepting:   make(map[int]bool),
		StateToken:  make(map[int]string),
	}

	// Paso 3: El estado inicial es FirstPos de la raiz del arbol.
	// Esto corresponde al conjunto de posiciones que pueden ser las primeras
	// en cualquier cadena aceptada por la expresion regular.
	initial := FirstPos(root)
	stateID := 0
	// keyToID mapea la representacion canonica de un conjunto de posiciones a su ID de estado.
	// Se usa para deduplicacion: evitar crear el mismo estado dos veces.
	keyToID := make(map[string]int)

	initialKey := setKey(initial)
	keyToID[initialKey] = stateID
	dfa.States[stateID] = initial

	// Si el estado inicial contiene la posicion del marcador de fin,
	// es un estado de aceptacion (la expresion acepta la cadena vacia).
	if initial[endPos] {
		dfa.Accepting[stateID] = true
		dfa.StateToken[stateID] = tokenName
	}

	// Paso 4: Algoritmo de lista de trabajo (BFS).
	// Se exploran estados no procesados hasta que no queden mas por procesar.
	var worklist []int
	worklist = append(worklist, stateID)
	stateID++

	for len(worklist) > 0 {
		// Extraer el primer estado de la lista (FIFO = BFS).
		curr := worklist[0]
		worklist = worklist[1:]

		currState := dfa.States[curr]

		// Paso 4a: Recolectar todos los simbolos que aparecen en las posiciones
		// del estado actual, excluyendo el marcador de fin (#).
		symbols := make(map[rune]bool)
		for pos := range currState {
			sym := posToSymbol[pos]
			if sym != regex.EndMarker {
				symbols[sym] = true
			}
		}

		// Paso 4b: Para cada simbolo, calcular el estado destino.
		for sym := range symbols {
			// U = union de FollowPos(i) para toda posicion i en el estado actual
			// cuyo simbolo asociado sea igual a sym.
			// Esto implementa la regla: delta(S, a) = U followpos(i) para i en S con simbolo(i) = a
			U := make(map[int]bool)
			for pos := range currState {
				if posToSymbol[pos] == sym {
					for fp := range followPos[pos] {
						U[fp] = true
					}
				}
			}

			// Si la union esta vacia, no hay transicion para este simbolo.
			if len(U) == 0 {
				continue
			}

			// Paso 4c: Verificar si este conjunto de posiciones ya corresponde a un estado existente.
			// Se usa setKey para generar una clave canonica y buscar en el mapa de deduplicacion.
			key := setKey(U)
			nextID, exists := keyToID[key]
			if !exists {
				// Es un estado nuevo: asignar ID, registrarlo y agregarlo a la lista de trabajo.
				nextID = stateID
				stateID++
				keyToID[key] = nextID
				dfa.States[nextID] = U
				worklist = append(worklist, nextID)

				// Verificar si el nuevo estado es de aceptacion
				// (contiene la posicion del marcador de fin #).
				if U[endPos] {
					dfa.Accepting[nextID] = true
					dfa.StateToken[nextID] = tokenName
				}
			}

			// Paso 4d: Registrar la transicion del estado actual al estado destino con el simbolo.
			if dfa.Transitions[curr] == nil {
				dfa.Transitions[curr] = make(map[rune]int)
			}
			dfa.Transitions[curr][sym] = nextID
		}
	}

	return dfa
}
