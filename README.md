# YALex Lexer Generator

This project is a powerful and efficient **Lexer Generator** written in Go. It translates YALex (`.yal`) specification files into standalone, optimized Go source code for lexical analysis.

## Key Features

*   **Lexer Generation:** Generates standalone `.go` files containing a complete table-driven lexical analyzer.
*   **YALex Compatible:** Full support for the YALex specification, including macros, rules, and priority-based disambiguation.
*   **Direct DFA Construction:** Builds Deterministic Finite Automata (DFA) directly from syntax trees (Aho-Sethi-Ullman algorithm).
*   **DFA Minimization:** Implements state minimization (Table-Filling algorithm) to produce the most compact lexers possible.
*   **Maximal Munch:** The generated lexers always identify the longest possible match from the current position.
*   **Visualization:** Can generate Graphviz DOT files to visualize the syntax trees of your regular expressions.

## How to Use

The tool can be used either to tokenize a file directly (engine mode) or to generate a standalone lexer (generator mode).

### 1. Generator Mode (Recommended)

To generate a standalone lexer source file from a `.yal` specification:

```bash
go run main.go -yal <path_to_yal_file> -out <output_lexer.go>
```

Then, you can compile and run your generated lexer:

```bash
go run <output_lexer.go> -src <path_to_input_file>
```

### 2. Engine Mode

To tokenize an input file directly without generating intermediate source code:

```bash
go run main.go -yal <path_to_yal_file> -src <path_to_input_file>
```

### 3. Visualization Mode

To generate syntax tree diagrams:

```bash
go run main.go -yal <path_to_yal_file> -tree
```

## Project Structure

*   `main.go`: Tool entry point (Generator/Engine CLI).
*   `internal/yalex`: YALex parser and macro expander.
*   `internal/regex`: Regex tokenizer and postfix converter.
*   `internal/dfa`: DFA construction, minimization, and tree building.
*   `internal/generator`: Source code generation templates.
*   `internal/lexer`: Core simulation logic and input reading.

## Compliance

This implementation strictly follows the requirements set in the project instructions:
1.  **Input:** Reads `.yal` lexer specifications.
2.  **Output:** Generates a **source program** implementing the lexer and **visualized syntax trees**.
3.  **Functionality:** Recognizes tokens based on regular definitions and detects lexical errors.
