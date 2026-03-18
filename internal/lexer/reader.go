// Package lexer proporciona las herramientas para leer archivos fuente
// y realizar analisis lexico (tokenizacion) usando Automatas Finitos
// Deterministas (DFAs). Este archivo en particular se encarga de la
// lectura de archivos fuente, normalizando los finales de linea para
// garantizar un procesamiento uniforme independientemente del sistema
// operativo de origen del archivo.
package lexer

import (
	"fmt"
	"os"
	"strings"
)

// SourceFile almacena el contenido y los metadatos de un archivo fuente de entrada.
// Se usa como estructura principal para pasar el texto fuente al tokenizador.
//
// Campos:
//   - Path:    ruta en disco del archivo leido (para mensajes de error/diagnostico).
//   - Content: contenido completo del archivo como una sola cadena, ya normalizado.
//   - Lines:   el contenido dividido linea por linea (util para reportar errores con contexto).
type SourceFile struct {
	Path    string
	Content string
	Lines   []string
}

// ReadSource lee un archivo fuente desde disco y devuelve un puntero a SourceFile.
// Realiza la normalizacion de finales de linea para que todo el contenido
// use exclusivamente '\n' como separador, sin importar si el archivo original
// usaba '\r\n' (Windows) o '\r' (Mac clasico).
//
// Parametros:
//   - path: ruta del archivo fuente a leer.
//
// Retorna:
//   - *SourceFile: estructura con el contenido normalizado y las lineas separadas.
//   - error: si ocurre un problema al leer el archivo (por ejemplo, archivo no encontrado).
//
// Proceso:
//  1. Lee todo el archivo a memoria usando os.ReadFile.
//  2. Convierte los bytes a cadena (string).
//  3. Reemplaza todas las secuencias "\r\n" por "\n" (normalizacion Windows).
//  4. Reemplaza cualquier "\r" restante por "\n" (normalizacion Mac clasico).
//  5. Divide el contenido en lineas usando "\n" como delimitador.
//  6. Construye y retorna la estructura SourceFile.
func ReadSource(path string) (*SourceFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading source file %q: %w", path, err)
	}

	content := string(data)
	// Normalizar finales de linea estilo Windows (\r\n) a Unix (\n)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Normalizar retornos de carro solitarios (\r) a salto de linea (\n).
	// Esto cubre el formato Mac clasico (pre-OSX).
	content = strings.ReplaceAll(content, "\r", "\n")

	// Dividir el contenido en lineas individuales para facilitar
	// la referencia por numero de linea en mensajes de error.
	lines := strings.Split(content, "\n")

	return &SourceFile{
		Path:    path,
		Content: content,
		Lines:   lines,
	}, nil
}
