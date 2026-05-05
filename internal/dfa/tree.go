// Package dfa implementa la construccion y minimizacion de Automatas Finitos Deterministas (AFD)
// a partir de arboles sintacticos de expresiones regulares. Utiliza el algoritmo de construccion
// directa de AFD descrito en el Dragon Book (Aho, Sethi, Ullman), Seccion 3.9.
//
// El flujo general es:
//  1. Convertir la expresion regular (en postfijo) a un arbol sintactico (tree.go)
//  2. Calcular las funciones Nullable, FirstPos y LastPos (positions.go)
//  3. Calcular la tabla FollowPos (followpos.go)
//  4. Construir el AFD usando el algoritmo de lista de trabajo (builder.go)
//  5. Minimizar el AFD usando el algoritmo de llenado de tabla (minimizer.go)
package dfa

import (
	"fmt"
	"strings"

	"genanalex/internal/regex"
)

// NodeKind identifica el tipo de operacion o hoja en el arbol sintactico.
// Cada nodo del arbol puede ser una hoja (literal o epsilon) o un operador
// (concatenacion, alternancia, cerradura de Kleene, cerradura positiva, o opcional).
type NodeKind int

const (
	NodeLeaf    NodeKind = iota // Nodo hoja que contiene un simbolo de entrada.
	NodeEpsilon                 // Nodo epsilon (vacio) que representa la cadena vacia.
	NodeCat                     // Nodo de concatenacion (·) que une dos subexpresiones secuencialmente.
	NodeOr                      // Nodo de alternancia (|) que representa la union de dos subexpresiones.
	NodeStar                    // Nodo de cerradura de Kleene (*) para 0 o mas repeticiones.
	NodePlus                    // Nodo de cerradura positiva (+) para 1 o mas repeticiones.
	NodeOpt                     // Nodo opcional (?) para 0 o 1 ocurrencia.
)

// Node representa un solo elemento en el arbol sintactico de una expresion regular.
// Los nodos hoja (NodeLeaf) contienen un simbolo y una posicion unica; los nodos internos
// (operadores) tienen uno o dos hijos que representan sus operandos.
//
// Campos:
//   - Kind:   tipo de nodo (hoja, operador, epsilon)
//   - Symbol: el caracter literal (solo valido para NodeLeaf)
//   - Pos:    identificador de posicion unico (1-indexado), asignado solo a nodos hoja
//   - Left:   hijo izquierdo (primer operando)
//   - Right:  hijo derecho (segundo operando, nil para operadores unarios)
//
// Los campos de cache (nullableCache, firstPosCache, lastPosCache) se usan para
// memoizacion de las funciones Nullable, FirstPos y LastPos, evitando recalculos
// costosos durante el recorrido repetido del arbol.
type Node struct {
	Kind   NodeKind
	Symbol rune // El simbolo de entrada (valido para NodeLeaf).
	Pos    int  // El ID de posicion unico (1-indexado) asignado a nodos hoja.
	Left   *Node
	Right  *Node

	// Caches de memoizacion usados para optimizar el calculo de propiedades del AFD.
	// Se usa *bool para nullableCache para distinguir "no calculado" (nil) de "calculado como false".
	nullableCache *bool
	firstPosCache map[int]bool
	lastPosCache  map[int]bool
}

