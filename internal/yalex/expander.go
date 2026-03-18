// Este archivo implementa la expansion de macros para patrones de expresiones regulares YALex.
//
// La expansion resuelve las referencias a macros dentro de los patrones de las reglas de tokens.
// Las macros pueden referenciarse entre si, por lo que la expansion se realiza de forma recursiva
// con deteccion de ciclos para evitar recursion infinita.
//
// Ejemplo de expansion:
//
//	let digit = ['0'-'9']
//	let number = digit+           ->  se expande a: (['0'-'9'])+
//	| number { TOKEN_NUM }        ->  patron expandido: ((['0'-'9'])+)
package yalex

import (
	"fmt"
	"strings"
)

// Expand sustituye las referencias a macros en los patrones de las reglas de tokens.
// Primero expande las macros entre si (ya que una macro puede referenciar a otra),
// y luego aplica las macros expandidas a cada patron de regla.
//
// Parametros:
//   - macros: mapa de nombre de macro -> expresion regular sin expandir
//   - rules: lista de reglas de tokens con patrones que pueden contener referencias a macros
//
// Retorna:
//   - []TokenRule: nueva lista de reglas con todos los patrones completamente expandidos
//   - error: si hay macros indefinidas, referencias ciclicas, u otros errores de expansion
func Expand(macros map[string]string, rules []TokenRule) ([]TokenRule, error) {
	// Primero expandir las macros internamente (pueden referenciarse entre si).
	// Esto produce un mapa donde cada macro tiene su expresion completamente resuelta.
	expanded, err := expandMacros(macros)
	if err != nil {
		return nil, err
	}

	// Luego expandir cada patron de regla usando las macros ya completamente resueltas
	result := make([]TokenRule, len(rules))
	for i, rule := range rules {
		expandedPattern, err := expandPattern(rule.Pattern, expanded)
		if err != nil {
			return nil, fmt.Errorf("expanding rule %d (%q): %w", i, rule.Pattern, err)
		}
		result[i] = TokenRule{
			Pattern:  expandedPattern,
			Action:   rule.Action,
			Priority: rule.Priority,
		}
	}
	return result, nil
}

// expandMacros expande completamente todas las macros en orden topologico implicito.
// Usa un enfoque de expansion bajo demanda (lazy) con memoizacion y deteccion de ciclos.
//
// Estructuras de control:
//   - expanded: macros ya completamente expandidas (cache/memoizacion)
//   - visited: macros que ya se intentaron expandir (para evitar trabajo duplicado)
//   - inStack: macros actualmente en la pila de recursion (para detectar ciclos)
//
// Algoritmo:
//  1. Para cada macro, llamar a expandOne (si no esta ya expandida)
//  2. expandOne verifica si ya esta en cache (retorna inmediatamente)
//  3. Si esta en la pila actual -> ciclo detectado -> error
//  4. Si no, marca como en pila, expande recursivamente, guarda en cache
//
// Este enfoque resuelve automaticamente el orden topologico sin necesitar
// un sort explicito, ya que las dependencias se resuelven por demanda.
func expandMacros(macros map[string]string) (map[string]string, error) {
	expanded := make(map[string]string) // Cache de macros ya expandidas
	visited := make(map[string]bool)    // Macros ya procesadas
	inStack := make(map[string]bool)    // Macros en la pila de recursion actual (deteccion de ciclos)

	// expandOne es la funcion recursiva que expande una macro individual.
	// Se llama a si misma indirectamente a traves de expandPatternWithResolver
	// cuando el patron de la macro contiene referencias a otras macros.
	var expandOne func(name string) (string, error)
	expandOne = func(name string) (string, error) {
		// Si ya esta expandida, retornar el resultado cacheado
		if v, ok := expanded[name]; ok {
			return v, nil
		}
		// Si esta en la pila de recursion actual, hay una referencia circular
		if inStack[name] {
			return "", fmt.Errorf("cyclic macro reference: %s", name)
		}
		// Verificar que la macro existe
		val, ok := macros[name]
		if !ok {
			return "", fmt.Errorf("undefined macro: %s", name)
		}

		// Detectar auto-referencia literal: si el valor recortado es un unico
		// identificador igual al nombre de la macro (ej: let or = or),
		// tratar el valor como secuencia de caracteres literales en vez de
		// referencia a macro. Se convierte cada caracter a comilla simple
		// para que el normalizador lo trate como literal.
		trimmedVal := strings.TrimSpace(val)
		if trimmedVal == name && isSingleIdent(trimmedVal) {
			var literal strings.Builder
			for _, ch := range trimmedVal {
				literal.WriteRune('\'')
				literal.WriteRune(ch)
				literal.WriteRune('\'')
			}
			expanded[name] = literal.String()
			return literal.String(), nil
		}

		// Marcar como en la pila de recursion para deteccion de ciclos
		inStack[name] = true
		visited[name] = true

		// Expandir el valor de la macro usando el resolver recursivo.
		// Se usa expandPatternWithResolver en vez de expandPattern simple
		// para garantizar que las macros no resueltas no se traten silenciosamente
		// como identificadores literales.
		result, err := expandPatternWithResolver(val, expanded, expandOne)
		if err != nil {
			return "", fmt.Errorf("macro %s: %w", name, err)
		}

		// Quitar de la pila de recursion y guardar en cache
		inStack[name] = false
		expanded[name] = result
		return result, nil
	}

	// Iterar sobre todas las macros y expandir cada una
	for name := range macros {
		if _, err := expandOne(name); err != nil {
			return nil, err
		}
	}
	return expanded, nil
}

