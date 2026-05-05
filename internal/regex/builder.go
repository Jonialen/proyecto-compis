// builder.go -- Conversion de expresiones regulares tokenizadas a notacion postfija.
//
// Este archivo implementa la segunda etapa del procesamiento de regex:
// toma la secuencia de tokens normalizada (en notacion infija, con concatenaciones
// explicitas) y la convierte a notacion postfija (Reverse Polish Notation) usando
// el algoritmo Shunting-Yard de Edsger Dijkstra.
//
// Adicionalmente, "aumenta" la expresion envolviendola en parentesis y agregando
// el marcador de fin (EndMarker = '#') concatenado al final. Esto es necesario
// para la construccion directa del DFA, donde el marcador de fin permite
// identificar los estados de aceptacion.
//
// Ejemplo de aumentacion:
//
//	Entrada:  a | b
//	Aumentada: ( a | b ) · #
//	Postfijo:  a b | # ·
package regex

import (
	"fmt"
)

// BuildPostfix convierte una secuencia de tokens normalizada (notacion infija)
// en una secuencia en notacion postfija usando el algoritmo Shunting-Yard.
//
// Antes de la conversion, la expresion se "aumenta" agregando:
//  1. Parentesis que envuelven toda la expresion original: ( expr )
//  2. Un operador de concatenacion explicito (ConcatOp).
//  3. El marcador de fin (EndMarker = '\x00'), representado como '#'.
//
// El resultado es la expresion aumentada: ( expr ) · #
// convertida a notacion postfija, lista para construir el arbol sintactico
// y el DFA directo.
//
// Parametros:
//   - normalized: la secuencia de tokens producida por Normalize(), ya con
//     operadores de concatenacion explicitos.
//
// Retorna:
//   - []RegexToken: la secuencia en notacion postfija.
//   - error: si hay parentesis desbalanceados u otros errores de sintaxis.
func BuildPostfix(normalized []RegexToken) ([]RegexToken, error) {
	// --- Paso 1: Aumentar la expresion ---
	// Se envuelve la expresion en parentesis y se concatena el EndMarker.
	// La capacidad se pre-asigna para evitar reasignaciones: len + 4 tokens
	// adicionales (apertura, cierre, ConcatOp, EndMarker).
	augmented := make([]RegexToken, 0, len(normalized)+4)
	augmented = append(augmented, openTok())          // '(' de apertura
	augmented = append(augmented, normalized...)      // expresion original
	augmented = append(augmented, closeTok())         // ')' de cierre
	augmented = append(augmented, opTok(ConcatOp))    // operador de concatenacion '·'
	augmented = append(augmented, atomTok(EndMarker)) // marcador de fin '#'

	// --- Paso 2: Convertir a postfijo ---
	return shuntingYard(augmented)
}

// precedence retorna la precedencia de un operador regex.
// Se usa en el algoritmo Shunting-Yard para determinar el orden de
// evaluacion de los operadores.
//
// Niveles de precedencia (de mayor a menor):
//   - 3: Cuantificadores unarios postfijos (*, +, ?) -- se aplican primero.
//   - 2: Concatenacion (ConcatOp = '\x01') -- se aplica despues de cuantificadores.
//   - 1: Alternacion (|) -- se aplica al final, tiene la menor precedencia.
//   - 0: Cualquier otro (valor por defecto, no deberia ocurrir en entrada valida).
//
// Esta jerarquia asegura que "a|bc*" se interprete como "a|(b(c*))" y no
// como "(a|b)(c*)".
func precedence(op rune) int {
	switch op {
	case '*', '+', '?':
		return 3
	case ConcatOp:
		return 2
	case '|':
		return 1
	}
	return 0
}

// isUnary retorna true si el operador es un operador unario postfijo.
// Los operadores unarios (*, +, ?) se aplican al operando inmediatamente
// anterior y no requieren procesamiento especial en la pila de operadores
// del algoritmo Shunting-Yard (se emiten directamente a la salida).
// Esto se debe a que estos operadores ya estan en posicion postfija
// (aparecen despues de su operando).
func isUnary(op rune) bool {
	return op == '*' || op == '+' || op == '?'
}