// BuildTree transforma una expresion regular en notacion postfija en un arbol sintactico.
// Utiliza un algoritmo basado en pila: los operandos se apilan y los operadores los desapilan
// para construir subarboles.
//
// Parametros:
//   - postfix: secuencia de tokens en notacion postfija generada por el modulo regex
//
// Retorna:
//   - *Node: el nodo raiz del arbol sintactico construido
//   - map[int]rune: mapeo de posiciones (enteros) a sus simbolos correspondientes,
//     necesario para la construccion posterior del AFD
//   - error: si la expresion postfija es invalida (operandos insuficientes, parentesis inesperados, etc.)
//
// Algoritmo:
//  1. Recorrer cada token de la expresion postfija
//  2. Si es un atomo (literal), crear un nodo hoja con posicion unica y apilarlo
//  3. Si es un operador binario (|, concat), desapilar dos operandos y crear un nodo padre
//  4. Si es un operador unario (*, +, ?), desapilar un operando y crear un nodo padre
//  5. Al final, la pila debe contener exactamente un nodo: la raiz del arbol
func BuildTree(postfix []regex.RegexToken) (*Node, map[int]rune, error) {
	var stack []*Node
	posCounter := 0                   // Contador global de posiciones para asignar IDs unicos a hojas.
	posToSymbol := make(map[int]rune) // Mapeo de posicion -> simbolo, usado luego por BuildDFA.

	for _, tok := range postfix {
		switch tok.Kind {
		case regex.TokAtom:
			// Cada simbolo literal se convierte en un nodo hoja con una posicion unica.
			// El contador de posiciones se incrementa para garantizar unicidad.
			posCounter++
			leaf := &Node{
				Kind:   NodeLeaf,
				Symbol: tok.Atom,
				Pos:    posCounter,
			}
			posToSymbol[posCounter] = tok.Atom
			stack = append(stack, leaf)

		case regex.TokOp:
			// Los operadores desapilan sus operandos de la pila y empujan un nuevo subarbol.
			switch tok.Op {
			case '|':
				// Operador de alternancia: necesita dos operandos (izquierdo y derecho).
				if len(stack) < 2 {
					return nil, nil, fmt.Errorf("not enough operands for |")
				}
				r := stack[len(stack)-1]
				l := stack[len(stack)-2]
				stack = stack[:len(stack)-2]
				stack = append(stack, &Node{Kind: NodeOr, Left: l, Right: r})

			case regex.ConcatOp:
				// Operador de concatenacion: necesita dos operandos (izquierdo y derecho).
				if len(stack) < 2 {
					return nil, nil, fmt.Errorf("not enough operands for concat")
				}
				r := stack[len(stack)-1]
				l := stack[len(stack)-2]
				stack = stack[:len(stack)-2]
				stack = append(stack, &Node{Kind: NodeCat, Left: l, Right: r})

			case '*':
				// Cerradura de Kleene: necesita un operando (0 o mas repeticiones).
				if len(stack) < 1 {
					return nil, nil, fmt.Errorf("not enough operands for *")
				}
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stack = append(stack, &Node{Kind: NodeStar, Left: n})

			case '+':
				// Cerradura positiva: necesita un operando (1 o mas repeticiones).
				if len(stack) < 1 {
					return nil, nil, fmt.Errorf("not enough operands for +")
				}
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stack = append(stack, &Node{Kind: NodePlus, Left: n})

			case '?':
				// Operador opcional: necesita un operando (0 o 1 ocurrencia).
				if len(stack) < 1 {
					return nil, nil, fmt.Errorf("not enough operands for ?")
				}
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stack = append(stack, &Node{Kind: NodeOpt, Left: n})

			default:
				return nil, nil, fmt.Errorf("unknown operator: %q", tok.Op)
			}

		case regex.TokOpen, regex.TokClose:
			// Los parentesis se utilizan durante la conversion a postfijo pero no deben
			// aparecer en la expresion postfija final. Si aparecen, es un error.
			return nil, nil, fmt.Errorf("unexpected parenthesis in postfix")
		}
	}

	// Al finalizar, la pila debe contener exactamente un nodo (la raiz del arbol).
	// Si hay mas de uno, la expresion postfija tiene operandos sobrantes.
	// Si hay cero, la expresion estaba vacia.
	if len(stack) != 1 {
		return nil, nil, fmt.Errorf("syntax tree build error: %d nodes remaining (expected 1)", len(stack))
	}

	return stack[0], posToSymbol, nil
}

// ToDOT genera una representacion en formato DOT de Graphviz para visualizar el arbol sintactico.
// Realiza un recorrido en preorden del arbol, asignando un ID numerico a cada nodo y
// creando las aristas correspondientes entre padres e hijos.
//
// Parametros:
//   - root: el nodo raiz del arbol sintactico
//
// Retorna:
//   - string: el codigo DOT completo que puede ser procesado por Graphviz para generar
//     una imagen del arbol
func ToDOT(root *Node) string {
	var sb strings.Builder
	sb.WriteString("digraph syntaxtree {\n")
	sb.WriteString("  node [shape=circle];\n")
	counter := 0 // Contador para asignar IDs unicos a cada nodo en el grafo DOT.

	// visit recorre el arbol recursivamente en preorden.
	// Asigna un ID al nodo actual, escribe su etiqueta, y luego visita
	// recursivamente los hijos izquierdo y derecho creando las aristas.
	var visit func(n *Node) int
	visit = func(n *Node) int {
		if n == nil {
			return -1
		}
		id := counter
		counter++
		label := nodeLabel(n)
		sb.WriteString(fmt.Sprintf("  %d [label=%q];\n", id, label))
		if n.Left != nil {
			leftID := visit(n.Left)
			sb.WriteString(fmt.Sprintf("  %d -> %d;\n", id, leftID))
		}
		if n.Right != nil {
			rightID := visit(n.Right)
			sb.WriteString(fmt.Sprintf("  %d -> %d;\n", id, rightID))
		}
		return id
	}
	visit(root)
	sb.WriteString("}\n")
	return sb.String()
}

// nodeLabel genera la etiqueta descriptiva para cada nodo en la representacion DOT.
// Para nodos hoja muestra el simbolo y su posicion; para el marcador de fin (#)
// muestra solo la posicion; para operadores muestra el simbolo del operador.
func nodeLabel(n *Node) string {
	switch n.Kind {
	case NodeLeaf:
		// Si es el marcador de fin de expresion, se muestra como "#(posicion)".
		if n.Symbol == regex.EndMarker {
			return fmt.Sprintf("#(%d)", n.Pos)
		}
		// Para hojas normales, se muestra el simbolo entre comillas y su posicion.
		return fmt.Sprintf("%q(%d)", n.Symbol, n.Pos)
	case NodeEpsilon:
		return "ε"
	case NodeCat:
		return "·"
	case NodeOr:
		return "|"
	case NodeStar:
		return "*"
	case NodePlus:
		return "+"
	case NodeOpt:
		return "?"
	}
	return "?"
}
