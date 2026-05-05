// Package yalex proporciona un parser para el lenguaje de especificacion YALex (Yet Another Lex).
//
// Este paquete se encarga de traducir archivos .yal a estructuras en memoria que definen
// las macros (definiciones reutilizables de regex) y las reglas de tokens del analizador lexico.
//
// El formato YALex soporta:
//   - Comentarios delimitados por (* ... *)
//   - Bloques de encabezado y trailer entre llaves { ... }
//   - Definiciones de macros con la sintaxis: let IDENT = expresion_regular
//   - Reglas de tokens con la sintaxis: rule nombre = | patron { ACCION } | ...
package yalex

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TokenRule representa una regla lexica individual extraida del archivo .yal.
// Cada regla asocia un patron de expresion regular con una accion (nombre del token).
type TokenRule struct {
	Pattern  string // Patron de expresion regular que define que texto reconoce esta regla.
	Action   string // Nombre del token o accion asociada al patron (ej. "TOKEN_ID", "TOKEN_NUM").
	Priority int    // Orden de aparicion (base 0) en el archivo .yal; se usa para desambiguacion.
}

// ParseResult encapsula todos los componentes extraidos de una especificacion YALex parseada.
type ParseResult struct {
	Macros map[string]string // Definiciones reutilizables de regex (let IDENT = expresion).
	Rules  []TokenRule       // Secuencia ordenada de reglas de tokens (rule nombre = | patron { accion }).
}

// ParseFile lee una especificacion YALex desde la ruta de archivo indicada y retorna
// su representacion parseada. Es un wrapper conveniente sobre Parse que se encarga
// de la lectura del archivo.
//
// Parametros:
//   - path: ruta al archivo .yal a parsear
//
// Retorna:
//   - *ParseResult: macros y reglas extraidas
//   - error: si no se puede leer o parsear el archivo
func ParseFile(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading yal file: %w", err)
	}
	return Parse(string(data))
}

// Parse procesa el contenido crudo de un archivo .yal y extrae sus componentes.
// Ejecuta las siguientes etapas en orden:
//  1. Eliminar comentarios YALex delimitados por (* y *)
//  2. Eliminar bloques de encabezado (header) y trailer delimitados por { }
//  3. Extraer definiciones de macros con la sintaxis "let IDENT = regex"
//  4. Localizar la seccion de reglas (rule nombre = ...) y parsear cada regla individual
//
// Parametros:
//   - content: contenido completo del archivo .yal como string
//
// Retorna:
//   - *ParseResult: estructura con macros y reglas extraidas
//   - error: si hay errores de sintaxis en las reglas
func Parse(content string) (*ParseResult, error) {
	// Paso 1: Eliminar comentarios YALex delimitados por (* y *).
	// Esto se hace primero para evitar que el contenido de los comentarios
	// interfiera con el parseo posterior.
	content = removeComments(content)

	// Paso 2: Eliminar bloques de encabezado y trailer delimitados por llaves.
	// Estos bloques contienen codigo auxiliar que no es parte de las reglas lexicas.
	content = removeHeaderTrailer(content)

	result := &ParseResult{
		Macros: make(map[string]string),
	}

	// Paso 3: Extraer definiciones de macros.
	// Se buscan lineas con el formato: let IDENTIFICADOR = expresion_regular
	// La regex permite espacios flexibles entre los componentes.
	letRe := regexp.MustCompile(`(?m)^[ \t]*let[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t]*=[ \t]*(.+)$`)
	for _, m := range letRe.FindAllStringSubmatch(content, -1) {
		name := strings.TrimSpace(m[1])
		val := strings.TrimSpace(m[2])
		result.Macros[name] = val
	}

	// Paso 4: Identificar la seccion de reglas.
	// Se busca el bloque que comienza con "rule IDENTIFICADOR = ..." y se extrae
	// todo lo que sigue como el cuerpo de las reglas.
	ruleRe := regexp.MustCompile(`(?s)rule\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*(.+)`)
	ruleMatch := ruleRe.FindStringSubmatch(content)
	if ruleMatch == nil {
		// Si no hay seccion de reglas, se retorna solo con las macros (sin error).
		return result, nil
	}
	ruleBody := ruleMatch[1]

	// Paso 5: Parsear las reglas individuales del cuerpo.
	// Cada regla tiene el formato: | patron { ACCION }
	rules, err := parseRules(ruleBody)
	if err != nil {
		return nil, err
	}
	result.Rules = rules
	return result, nil
}