// expandPattern reemplaza identificadores de macros en un patron con sus valores expandidos.
// Solo reemplaza identificadores que NO estan dentro de comillas o corchetes.
// Es un wrapper sobre expandPatternWithResolver que usa un resolver simple de busqueda
// directa en el mapa de macros.
//
// Parametros:
//   - pattern: el patron de expresion regular que puede contener referencias a macros
//   - macros: mapa de macros ya completamente expandidas
//
// Retorna:
//   - string: patron con todas las macros sustituidas
//   - error: si se referencia una macro indefinida
func expandPattern(pattern string, macros map[string]string) (string, error) {
	return expandPatternWithResolver(pattern, macros, func(name string) (string, error) {
		if v, ok := macros[name]; ok {
			return v, nil
		}
		return "", fmt.Errorf("undefined macro: %s", name)
	})
}

// expandPatternWithResolver es la funcion central de expansion de patrones.
// Recorre el patron caracter por caracter, identificando los diferentes contextos
// sintacticos y sustituyendo las referencias a macros por sus valores expandidos.
//
// Contextos sintacticos manejados:
//   - Literales entre comillas simples ('c'): se copian tal cual sin buscar macros
//   - Cadenas entre comillas dobles ("abc"): se copian tal cual sin buscar macros
//   - Clases de caracteres ([...]): se copian tal cual sin buscar macros
//   - Parentesis y operadores: se pasan directamente al resultado
//   - Identificadores: se intentan resolver como macros
//
// Cuando un identificador se resuelve como macro, su valor expandido se envuelve
// en parentesis para preservar la precedencia de operadores. Por ejemplo:
//   - Si "digit" = ['0'-'9'], entonces "digit+" se convierte en "(['0'-'9'])+"
//
// Si el resolver retorna un error de "undefined macro", el identificador se escribe
// tal cual (podria ser un literal). Cualquier otro error se propaga.
//
// Parametros:
//   - pattern: patron a expandir
//   - macros: mapa de macros ya expandidas (para busquedas directas)
//   - resolver: funcion que resuelve un nombre de macro a su valor expandido
//     (permite inyectar la logica recursiva con deteccion de ciclos)
//
// Retorna:
//   - string: patron completamente expandido
//   - error: si hay errores de expansion (ciclos, macros indefinidas en contexto recursivo, etc.)
func expandPatternWithResolver(pattern string, macros map[string]string, resolver func(string) (string, error)) (string, error) {
	runes := []rune(pattern)
	var result strings.Builder
	i := 0

	for i < len(runes) {
		c := runes[i]

		// --- Literales entre comillas simples ('c') ---
		// Se copian tal cual sin buscar macros dentro.
		// Se manejan secuencias de escape (\n, \t, etc.) dentro de las comillas.
		if c == '\'' {
			result.WriteRune(c)
			i++
			for i < len(runes) && runes[i] != '\'' {
				if runes[i] == '\\' && i+1 < len(runes) {
					result.WriteRune(runes[i])
					i++
					result.WriteRune(runes[i])
					i++
				} else {
					result.WriteRune(runes[i])
					i++
				}
			}
			if i < len(runes) {
				result.WriteRune(runes[i]) // Comilla de cierre
				i++
			}
			continue
		}

		// --- Cadenas entre comillas dobles ("abc") ---
		// Se copian tal cual, incluyendo secuencias de escape.
		if c == '"' {
			result.WriteRune(c)
			i++
			for i < len(runes) && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < len(runes) {
					result.WriteRune(runes[i])
					i++
					result.WriteRune(runes[i])
					i++
				} else {
					result.WriteRune(runes[i])
					i++
				}
			}
			if i < len(runes) {
				result.WriteRune(runes[i]) // Comilla de cierre
				i++
			}
			continue
		}

		// --- Clases de caracteres entre corchetes [...] ---
		// Se copian literalmente sin buscar macros dentro.
		// Se manejan corchetes anidados con un contador de profundidad.
		if c == '[' {
			result.WriteRune(c)
			i++
			depth := 1
			for i < len(runes) && depth > 0 {
				if runes[i] == '[' {
					depth++
				} else if runes[i] == ']' {
					depth--
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// --- Parentesis ---
		// Se pasan directamente al resultado (son parte de la sintaxis de agrupacion de regex).
		if c == '(' || c == ')' {
			result.WriteRune(c)
			i++
			continue
		}

		// --- Operadores de regex ---
		// Pipe (|), estrella (*), mas (+), interrogacion (?), punto (.), guion bajo (_)
		// se pasan directamente al resultado sin intentar resolver como macros.
		if c == '|' || c == '*' || c == '+' || c == '?' || c == '.' || c == '_' {
			result.WriteRune(c)
			i++
			continue
		}

		// --- Identificadores (posibles referencias a macros) ---
		// Si encontramos un caracter que puede iniciar un identificador,
		// recolectamos el identificador completo y lo intentamos resolver como macro.
		if isIdentStart(c) {
			start := i
			for i < len(runes) && isIdentContinue(runes[i]) {
				i++
			}
			ident := string(runes[start:i])

			// Intentar resolver el identificador como macro.
			// Si se resuelve exitosamente, se envuelve en parentesis para
			// preservar la precedencia (ej: "digit+" -> "(def_de_digit)+").
			resolved, err := resolver(ident)
			if err == nil {
				result.WriteRune('(')
				result.WriteString(resolved)
				result.WriteRune(')')
			} else if strings.Contains(err.Error(), "undefined macro") {
				// No es una macro conocida: escribir el identificador tal cual.
				// Esto permite que identificadores que no son macros pasen sin error,
				// por ejemplo keywords literales en algunos contextos.
				result.WriteString(ident)
			} else {
				// Otro tipo de error (ej: referencia ciclica): propagar
				return "", err
			}
			continue
		}

		// --- Cualquier otro caracter ---
		// Espacios, tabs, caracteres especiales no manejados arriba: se copian tal cual.
		result.WriteRune(c)
		i++
	}

	return strings.TrimSpace(result.String()), nil
}

// isSingleIdent verifica si un string es exactamente un identificador valido
// (una secuencia de caracteres que comienza con letra o guion bajo y continua
// con letras, digitos o guion bajo, sin espacios ni operadores).
func isSingleIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	runes := []rune(s)
	if !isIdentStart(runes[0]) {
		return false
	}
	for _, r := range runes[1:] {
		if !isIdentContinue(r) {
			return false
		}
	}
	return true
}

// isIdentStart determina si un caracter puede ser el inicio de un identificador.
// Los identificadores validos comienzan con una letra (A-Z, a-z) o guion bajo (_).
func isIdentStart(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

// isIdentContinue determina si un caracter puede ser parte de un identificador
// (despues del primer caracter). Acepta letras, guion bajo y digitos (0-9).
func isIdentContinue(c rune) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
