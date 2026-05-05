// followpos.go calcula la tabla FollowPos para todas las posiciones del arbol sintactico.
//
// FollowPos es una funcion central en el algoritmo de construccion directa de AFD
// (Dragon Book, Seccion 3.9). Para cada posicion i en el arbol, FollowPos(i) es el
// conjunto de posiciones que pueden seguir inmediatamente a i en alguna cadena
// del lenguaje descrito por la expresion regular.
//
// Las reglas para calcular FollowPos son:
//
//  1. Concatenacion (c1 · c2): para cada posicion i en LastPos(c1),
//     se agregan todas las posiciones de FirstPos(c2) a FollowPos(i).
//     Esto modela que despues de terminar c1, puede comenzar c2.
//
//  2. Cerradura de Kleene (c*) y cerradura positiva (c+): para cada posicion i
//     en LastPos(c), se agregan todas las posiciones de FirstPos(c) a FollowPos(i).
//     Esto modela que despues de terminar una repeticion, puede comenzar otra.
//
//  3. Alternancia (c1 | c2) y opcional (c?): no contribuyen directamente a
//     FollowPos, pero se recorren recursivamente para procesar sus subarboles.
package dfa

// ComputeFollowPos calcula la funcion FollowPos para todas las posiciones del arbol sintactico.
// Recorre el arbol completo y aplica las reglas del Dragon Book para concatenacion
// y cerraduras (Kleene y positiva).
//
// Parametros:
//   - root: nodo raiz del arbol sintactico (debe estar ya aumentado con el marcador de fin #)
//
// Retorna:
//   - map[int]map[int]bool: tabla FollowPos donde la clave exterior es una posicion i
//     y el valor es el conjunto de posiciones que pueden seguir a i.
//     Ejemplo: followPos[3] = {5: true, 7: true} significa que las posiciones 5 y 7
//     pueden seguir a la posicion 3.
func ComputeFollowPos(root *Node) map[int]map[int]bool {
	followPos := make(map[int]map[int]bool)
	computeFollow(root, followPos)
	return followPos
}

// computeFollow recorre recursivamente el arbol sintactico y llena la tabla FollowPos.
// Solo los nodos de concatenacion (NodeCat) y cerraduras (NodeStar, NodePlus)
// contribuyen entradas nuevas a la tabla. Los demas nodos simplemente propagan
// la recursion a sus hijos.
//
// Parametros:
//   - n: nodo actual del arbol
//   - fp: tabla FollowPos que se va llenando incrementalmente
func computeFollow(n *Node, fp map[int]map[int]bool) {
	if n == nil {
		return
	}

	switch n.Kind {
	case NodeCat:
		// Regla de concatenacion (c1 · c2):
		// Para cada posicion i en LastPos del hijo izquierdo (c1),
		// agregar todas las posiciones de FirstPos del hijo derecho (c2)
		// al conjunto FollowPos(i).
		// Justificacion: si i es una de las ultimas posiciones de c1,
		// entonces las primeras posiciones de c2 pueden seguir a i.
		lp := LastPos(n.Left)
		fp1 := FirstPos(n.Right)
		for i := range lp {
			if fp[i] == nil {
				fp[i] = make(map[int]bool)
			}
			for j := range fp1 {
				fp[i][j] = true
			}
		}
		// Continuar recursivamente con ambos hijos para procesar
		// concatenaciones y cerraduras anidadas.
		computeFollow(n.Left, fp)
		computeFollow(n.Right, fp)

	case NodeStar, NodePlus:
		// Regla de cerraduras (* y +):
		// Para cada posicion i en LastPos del operando,
		// agregar todas las posiciones de FirstPos del operando
		// al conjunto FollowPos(i).
		// Justificacion: en una repeticion, despues de la ultima posicion
		// de una iteracion pueden venir las primeras posiciones de la siguiente.
		// Esto crea los ciclos necesarios en el AFD para reconocer repeticiones.
		lp := LastPos(n.Left)
		fp1 := FirstPos(n.Left)
		for i := range lp {
			if fp[i] == nil {
				fp[i] = make(map[int]bool)
			}
			for j := range fp1 {
				fp[i][j] = true
			}
		}
		// Continuar recursivamente con el operando.
		computeFollow(n.Left, fp)

	case NodeOr, NodeOpt:
		// La alternancia y el operador opcional no generan entradas en FollowPos,
		// pero debemos recorrer sus hijos para encontrar concatenaciones y
		// cerraduras anidadas dentro de ellos.
		computeFollow(n.Left, fp)
		if n.Right != nil {
			computeFollow(n.Right, fp)
		}

	case NodeLeaf, NodeEpsilon:
		// Los nodos hoja y epsilon son casos base de la recursion.
		// No generan entradas en FollowPos.
	}
}
