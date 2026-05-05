// positions.go implementa las funciones Nullable, FirstPos y LastPos del algoritmo
// de construccion directa de AFD (Dragon Book, Aho-Sethi-Ullman, Seccion 3.9).
//
// Estas tres funciones son fundamentales para el calculo de FollowPos y la posterior
// construccion del AFD. Cada una se define recursivamente sobre el arbol sintactico
// y utiliza memoizacion para evitar recalculos redundantes.
//
// Reglas del Dragon Book:
//
//   - Nullable(epsilon) = true
//
//   - Nullable(hoja) = false
//
//   - Nullable(c1 | c2) = Nullable(c1) OR Nullable(c2)
//
//   - Nullable(c1 · c2) = Nullable(c1) AND Nullable(c2)
//
//   - Nullable(c*) = true
//
//   - Nullable(c+) = Nullable(c)
//
//   - Nullable(c?) = true
//
//   - FirstPos(epsilon) = vacio
//
//   - FirstPos(hoja i) = {i}
//
//   - FirstPos(c1 | c2) = FirstPos(c1) U FirstPos(c2)
//
//   - FirstPos(c1 · c2) = si Nullable(c1) entonces FirstPos(c1) U FirstPos(c2), sino FirstPos(c1)
//
//   - FirstPos(c*) = FirstPos(c+) = FirstPos(c?) = FirstPos(c)
//
//   - LastPos sigue reglas simetricas a FirstPos pero considerando el hijo derecho.
package dfa

import "genanalex/internal/regex"

// Nullable determina si un nodo del arbol sintactico puede generar la cadena vacia (epsilon).
// Esta propiedad es esencial para calcular FirstPos y LastPos correctamente en nodos
// de concatenacion, ya que si un hijo es anulable, las posiciones del otro hijo
// tambien pueden ser primeras o ultimas posiciones.
//
// Parametros:
//   - n: nodo del arbol sintactico a evaluar
//
// Retorna:
//   - true si el nodo puede derivar la cadena vacia, false en caso contrario
//
// Utiliza memoizacion con un puntero a bool (nullableCache) para distinguir entre
// "aun no calculado" (nil) y "calculado como false" (*bool apuntando a false).
func Nullable(n *Node) bool {
	if n == nil {
		return false
	}
	// Verificar si el resultado ya fue calculado y almacenado en cache.
	if n.nullableCache != nil {
		return *n.nullableCache
	}
	var result bool
	switch n.Kind {
	case NodeEpsilon:
		// Epsilon siempre puede generar la cadena vacia por definicion.
		result = true
	case NodeLeaf:
		// Una hoja (simbolo literal) nunca puede generar la cadena vacia,
		// ya que siempre consume exactamente un caracter.
		result = false
	case NodeOr:
		// La alternancia es anulable si alguno de sus hijos es anulable.
		// Regla: Nullable(c1 | c2) = Nullable(c1) OR Nullable(c2)
		result = Nullable(n.Left) || Nullable(n.Right)
	case NodeCat:
		// La concatenacion es anulable solo si AMBOS hijos son anulables.
		// Regla: Nullable(c1 · c2) = Nullable(c1) AND Nullable(c2)
		result = Nullable(n.Left) && Nullable(n.Right)
	case NodeStar:
		// La cerradura de Kleene siempre es anulable (acepta 0 repeticiones).
		result = true
	case NodePlus:
		// La cerradura positiva es anulable solo si su operando es anulable,
		// ya que requiere al menos una repeticion.
		result = Nullable(n.Left)
	case NodeOpt:
		// El operador opcional siempre es anulable (acepta 0 ocurrencias).
		result = true
	}
	// Almacenar el resultado en cache para futuros accesos.
	n.nullableCache = &result
	return result
}

// FirstPos calcula el conjunto de posiciones que pueden coincidir con el primer
// caracter de una cadena generada por el subarbol con raiz en n.
//
// Parametros:
//   - n: nodo del arbol sintactico
//
// Retorna:
//   - map[int]bool: conjunto de posiciones (representado como mapa para busqueda O(1))
//
// Utiliza memoizacion con firstPosCache para evitar recalculos.
func FirstPos(n *Node) map[int]bool {
	if n == nil {
		return map[int]bool{}
	}
	// Retornar resultado del cache si ya fue calculado.
	if n.firstPosCache != nil {
		return n.firstPosCache
	}
	var result map[int]bool
	switch n.Kind {
	case NodeEpsilon:
		// Epsilon no tiene posiciones, retorna conjunto vacio.
		result = map[int]bool{}
	case NodeLeaf:
		// Una hoja tiene como FirstPos unicamente su propia posicion.
		result = map[int]bool{n.Pos: true}
	case NodeOr:
		// Para alternancia, FirstPos es la union de FirstPos de ambos hijos.
		// Regla: FirstPos(c1 | c2) = FirstPos(c1) U FirstPos(c2)
		result = unionSets(FirstPos(n.Left), FirstPos(n.Right))
	case NodeCat:
		// Para concatenacion, si el hijo izquierdo es anulable, las primeras
		// posiciones incluyen tanto las del hijo izquierdo como las del derecho.
		// Esto es porque si c1 puede generar epsilon, c2 puede ser lo primero que se lee.
		// Regla: si Nullable(c1) entonces FirstPos(c1) U FirstPos(c2), sino FirstPos(c1)
		if Nullable(n.Left) {
			result = unionSets(FirstPos(n.Left), FirstPos(n.Right))
		} else {
			result = FirstPos(n.Left)
		}
	case NodeStar, NodePlus, NodeOpt:
		// Para operadores unarios, FirstPos es igual a FirstPos de su operando.
		// Regla: FirstPos(c*) = FirstPos(c+) = FirstPos(c?) = FirstPos(c)
		result = FirstPos(n.Left)
	default:
		result = map[int]bool{}
	}
	// Almacenar resultado en cache.
	n.firstPosCache = result
	return result
}

