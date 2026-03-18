// Package regex proporciona herramientas para normalizar, tokenizar y convertir
// expresiones regulares a notacion postfija, adecuada para la construccion directa
// de automatas finitos deterministas (DFA).
//
// El flujo general de procesamiento es:
//  1. Normalizacion (normalizer.go): convierte un patron regex crudo en una
//     secuencia de tokens ([]RegexToken), expandiendo clases de caracteres,
//     resolviendo secuencias de escape, manejando comodines y agregando
//     operadores de concatenacion explicitos.
//  2. Conversion a postfijo (builder.go): transforma la secuencia de tokens
//     en notacion infija a notacion postfija usando el algoritmo Shunting-Yard,
//     aumentando la expresion con un marcador de fin (#) para la construccion
//     directa del DFA.
package regex

import (
	"fmt"
	"strings"
	"unicode"
)

// ConcatOp es una runa centinela que representa el operador de concatenacion explicito.
// Se usa el caracter no imprimible '\x01' (SOH) para evitar colisiones con caracteres
// del alfabeto de entrada. Se inserta durante la normalizacion para facilitar la
// construccion del arbol sintactico y la conversion a postfijo.
const ConcatOp = '\x01'

// EndMarker es un simbolo especial de aumentacion ('#') que marca el final de una
// expresion regular. Es esencial para identificar los estados de aceptacion durante
// la construccion directa del DFA. Se usa el caracter nulo '\x00' (NUL) como
// valor centinela.
const EndMarker = '\x00'

// TokKind define el tipo de un token de expresion regular.
// Se usa como enumeracion para distinguir las cuatro categorias de tokens
// que produce el normalizador.
type TokKind int

const (
	TokAtom  TokKind = iota // TokAtom: un caracter literal o simbolo del alfabeto.
	TokOp                   // TokOp: un operador regex (|, *, +, ?, ConcatOp).
	TokOpen                 // TokOpen: un parentesis de apertura '('.
	TokClose                // TokClose: un parentesis de cierre ')'.
)

// RegexToken representa un elemento individual en una expresion regular tokenizada
// o en notacion postfija. Cada token pertenece a una de las categorias definidas
// por TokKind.
//
// Campos:
//   - Kind: indica la categoria del token (atomo, operador, parentesis).
//   - Atom: contiene el caracter literal; solo es valido cuando Kind == TokAtom.
//   - Op:   contiene el simbolo del operador; solo es valido cuando Kind es
//     TokOp, TokOpen o TokClose.
type RegexToken struct {
	Kind TokKind
	Atom rune
	Op   rune
}

// --- Funciones auxiliares de construccion de tokens (funciones fabrica) ---
// Estas funciones son constructores compactos que simplifican la creacion de
// tokens y mejoran la legibilidad del codigo en el tokenizador y otros modulos.
// Se definen como funciones fabrica en lugar de usar literales de struct
// directamente para centralizar la logica de inicializacion y reducir errores.

// atomTok crea un token de tipo atomo (literal) con la runa dada.
// Es el constructor mas usado: cada caracter del patron se convierte en un atomo.
func atomTok(r rune) RegexToken { return RegexToken{Kind: TokAtom, Atom: r} }

// opTok crea un token de tipo operador con el operador dado.
// Se usa para operadores regex (|, *, +, ?) y para el operador de concatenacion
// explicito (ConcatOp).
func opTok(op rune) RegexToken { return RegexToken{Kind: TokOp, Op: op} }

// openTok crea un token de parentesis de apertura '('.
// El campo Op se inicializa con '(' para facilitar la depuracion y la
// serializacion con TokensToString.
func openTok() RegexToken { return RegexToken{Kind: TokOpen, Op: '('} }

// closeTok crea un token de parentesis de cierre ')'.
// El campo Op se inicializa con ')' para facilitar la depuracion y la
// serializacion con TokensToString.
func closeTok() RegexToken { return RegexToken{Kind: TokClose, Op: ')'} }