// removeComments elimina comentarios multilinea delimitados por (* ... *) del texto de entrada.
// Recorre el string caracter por caracter buscando la secuencia de apertura "(*" y luego
// la secuencia de cierre "*)". Todo el contenido entre ellos se omite.
// Si un comentario no tiene cierre, se trunca todo desde la apertura hasta el final.
func removeComments(s string) string {
	var buf strings.Builder
	i := 0
	for i < len(s) {
		// Detectar inicio de comentario: secuencia (*
		if i+1 < len(s) && s[i] == '(' && s[i+1] == '*' {
			// Buscar la secuencia de cierre *) correspondiente
			end := strings.Index(s[i+2:], "*)")
			if end == -1 {
				// Si el comentario no se cierra, se descarta todo el resto del texto.
				break
			}
			// Saltar todo el contenido del comentario, incluyendo los delimitadores.
			// i+2 es donde empieza el contenido, +end llega al inicio de *), +2 salta el *)
			i = i + 2 + end + 2
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String()
}

// removeHeaderTrailer aisla y elimina los bloques de encabezado { header } y trailer { trailer }
// que aparecen fuera de la definicion de reglas. La estrategia es:
//  1. Dividir el texto en "antes de rule" (prefijo) y "desde rule en adelante" (sufijo)
//  2. Eliminar bloques { } del prefijo (encabezado)
//  3. Eliminar el bloque { } final del sufijo (trailer)
func removeHeaderTrailer(s string) string {
	// Localizar donde empieza la seccion "rule" para separar el texto.
	ruleIdx := strings.Index(s, "rule ")
	if ruleIdx == -1 {
		ruleIdx = len(s)
	}

	prefix := s[:ruleIdx] // Todo antes de "rule": puede contener el bloque header
	suffix := s[ruleIdx:] // Desde "rule" en adelante: puede contener el bloque trailer al final

	// Eliminar bloques de llaves balanceadas del prefijo (encabezado)
	prefix = removeTopLevelBraces(prefix)
	// Eliminar el bloque de llaves final del sufijo (trailer)
	suffix = removeTrailingBraceBlock(suffix)

	return prefix + suffix
}

// removeTrailingBraceBlock identifica y elimina el bloque de trailer opcional
// al final de la especificacion. El trailer es el ultimo bloque { ... } que aparece
// despues de que todas las reglas ya terminaron (es decir, despues del ultimo } de una accion).
//
// Algoritmo:
//  1. Recorrer desde el final del string hacia atras contando llaves
//  2. Encontrar el { de apertura que corresponde al ultimo }
//  3. Verificar que antes de este { hay otro } (el cierre de la ultima regla)
//  4. Si la condicion se cumple, el bloque es un trailer y se elimina
func removeTrailingBraceBlock(s string) string {
	lastClose := strings.LastIndex(s, "}")
	if lastClose == -1 {
		return s
	}

	runes := []rune(s)
	depth := 0
	trailerStart := -1
	// Recorrer desde el final hacia atras para encontrar el { del ultimo bloque balanceado
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '}' {
			depth++
		} else if runes[i] == '{' {
			depth--
			if depth == 0 {
				trailerStart = i
				break
			}
		}
	}
	if trailerStart == -1 {
		return s
	}

	// Verificar que el texto antes del supuesto trailer termina con "}",
	// lo que confirma que hay reglas anteriores y esto es realmente un trailer.
	before := strings.TrimSpace(string(runes[:trailerStart]))
	if !strings.HasSuffix(before, "}") {
		return s
	}

	return before
}

// removeTopLevelBraces elimina todos los bloques balanceados { ... } de nivel superior
// del string proporcionado. Se usa para eliminar el bloque de encabezado del prefijo.
// Mantiene todo el texto que esta fuera de las llaves; descarta el contenido dentro de ellas.
func removeTopLevelBraces(s string) string {
	var buf strings.Builder
	depth := 0
	for _, c := range s {
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
		} else if depth == 0 {
			// Solo se escribe el caracter si esta fuera de cualquier bloque de llaves
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

// parseRules descompone el cuerpo de la seccion de reglas en pares patron-accion individuales.
// Maneja el caso de reglas agrupadas donde multiples patrones comparten una sola accion:
//
//	| patron1
//	| patron2
//	| patron3 { ACCION_COMPARTIDA }
//
// En este caso, los tres patrones reciben la misma accion.
//
// Algoritmo:
//  1. Dividir el cuerpo por pipes (|) de nivel superior
//  2. Para cada segmento, verificar si contiene un bloque de accion { ... }
//  3. Si tiene accion: todos los segmentos pendientes (sin accion) reciben esa accion
//  4. Si no tiene accion: se acumula en la lista de pendientes
//  5. Al final, si quedan segmentos sin accion asignada, se reporta error
//
// Parametros:
//   - body: el cuerpo de la seccion rule (todo despues de "rule nombre =")
//
// Retorna:
//   - []TokenRule: lista ordenada de reglas con patron, accion y prioridad
//   - error: si algun patron queda sin accion asignada
func parseRules(body string) ([]TokenRule, error) {
	var rules []TokenRule
	priority := 0 // Contador incremental que asigna prioridad por orden de aparicion

	// Dividir el cuerpo por pipes de nivel superior (respetando comillas y parentesis)
	segments := splitByPipe(body)

	var pending []string // Patrones acumulados esperando un bloque de accion
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		// Intentar extraer un bloque de accion { ... } del segmento
		_, _, err := extractPatternAction(seg)
		hasAction := (err == nil)

		if hasAction {
			// Este segmento tiene accion -> asignar la accion a todos los patrones pendientes
			pending = append(pending, seg)
			_, action, _ := extractPatternAction(pending[len(pending)-1])
			action = strings.TrimSpace(action)

			// Crear una regla para cada patron pendiente con la misma accion
			for _, p := range pending {
				pattern, _, _ := extractPatternAction(p)
				pattern = strings.TrimSpace(pattern)
				if pattern == "" {
					continue
				}
				rules = append(rules, TokenRule{
					Pattern:  pattern,
					Action:   action,
					Priority: priority,
				})
				priority++
			}
			pending = pending[:0] // Limpiar la lista de pendientes
		} else {
			// Este segmento no tiene accion -> acumular y esperar al siguiente con accion
			pending = append(pending, seg)
		}
	}

	// Si quedan segmentos pendientes sin accion, es un error de sintaxis
	for _, p := range pending {
		if strings.TrimSpace(p) != "" {
			return nil, fmt.Errorf("parsing rule %q: no action block found", p)
		}
	}

	return rules, nil
}

// splitByPipe divide el cuerpo de reglas por caracteres pipe (|) de nivel superior.
// "Nivel superior" significa que se ignoran los pipes que aparecen dentro de:
//   - Comillas simples ('...')
//   - Comillas dobles ("...")
//   - Corchetes ([...]) -- clases de caracteres
//   - Llaves ({...}) -- bloques de accion
//   - Parentesis ((...)) -- grupos de regex
//
// Se usa un conjunto de contadores de profundidad y banderas de estado para
// rastrear el contexto actual del caracter mientras se recorre el string.
//
// Parametros:
//   - s: el cuerpo de reglas a dividir
//
// Retorna:
//   - []string: segmentos resultantes de dividir por pipes de nivel superior
func splitByPipe(s string) []string {
	var parts []string
	var cur strings.Builder
	inSingle := false // Dentro de comillas simples
	inDouble := false // Dentro de comillas dobles
	inBracket := 0    // Profundidad de corchetes (clases de caracteres)
	inBrace := 0      // Profundidad de llaves (bloques de accion)
	inParen := 0      // Profundidad de parentesis (grupos de regex)

	i := 0
	runes := []rune(s)
	for i < len(runes) {
		c := runes[i]
		switch {
		// Alternar estado de comillas simples (solo fuera de comillas dobles y corchetes)
		case c == '\'' && !inDouble && inBracket == 0:
			inSingle = !inSingle
			cur.WriteRune(c)
		// Dentro de comillas simples: copiar todo literalmente, manejar escapes
		case inSingle:
			if c == '\\' && i+1 < len(runes) {
				cur.WriteRune(c)
				i++
				cur.WriteRune(runes[i])
			} else {
				cur.WriteRune(c)
			}
		// Alternar estado de comillas dobles
		case c == '"' && !inSingle && inBracket == 0:
			inDouble = !inDouble
			cur.WriteRune(c)
		// Entrada a clase de caracteres [...]
		case c == '[' && !inSingle && !inDouble:
			inBracket++
			cur.WriteRune(c)
		// Salida de clase de caracteres
		case c == ']' && !inSingle && !inDouble:
			if inBracket > 0 {
				inBracket--
			}
			cur.WriteRune(c)
		// Entrada a grupo de parentesis
		case c == '(' && !inSingle && !inDouble && inBracket == 0:
			inParen++
			cur.WriteRune(c)
		// Salida de grupo de parentesis
		case c == ')' && !inSingle && !inDouble && inBracket == 0:
			if inParen > 0 {
				inParen--
			}
			cur.WriteRune(c)
		// Entrada a bloque de accion
		case c == '{' && !inSingle && !inDouble:
			inBrace++
			cur.WriteRune(c)
		// Salida de bloque de accion
		case c == '}' && !inSingle && !inDouble:
			inBrace--
			cur.WriteRune(c)
		// Pipe de nivel superior: dividir aqui
		case c == '|' && !inSingle && !inDouble && inBracket == 0 && inBrace == 0 && inParen == 0:
			parts = append(parts, cur.String())
			cur.Reset()
		// Cualquier otro caracter: copiar al segmento actual
		default:
			cur.WriteRune(c)
		}
		i++
	}
	// No olvidar el ultimo segmento (despues del ultimo pipe)
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// extractPatternAction separa un segmento de regla en sus componentes de patron y accion.
// Busca el primer caracter '{' de nivel superior (fuera de comillas y corchetes)
// que marca el inicio del bloque de accion, luego encuentra la '}' correspondiente.
//
// Ejemplo: `  ['0'-'9']+ { TOKEN_NUM }  ` -> patron="['0'-'9']+", accion="TOKEN_NUM"
//
// Parametros:
//   - line: segmento de regla a analizar
//
// Retorna:
//   - pattern: la parte del patron (antes de {)
//   - action: el contenido dentro de { ... } (sin las llaves)
//   - err: si no se encuentra un bloque de accion, o si esta mal balanceado
func extractPatternAction(line string) (pattern, action string, err error) {
	runes := []rune(line)
	inSingle := false
	inDouble := false
	inBracket := 0
	actionStart := -1 // Posicion del '{' que inicia el bloque de accion

	// Fase 1: Buscar el '{' de nivel superior que inicia el bloque de accion.
	// Se respetan comillas y corchetes para no confundir '{' dentro de literales.
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '\'' && !inDouble && inBracket == 0:
			inSingle = !inSingle
		case inSingle:
			if c == '\\' && i+1 < len(runes) {
				i++ // Saltar el caracter escapado
			}
		case c == '"' && !inSingle && inBracket == 0:
			inDouble = !inDouble
		case c == '[' && !inSingle && !inDouble:
			inBracket++
		case c == ']' && !inSingle && !inDouble:
			if inBracket > 0 {
				inBracket--
			}
		case c == '{' && !inSingle && !inDouble && inBracket == 0:
			actionStart = i
			goto foundAction // Se encontro el inicio del bloque de accion
		}
	}

foundAction:
	if actionStart == -1 {
		return line, "", fmt.Errorf("no action block found")
	}

	// Fase 2: Encontrar la '}' que cierra el bloque de accion.
	// Se cuentan llaves anidadas para manejar bloques de accion que contengan
	// otras llaves (por ejemplo, codigo con condicionales).
	depth := 0
	actionEnd := -1
	for i := actionStart; i < len(runes); i++ {
		if runes[i] == '{' {
			depth++
		} else if runes[i] == '}' {
			depth--
			if depth == 0 {
				actionEnd = i
				break
			}
		}
	}
	if actionEnd == -1 {
		return "", "", fmt.Errorf("unclosed action block")
	}

	// Separar el patron (antes del {) y la accion (contenido entre { y })
	pattern = string(runes[:actionStart])
	action = string(runes[actionStart+1 : actionEnd])
	return pattern, action, nil
}
