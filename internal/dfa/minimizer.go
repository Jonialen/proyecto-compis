// minimizer.go implementa la minimizacion de un AFD (Automata Finito Determinista)
// usando el algoritmo de llenado de tabla (Table-Filling), tambien conocido como
// algoritmo de marcado de pares.
//
// A diferencia del algoritmo de Hopcroft (que usa particionamiento), este algoritmo:
//  1. Crea una tabla triangular de pares de estados
//  2. Marca como distinguibles los pares donde uno es de aceptacion y el otro no
//  3. Itera propagando marcas hasta alcanzar un punto fijo
//  4. Agrupa estados no distinguibles usando Union-Find con compresion de caminos
//  5. Reconstruye el AFD minimal con los grupos resultantes
//
// Complejidad: O(n^2 * k) donde n es el numero de estados y k el tamanno del alfabeto.
// Es mas simple de implementar que Hopcroft aunque menos eficiente para AFDs grandes.
package dfa

// Minimize aplica el algoritmo de llenado de tabla para reducir un AFD a su
// numero minimo de estados fusionando estados equivalentes.
// Dos estados son equivalentes si para toda cadena de entrada posible, ambos
// llevan a aceptacion o ambos llevan a rechazo.
//
// Parametros:
//   - d: el AFD a minimizar
//
// Retorna:
//   - *DFA: un nuevo AFD con el minimo numero de estados posible, equivalente al original
//
// El algoritmo tiene 5 fases principales:
//  1. Preparacion: obtener lista ordenada de estados y crear indice
//  2. Tabla de distinguibilidad: crear matriz triangular e inicializar marcas base
//  3. Propagacion iterativa: propagar distinguibilidad transitivamente
//  4. Agrupacion: usar Union-Find para agrupar estados equivalentes
//  5. Reconstruccion: construir el AFD minimal a partir de los grupos
func Minimize(d *DFA) *DFA {
	if len(d.States) == 0 {
		return d
	}

	// === FASE 1: Preparacion ===
	// Obtener una lista estable y ordenada de IDs de estado para poder
	// indexar la tabla triangular de forma determinista.
	stateList := sortedStateIDs(d)
	n := len(stateList)
	if n == 0 {
		return d
	}

	// Crear un indice inverso: dado un ID de estado original, obtener su indice
	// en stateList. Esto permite busqueda O(1) al consultar la tabla.
	stateIndex := make(map[int]int, n)
	for i, s := range stateList {
		stateIndex[s] = i
	}

	// === FASE 2: Tabla de distinguibilidad ===
	// Matriz triangular superior donde marked[i][j] = true (con i < j) indica que
	// los estados con indices i y j han sido demostrados como no equivalentes (distinguibles).
	// Se usa solo la mitad superior para ahorrar memoria (la relacion es simetrica).
	marked := make([][]bool, n)
	for i := range marked {
		marked[i] = make([]bool, n)
	}

	// Funcion auxiliar para marcar un par de estados como distinguibles.
	// Normaliza los indices para que siempre i < j (mitad superior de la matriz).
	mark := func(i, j int) {
		if i > j {
			i, j = j, i
		}
		marked[i][j] = true
	}
	// Funcion auxiliar para consultar si un par de estados ya esta marcado como distinguible.
	isMarked := func(i, j int) bool {
		if i > j {
			i, j = j, i
		}
		return marked[i][j]
	}

	// Marcado inicial (caso base): un par de estados es trivialmente distinguible
	// si uno es de aceptacion y el otro no. Esta es la semilla del algoritmo.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			si, sj := stateList[i], stateList[j]
			if d.Accepting[si] != d.Accepting[sj] {
				mark(i, j)
			}
		}
	}

	// === FASE 3: Propagacion iterativa ===
	// Para cada par de estados no marcados (i, j), verificar si sus transiciones
	// con algun simbolo llevan a un par de estados (ri, rj) ya marcado como distinguible.
	// Si es asi, (i, j) tambien son distinguibles.
	// Se repite hasta que no se puedan hacer mas marcas (punto fijo).
	changed := true
	for changed {
		changed = false
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				// Saltar pares ya marcados como distinguibles.
				if isMarked(i, j) {
					continue
				}
				si, sj := stateList[i], stateList[j]

				// Recolectar la union de todos los simbolos usados en las transiciones
				// de ambos estados. Se necesita verificar todos los simbolos posibles.
				symbols := make(map[rune]bool)
				for sym := range d.Transitions[si] {
					symbols[sym] = true
				}
				for sym := range d.Transitions[sj] {
					symbols[sym] = true
				}

				for sym := range symbols {
					ri, hasRi := d.Transitions[si][sym]
					rj, hasRj := d.Transitions[sj][sym]

					if hasRi != hasRj {
						// Si un estado tiene transicion para este simbolo y el otro no,
						// los estados son distinguibles (uno puede avanzar y el otro no).
						mark(i, j)
						changed = true
						break
					}
					if !hasRi && !hasRj {
						// Ambos estados no tienen transicion para este simbolo: no aporta
						// informacion de distinguibilidad para este simbolo.
						continue
					}
					if ri == rj {
						// Ambos transicionan al mismo estado: no aporta distinguibilidad.
						continue
					}
					// Si los estados destino (ri, rj) ya estan marcados como distinguibles,
					// entonces los estados origen (si, sj) tambien lo son por transitividad.
					riIdx := stateIndex[ri]
					rjIdx := stateIndex[rj]
					if isMarked(riIdx, rjIdx) {
						mark(i, j)
						changed = true
						break
					}
				}
			}
		}
	}

	// === FASE 4: Agrupacion con Union-Find ===
	// Los pares de estados que NO fueron marcados como distinguibles son equivalentes.
	// Se usa una estructura Union-Find (tambien conocida como Disjoint Set Union)
	// con compresion de caminos para agrupar eficientemente los estados equivalentes
	// en clases de equivalencia.

	// Inicializar: cada estado es su propio representante.
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	// find retorna el representante de la clase de equivalencia de x.
	// Aplica compresion de caminos para optimizar consultas futuras:
	// cada nodo visitado se conecta directamente a la raiz.
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x]) // Compresion de caminos.
		}
		return parent[x]
	}

	// union fusiona las clases de equivalencia de x y y en una sola.
	union := func(x, y int) {
		px, py := find(x), find(y)
		if px != py {
			parent[px] = py
		}
	}

	// Unir todos los pares de estados que no fueron marcados como distinguibles.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if !isMarked(i, j) {
				union(i, j)
			}
		}
	}

	// Asignar un nuevo ID de estado unico a cada clase de equivalencia (grupo).
	// classRep mapea el indice del representante del grupo a su nuevo ID en el AFD minimal.
	classRep := make(map[int]int)
	newStateID := 0
	// indexToNew mapea cada indice original a su nuevo ID de estado en el AFD minimal.
	indexToNew := make(map[int]int)

	for i := 0; i < n; i++ {
		rep := find(i)
		if _, ok := classRep[rep]; !ok {
			classRep[rep] = newStateID
			newStateID++
		}
		indexToNew[i] = classRep[rep]
	}

	// === FASE 5: Reconstruccion del AFD minimal ===
	// Se construye un nuevo AFD donde cada estado representa una clase de equivalencia
	// completa del AFD original.
	newDFA := &DFA{
		States:      make(map[int]map[int]bool),
		Transitions: make(map[int]map[rune]int),
		Accepting:   make(map[int]bool),
		StateToken:  make(map[int]string),
	}

	// Mapear el estado inicial original a su representante en el AFD minimal.
	startIdx := stateIndex[d.Start]
	newDFA.Start = indexToNew[startIdx]

	for i, s := range stateList {
		newS := indexToNew[i]

		// Fusionar los conjuntos de posiciones del arbol sintactico de todos
		// los estados en la misma clase de equivalencia.
		if newDFA.States[newS] == nil {
			newDFA.States[newS] = make(map[int]bool)
		}
		for p := range d.States[s] {
			newDFA.States[newS][p] = true
		}

		// Preservar el estatus de aceptacion y la asociacion con el nombre del token.
		// Si cualquier estado del grupo era de aceptacion, el grupo es de aceptacion.
		if d.Accepting[s] {
			newDFA.Accepting[newS] = true
			if tok, ok := d.StateToken[s]; ok {
				newDFA.StateToken[newS] = tok
			}
		}

		// Reconstruir las transiciones: las transiciones de cualquier estado del grupo
		// apuntan al representante de la clase de equivalencia del estado destino.
		for sym, next := range d.Transitions[s] {
			nextIdx := stateIndex[next]
			newNext := indexToNew[nextIdx]
			if newDFA.Transitions[newS] == nil {
				newDFA.Transitions[newS] = make(map[rune]int)
			}
			newDFA.Transitions[newS][sym] = newNext
		}
	}

	return newDFA
}

// sortedStateIDs proporciona un ordenamiento estable y determinista de los IDs de estado.
// Esto es importante porque los mapas en Go no garantizan orden de iteracion,
// y el algoritmo de minimizacion necesita un orden consistente para indexar
// la tabla triangular de distinguibilidad.
//
// Parametros:
//   - d: el AFD cuyos IDs de estado se van a ordenar
//
// Retorna:
//   - []int: IDs de estado ordenados de menor a mayor
//
// Utiliza ordenamiento por insercion, adecuado para las cantidades tipicamente
// pequenas de estados en AFDs generados a partir de expresiones regulares.
func sortedStateIDs(d *DFA) []int {
	ids := make([]int, 0, len(d.States))
	for id := range d.States {
		ids = append(ids, id)
	}
	// Ordenamiento por insercion: para cada elemento, se desplaza hacia la izquierda
	// hasta encontrar su posicion correcta. Es estable y eficiente para arreglos pequenos.
	for i := 1; i < len(ids); i++ {
		key := ids[i]
		j := i - 1
		for j >= 0 && ids[j] > key {
			ids[j+1] = ids[j]
			j--
		}
		ids[j+1] = key
	}
	return ids
}