// Normalize transforma un patron YALex crudo en una secuencia de RegexTokens.
// Este es el punto de entrada principal del normalizador.
//
// Pasos internos:
//  1. Tokeniza el patron: expande clases de caracteres, maneja comodines,
//     resuelve secuencias de escape y convierte literales entrecomillados.
//  2. Inserta operadores de concatenacion explicitos (ConcatOp) donde la
//     concatenacion es implicita en la sintaxis regex estandar.
//
// Parametros:
//   - pattern: el patron regex en formato YALex (puede incluir comillas simples,
//     comillas dobles, clases de caracteres, comodines, etc.).
//
// Retorna:
//   - []RegexToken: la secuencia de tokens normalizada con concatenaciones explicitas.
//   - error: si el patron contiene errores de sintaxis (comillas sin cerrar,
//     clases de caracteres vacias, etc.).
func Normalize(pattern string) ([]RegexToken, error) {
	tokens, err := tokenize(pattern)
	if err != nil {
		return nil, fmt.Errorf("tokenizing %q: %w", pattern, err)
	}
	// Paso 2: insertar operadores de concatenacion explicitos (·) donde
	// la concatenacion es implicita en la sintaxis regex. Por ejemplo,
	// "ab" se convierte en "a·b", y "a(b)" se convierte en "a·(b)".
	return insertConcat(tokens), nil
}