// shuntingYard implementa el algoritmo Shunting-Yard de Dijkstra adaptado
// para expresiones regulares tokenizadas.
//
// El algoritmo convierte una expresion en notacion infija a notacion postfija
// usando dos estructuras:
//   - output: cola de salida donde se acumulan los tokens en orden postfijo.
//   - opStack: pila de operadores que se usa para reordenar segun precedencia.
//
// Reglas del algoritmo para cada tipo de token:
//  1. Atomo (literal): se envia directamente a la salida.
//  2. Parentesis de apertura '(': se empuja a la pila de operadores.
//  3. Parentesis de cierre ')': se desapilan operadores a la salida hasta
//     encontrar el '(' correspondiente. Si no se encuentra, error de parentesis.
//  4. Operador unario postfijo (*, +, ?): se envia directamente a la salida
//     (siempre tiene la mayor precedencia y ya esta en la posicion correcta).
//  5. Operador binario (ConcatOp, |): se desapilan operadores de la pila
//     hacia la salida mientras tengan precedencia >= al operador actual
//     (asociatividad izquierda), luego se empuja el operador actual.
//
// Al finalizar, se vacia la pila de operadores a la salida. Si queda un '('
// sin cerrar, se reporta error.
//
// Parametros:
//   - tokens: secuencia de tokens en notacion infija (ya aumentada).
//
// Retorna:
//   - []RegexToken: secuencia en notacion postfija.
//   - error: si hay parentesis desbalanceados.
func shuntingYard(tokens []RegexToken) ([]RegexToken, error) {
	var output []RegexToken  // Cola de salida (resultado en notacion postfija)
	var opStack []RegexToken // Pila de operadores para reordenamiento por precedencia

	for _, tok := range tokens {
		switch tok.Kind {
		case TokAtom:
			// Regla 1: Los atomos (literales) van directamente a la salida.
			output = append(output, tok)

		case TokOpen:
			// Regla 2: El parentesis de apertura se empuja a la pila.
			// Actua como barrera para que los operadores no se desapilen
			// mas alla del grupo actual.
			opStack = append(opStack, tok)

		case TokClose:
			// Regla 3: Desapilar operadores a la salida hasta encontrar
			// el parentesis de apertura correspondiente.
			for len(opStack) > 0 && opStack[len(opStack)-1].Kind != TokOpen {
				output = append(output, opStack[len(opStack)-1])
				opStack = opStack[:len(opStack)-1]
			}
			if len(opStack) == 0 {
				return nil, fmt.Errorf("mismatched parentheses: no matching (")
			}
			// Descartar el parentesis de apertura (ya cumplio su funcion).
			opStack = opStack[:len(opStack)-1]

		case TokOp:
			op := tok.Op
			if isUnary(op) {
				// Regla 4: Los operadores unarios postfijos (*, +, ?) se
				// emiten directamente a la salida. Siempre se aplican al
				// operando/grupo inmediatamente anterior y tienen la mayor
				// precedencia, por lo que no necesitan pasar por la pila.
				// Ya estan en posicion postfija naturalmente.
				output = append(output, tok)
			} else {
				// Regla 5: Operadores binarios (ConcatOp, |).
				// Se desapilan operadores de la pila mientras:
				//   a) La pila no este vacia.
				//   b) El tope no sea un parentesis de apertura.
				//   c) El tope sea un operador con precedencia >= al actual.
				// Esto implementa la asociatividad izquierda.
				for len(opStack) > 0 {
					top := opStack[len(opStack)-1]
					if top.Kind == TokOpen {
						break // No desapilar mas alla del grupo
					}
					if top.Kind != TokOp {
						break
					}
					topOp := top.Op
					// Asociatividad izquierda: desapilar si precedencia >= actual.
					if precedence(topOp) >= precedence(op) {
						output = append(output, top)
						opStack = opStack[:len(opStack)-1]
					} else {
						break
					}
				}
				// Empujar el operador actual a la pila.
				opStack = append(opStack, tok)
			}
		}
	}

	// Vaciar la pila de operadores restantes a la salida.
	// Si se encuentra un parentesis de apertura aqui, significa que
	// no tiene su cierre correspondiente -> error de sintaxis.
	for len(opStack) > 0 {
		top := opStack[len(opStack)-1]
		opStack = opStack[:len(opStack)-1]
		if top.Kind == TokOpen {
			return nil, fmt.Errorf("mismatched parentheses: unclosed (")
		}
		output = append(output, top)
	}

	return output, nil
}