// LastPos calcula el conjunto de posiciones que pueden coincidir con el ultimo
// caracter de una cadena generada por el subarbol con raiz en n.
// Es la funcion simetrica de FirstPos.
//
// Parametros:
//   - n: nodo del arbol sintactico
//
// Retorna:
//   - map[int]bool: conjunto de posiciones
//
// Utiliza memoizacion con lastPosCache para evitar recalculos.
func LastPos(n *Node) map[int]bool {
	if n == nil {
		return map[int]bool{}
	}
	// Retornar resultado del cache si ya fue calculado.
	if n.lastPosCache != nil {
		return n.lastPosCache
	}
	var result map[int]bool
	switch n.Kind {
	case NodeEpsilon:
		// Epsilon no tiene posiciones, retorna conjunto vacio.
		result = map[int]bool{}
	case NodeLeaf:
		// Una hoja tiene como LastPos unicamente su propia posicion.
		result = map[int]bool{n.Pos: true}
	case NodeOr:
		// Para alternancia, LastPos es la union de LastPos de ambos hijos.
		// Regla: LastPos(c1 | c2) = LastPos(c1) U LastPos(c2)
		result = unionSets(LastPos(n.Left), LastPos(n.Right))
	case NodeCat:
		// Para concatenacion, si el hijo derecho es anulable, las ultimas
		// posiciones incluyen tanto las del hijo derecho como las del izquierdo.
		// Esto es porque si c2 puede generar epsilon, c1 puede ser lo ultimo que se lee.
		// Regla: si Nullable(c2) entonces LastPos(c1) U LastPos(c2), sino LastPos(c2)
		if Nullable(n.Right) {
			result = unionSets(LastPos(n.Left), LastPos(n.Right))
		} else {
			result = LastPos(n.Right)
		}
	case NodeStar, NodePlus, NodeOpt:
		// Para operadores unarios, LastPos es igual a LastPos de su operando.
		// Regla: LastPos(c*) = LastPos(c+) = LastPos(c?) = LastPos(c)
		result = LastPos(n.Left)
	default:
		result = map[int]bool{}
	}
	// Almacenar resultado en cache.
	n.lastPosCache = result
	return result
}

// unionSets calcula la union de dos conjuntos de posiciones.
// Crea un nuevo mapa conteniendo todas las posiciones presentes en a y/o en b.
// Se usa internamente por FirstPos y LastPos para combinar conjuntos.
//
// Parametros:
//   - a, b: conjuntos de posiciones representados como map[int]bool
//
// Retorna:
//   - map[int]bool: un nuevo conjunto con la union de a y b
func unionSets(a, b map[int]bool) map[int]bool {
	result := make(map[int]bool)
	for k := range a {
		result[k] = true
	}
	for k := range b {
		result[k] = true
	}
	return result
}

// setKey convierte un conjunto de posiciones a una clave de cadena canonica para uso en mapas.
// Esto permite comparar conjuntos de posiciones eficientemente: dos conjuntos son iguales
// si y solo si producen la misma clave. Se usa en BuildDFA para detectar si un estado
// ya fue creado (deduplicacion de estados).
//
// Parametros:
//   - s: conjunto de posiciones
//
// Retorna:
//   - string: representacion canonica ordenada, e.g. "1,3,5"
//
// El conjunto se ordena para garantizar que el mismo conjunto siempre produzca
// la misma clave independientemente del orden de insercion.
func setKey(s map[int]bool) string {
	if len(s) == 0 {
		return ""
	}
	keys := sortedKeys(s)
	result := make([]byte, 0, len(keys)*4)
	for i, k := range keys {
		if i > 0 {
			result = append(result, ',')
		}
		result = appendInt(result, k)
	}
	return string(result)
}

// appendInt convierte un entero no negativo a su representacion ASCII y lo agrega
// al slice de bytes. Es una alternativa eficiente a fmt.Sprintf para evitar
// asignaciones de memoria innecesarias en la generacion de claves.
//
// Parametros:
//   - b: slice de bytes al que se agrega el numero
//   - n: entero no negativo a convertir
//
// Retorna:
//   - []byte: el slice con el numero agregado al final
func appendInt(b []byte, n int) []byte {
	if n == 0 {
		return append(b, '0')
	}
	// Se usa un buffer temporal de 20 bytes (suficiente para int64 en base 10).
	// Se llena de derecha a izquierda extrayendo digitos con modulo 10.
	var tmp [20]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte('0' + n%10)
		n /= 10
	}
	return append(b, tmp[i:]...)
}

// sortedKeys extrae las claves de un conjunto de posiciones y las retorna ordenadas.
// Utiliza un algoritmo de ordenamiento por seleccion simple (O(n^2)) que es adecuado
// para los conjuntos pequenos tipicos en la construccion de AFD.
//
// Parametros:
//   - s: conjunto de posiciones
//
// Retorna:
//   - []int: posiciones ordenadas de menor a mayor
func sortedKeys(s map[int]bool) []int {
	keys := make([]int, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	// Ordenamiento por seleccion: para cada posicion i, busca el minimo en el
	// resto del arreglo y lo intercambia con la posicion i.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// EndMarkerSym es el simbolo especial que marca el fin de la expresion regular aumentada.
// Se utiliza para identificar la posicion del marcador de fin (#) al determinar
// cuales estados del AFD son estados de aceptacion.
var EndMarkerSym = regex.EndMarker