// tokenize convierte el string del patron de entrada en una secuencia plana
// de RegexTokens. Recorre el patron caracter por caracter y maneja:
//   - Parentesis: '(' y ')' se convierten directamente en tokens TokOpen/TokClose.
//   - Operadores: '|', '*', '+', '?' se convierten en tokens TokOp.
//   - Literales entre comillas simples: 'a', '\n', '\" — un solo caracter.
//   - Literales entre comillas dobles: "abc" — se expande como secuencia
//     concatenada (a·b·c) envuelta en parentesis.
//   - Clases de caracteres: [a-z], [0-9], [^0-9] — se expanden como
//     alternaciones (a|b|c|...) agrupadas en parentesis.
//   - Comodines: '.' y '_' — se expanden al alfabeto completo como alternacion.
//   - Espacios en blanco: se ignoran (son separadores en la sintaxis YALex).
//   - Otros caracteres: se tratan como atomos literales.
func tokenize(pattern string) ([]RegexToken, error) {
	var tokens []RegexToken
	runes := []rune(pattern)
	i := 0

	for i < len(runes) {
		c := runes[i]

		switch c {
		// --- Parentesis: se convierten directamente a tokens de agrupacion ---
		case '(':
			tokens = append(tokens, openTok())
			i++
		case ')':
			tokens = append(tokens, closeTok())
			i++

		// --- Operadores regex: alternacion, cuantificadores ---
		case '|':
			tokens = append(tokens, opTok('|'))
			i++
		case '*':
			tokens = append(tokens, opTok('*'))
			i++
		case '+':
			tokens = append(tokens, opTok('+'))
			i++
		case '?':
			tokens = append(tokens, opTok('?'))
			i++

		case '\'':
			// --- Literales entre comillas simples ---
			// Formato: 'c' o '\n' (un solo caracter, posiblemente escapado).
			// Se avanza pasando la comilla de apertura, se parsea el caracter
			// (con posible escape), y se verifica la comilla de cierre.
			i++ // Saltar comilla de apertura
			r, newI, err := parseSingleQuoted(runes, i)
			if err != nil {
				return nil, err
			}
			i = newI
			if i >= len(runes) || runes[i] != '\'' {
				return nil, fmt.Errorf("expected closing ' at position %d", i)
			}
			i++ // Saltar comilla de cierre
			tokens = append(tokens, atomTok(r))

		case '"':
			// --- Literales entre comillas dobles ---
			// Formato: "abc" -- se convierte en la secuencia (a·b·c).
			// Se recolectan todos los caracteres (resolviendo escapes) hasta
			// encontrar la comilla de cierre.
			i++ // Saltar comilla de apertura
			var chars []rune
			for i < len(runes) && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < len(runes) {
					// Secuencia de escape dentro de comillas dobles
					r, err := parseEscape(runes[i+1])
					if err != nil {
						return nil, err
					}
					chars = append(chars, r)
					i += 2
				} else {
					chars = append(chars, runes[i])
					i++
				}
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("unclosed double quote")
			}
			i++ // Saltar comilla de cierre

			// Generar tokens segun la longitud del string:
			if len(chars) == 0 {
				// Cadena vacia (epsilon) -- se omite porque no se manejan
				// transiciones vacias en esta implementacion.
			} else if len(chars) == 1 {
				// Un solo caracter: se emite directamente como atomo.
				tokens = append(tokens, atomTok(chars[0]))
			} else {
				// Multiples caracteres: se envuelven en parentesis con
				// concatenacion explicita entre cada par.
				// Ejemplo: "abc" -> ( a · b · c )
				tokens = append(tokens, openTok())
				for j, r := range chars {
					tokens = append(tokens, atomTok(r))
					if j < len(chars)-1 {
						tokens = append(tokens, opTok(ConcatOp))
					}
				}
				tokens = append(tokens, closeTok())
			}

		case '[':
			// --- Clases de caracteres ---
			// Formato: [a-z], [0-9], [^a-z] (negada), [' ' '\t'], etc.
			// Se delega a expandCharClass que retorna los tokens equivalentes
			// como una alternacion agrupada: (a|b|c|...).
			i++ // Saltar '['
			classTokens, newI, err := expandCharClass(runes, i)
			if err != nil {
				return nil, fmt.Errorf("expanding char class: %w", err)
			}
			i = newI
			tokens = append(tokens, classTokens...)

		case '.':
			// --- Comodin (punto) ---
			// Coincide con cualquier caracter del alfabeto (excluyendo '\n').
			// Se expande como una alternacion de todos los caracteres del alfabeto.
			classTokens := buildWildcardTokens()
			tokens = append(tokens, classTokens...)
			i++

		case '_':
			// --- Comodin alternativo (guion bajo) ---
			// Sintaxis comun en YALex para representar "cualquier caracter".
			// Se expande igual que el punto '.'.
			classTokens := buildWildcardTokens()
			tokens = append(tokens, classTokens...)
			i++

		default:
			if unicode.IsSpace(c) {
				// Los espacios en blanco se ignoran como separadores en YALex.
				i++
				continue
			}
			// Cualquier otro caracter se trata como un atomo literal.
			tokens = append(tokens, atomTok(c))
			i++
		}
	}

	return tokens, nil
}

// parseSingleQuoted extrae un caracter literal de una secuencia entre comillas simples.
// Maneja tanto caracteres literales ('a') como secuencias de escape ('\n', '\\', etc.).
//
// Parametros:
//   - runes: el arreglo completo de runas del patron.
//   - i: la posicion actual (justo despues de la comilla de apertura).
//
// Retorna:
//   - rune: el caracter literal extraido.
//   - int:  la nueva posicion del cursor despues de consumir el caracter.
//   - error: si se llega al final inesperadamente.
func parseSingleQuoted(runes []rune, i int) (rune, int, error) {
	if i >= len(runes) {
		return 0, i, fmt.Errorf("unexpected end inside single quote")
	}
	if runes[i] == '\\' {
		// Es una secuencia de escape: consumir la barra invertida y el caracter siguiente.
		if i+1 >= len(runes) {
			return 0, i, fmt.Errorf("unexpected end after backslash")
		}
		r, err := parseEscape(runes[i+1])
		if err != nil {
			return 0, i, err
		}
		return r, i + 2, nil
	}
	// Caracter literal sin escape: se retorna directamente.
	return runes[i], i + 1, nil
}

// parseEscape mapea un caracter escapado a su runa correspondiente.
// Se usa despues de encontrar una barra invertida '\' para interpretar
// secuencias de escape estandar.
//
// Secuencias reconocidas:
//   - \n -> salto de linea (0x0A)
//   - \t -> tabulacion (0x09)
//   - \r -> retorno de carro (0x0D)
//   - \\ -> barra invertida literal
//   - \' -> comilla simple literal
//   - \" -> comilla doble literal
//   - \0 -> caracter nulo (0x00)
//
// Si el caracter no corresponde a ninguna secuencia conocida, se retorna
// el caracter tal cual (comportamiento tolerante para secuencias desconocidas).
func parseEscape(c rune) (rune, error) {
	switch c {
	case 'n':
		return '\n', nil
	case 't':
		return '\t', nil
	case 'r':
		return '\r', nil
	case '\\':
		return '\\', nil
	case '\'':
		return '\'', nil
	case '"':
		return '"', nil
	case '0':
		return '\x00', nil
	default:
		// Secuencias de escape desconocidas se tratan como el caracter literal.
		return c, nil
	}
}

// expandCharClass parsea un bloque [...] y lo convierte en una alternacion
// agrupada equivalente: (a|b|c|...).
//
// Funcionalidades soportadas:
//   - Rangos literales: [a-z], [0-9]
//   - Rangos con escape: [\x00-\xFF]
//   - Caracteres individuales: [abc]
//   - Caracteres entre comillas simples dentro de la clase: [' ' '\t']
//   - Secuencias de escape: [\n\t\\]
//   - Negacion (complemento): [^a-z] incluye todos los caracteres del
//     alfabeto que NO estan en el conjunto especificado.
//   - Espacios y tabulaciones como separadores (estilo YALex).
//
// Parametros:
//   - runes: el arreglo completo de runas del patron.
//   - i: la posicion actual (justo despues del '[' de apertura).
//
// Retorna:
//   - []RegexToken: tokens de la alternacion agrupada, ej: (a|b|c).
//   - int: la nueva posicion del cursor (despues del ']' de cierre).
//   - error: si la clase de caracteres esta vacia o tiene errores de sintaxis.
func expandCharClass(runes []rune, i int) ([]RegexToken, int, error) {
	// Verificar si la clase es negada (complemento): [^...]
	complement := false
	if i < len(runes) && runes[i] == '^' {
		complement = true
		i++
	}

	// Recolectar todos los caracteres que pertenecen a la clase.
	var chars []rune

	for i < len(runes) && runes[i] != ']' {
		c := runes[i]

		if c == '\'' {
			// Caracter entre comillas simples dentro de la clase.
			// Ejemplo: [' ' '\t'] -- permite incluir espacios y caracteres
			// especiales de forma explicita.
			i++
			r, newI, err := parseSingleQuoted(runes, i)
			if err != nil {
				return nil, i, err
			}
			i = newI
			if i < len(runes) && runes[i] == '\'' {
				i++ // Saltar la comilla de cierre
			}
			chars = append(chars, r)
		} else if c == '\\' {
			// Secuencia de escape dentro de la clase de caracteres.
			if i+1 >= len(runes) {
				return nil, i, fmt.Errorf("unexpected end after backslash in class")
			}
			r, err := parseEscape(runes[i+1])
			if err != nil {
				return nil, i, err
			}
			i += 2
			// Verificar si el caracter escapado es el inicio de un rango.
			// Ejemplo: [\x00-\xFF] -> rango desde el caracter escapado
			// hasta el caracter final.
			if i+1 < len(runes) && runes[i] == '-' && runes[i+1] != ']' {
				i++ // Saltar el '-'
				end := runes[i]
				i++
				for r2 := r; r2 <= end; r2++ {
					chars = append(chars, r2)
				}
			} else {
				chars = append(chars, r)
			}
		} else if c == ' ' || c == '\t' {
			// Los espacios y tabulaciones actuan como separadores en
			// las clases de caracteres de YALex (no son parte de la clase).
			i++
		} else {
			// Caracter literal. Verificar si es el inicio de un rango.
			// Ejemplo: [a-z] -> rango de 'a' a 'z'.
			if i+2 < len(runes) && runes[i+1] == '-' && runes[i+2] != ']' {
				start := c
				end := runes[i+2]
				i += 3
				// Generar todos los caracteres del rango inclusive.
				for r := start; r <= end; r++ {
					chars = append(chars, r)
				}
			} else {
				chars = append(chars, c)
				i++
			}
		}
	}

	// Consumir el ']' de cierre si existe.
	if i < len(runes) && runes[i] == ']' {
		i++
	}

	if complement {
		// Para clases negadas [^...]: se construye el complemento.
		// Se obtiene el alfabeto completo y se excluyen los caracteres
		// que si estan en la clase, dejando solo los que NO estan.
		alphabet := buildAlphabet()
		exclude := make(map[rune]bool)
		for _, c := range chars {
			exclude[c] = true
		}
		var remaining []rune
		for _, r := range alphabet {
			if !exclude[r] {
				remaining = append(remaining, r)
			}
		}
		chars = remaining
	}

	if len(chars) == 0 {
		return nil, i, fmt.Errorf("empty character class")
	}

	// Convertir la lista de caracteres en una alternacion agrupada: (a|b|c|...).
	return buildAlternationGroup(chars), i, nil
}

// buildAlternationGroup crea una secuencia de tokens que representa una
// alternacion agrupada (a|b|c|...) a partir de una lista de runas.
//
// Si la lista tiene un solo caracter, se retorna un solo token atomo
// (sin parentesis ni operadores de alternacion).
// Si tiene multiples caracteres, se envuelven en parentesis con operadores
// '|' entre cada par: ( a | b | c ).
//
// Parametros:
//   - chars: la lista de caracteres a incluir en la alternacion.
//
// Retorna:
//   - []RegexToken: la secuencia de tokens representando la alternacion.
func buildAlternationGroup(chars []rune) []RegexToken {
	if len(chars) == 1 {
		return []RegexToken{atomTok(chars[0])}
	}
	tokens := []RegexToken{openTok()}
	for i, r := range chars {
		tokens = append(tokens, atomTok(r))
		if i < len(chars)-1 {
			tokens = append(tokens, opTok('|'))
		}
	}
	tokens = append(tokens, closeTok())
	return tokens
}

// buildWildcardTokens crea el grupo de tokens que representa el comodin
// (cualquier caracter del alfabeto). Es equivalente a una alternacion
// de todos los caracteres retornados por buildAlphabet().
func buildWildcardTokens() []RegexToken {
	return buildAlternationGroup(buildAlphabet())
}

// buildAlphabet retorna el conjunto de caracteres que conforman el alfabeto
// reconocido por el analizador lexico.
//
// El alfabeto incluye 97 caracteres:
//   - Caracteres de control: tabulacion (\t = 0x09) y retorno de carro (\r = 0x0D).
//   - ASCII imprimible: codigos 32 a 126 (espacio, letras, digitos, puntuacion).
//   - Excluye: salto de linea (\n = 0x0A), ya que se usa como delimitador de linea
//     y no debe coincidir con el comodin '.'.
//
// Este alfabeto se usa para expandir comodines ('.' y '_') y para calcular
// el complemento de clases de caracteres negadas ([^...]).
func buildAlphabet() []rune {
	var alphabet []rune
	// Agregar caracteres de control utiles: tabulacion y retorno de carro.
	alphabet = append(alphabet, '\t', '\r')
	// Agregar todos los caracteres ASCII imprimibles (32=' ' hasta 126='~').
	for r := rune(32); r <= 126; r++ {
		alphabet = append(alphabet, r)
	}
	return alphabet
}

// insertConcat agrega operadores de concatenacion explicitos (ConcatOp = '\x01')
// entre tokens adyacentes donde la concatenacion es implicita.
//
// Esto es necesario porque en la notacion regex estandar, la concatenacion no
// tiene un operador explicito (ej: "ab" implica a·b). Para convertir a postfijo
// con el algoritmo Shunting-Yard, necesitamos que todos los operadores binarios
// esten presentes de forma explicita.
//
// Recorre la secuencia de tokens y entre cada par consecutivo verifica si
// se necesita insertar un operador de concatenacion usando needsConcat().
func insertConcat(tokens []RegexToken) []RegexToken {
	var result []RegexToken

	for i, tok := range tokens {
		result = append(result, tok)
		if i+1 < len(tokens) {
			if needsConcat(tokens[i], tokens[i+1]) {
				result = append(result, opTok(ConcatOp))
			}
		}
	}

	return result
}

// needsConcat determina si se necesita un operador de concatenacion entre dos
// tokens adyacentes.
//
// La regla es: se necesita concatenacion cuando el token izquierdo "termina"
// una expresion y el token derecho "comienza" una nueva expresion.
//
// Un token "termina" una expresion si es:
//   - Un atomo (literal)
//   - Un parentesis de cierre ')'
//   - Un cuantificador postfijo (*, +, ?)
//
// Un token "comienza" una expresion si es:
//   - Un atomo (literal)
//   - Un parentesis de apertura '('
//
// Ejemplos donde se inserta concatenacion:
//   - a b  ->  a · b    (atomo seguido de atomo)
//   - a (  ->  a · (    (atomo seguido de grupo)
//   - ) a  ->  ) · a    (cierre de grupo seguido de atomo)
//   - a* b ->  a* · b   (cuantificador seguido de atomo)
//   - ) (  ->  ) · (    (cierre seguido de apertura)
func needsConcat(left, right RegexToken) bool {
	// Verificar si el token izquierdo "produce" un valor (termina una expresion).
	leftOutput := left.Kind == TokAtom ||
		left.Kind == TokClose ||
		(left.Kind == TokOp && (left.Op == '*' || left.Op == '+' || left.Op == '?'))

	// Verificar si el token derecho "consume" un valor (inicia una expresion).
	rightInput := right.Kind == TokAtom || right.Kind == TokOpen

	return leftOutput && rightInput
}

// TokensToString convierte una secuencia de RegexTokens en una representacion
// legible por humanos, util para depuracion, pruebas y logging.
//
// Esta funcion es exportada porque tambien se utiliza en los tests y en
// otros paquetes que necesitan inspeccionar las expresiones tokenizadas.
//
// Formato de salida:
//   - Atomos: se muestran como A('c') con el caracter entre comillas (usando %q
//     para mostrar caracteres de control de forma legible, ej: A('\n')).
//   - Operador de concatenacion (ConcatOp): se muestra como '·' (punto medio)
//     en lugar de '\x01' para mayor legibilidad.
//   - Otros operadores: se muestran tal cual (|, *, +, ?).
//   - Parentesis: se muestran como '(' y ')'.
//
// Ejemplo: para la expresion "ab|c", la salida seria: A('a')·A('b')|A('c')
func TokensToString(tokens []RegexToken) string {
	var sb strings.Builder
	for _, tok := range tokens {
		switch tok.Kind {
		case TokAtom:
			sb.WriteString(fmt.Sprintf("A(%q)", tok.Atom))
		case TokOp:
			if tok.Op == ConcatOp {
				sb.WriteString("·")
			} else {
				sb.WriteRune(tok.Op)
			}
		case TokOpen:
			sb.WriteRune('(')
		case TokClose:
			sb.WriteRune(')')
		}
	}
	return sb.String()
}
