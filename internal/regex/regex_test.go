package regex

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// tokensEqual reports whether two token slices are identical.
func tokensEqual(a, b []RegexToken) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// dumpTokens returns a human-readable representation for test diffs.
func dumpTokens(tokens []RegexToken) string {
	var s string
	for i, t := range tokens {
		if i > 0 {
			s += " "
		}
		switch t.Kind {
		case TokAtom:
			s += fmt.Sprintf("Atom(%q)", t.Atom)
		case TokOp:
			if t.Op == ConcatOp {
				s += "Op(·)"
			} else {
				s += fmt.Sprintf("Op(%c)", t.Op)
			}
		case TokOpen:
			s += "Open"
		case TokClose:
			s += "Close"
		}
	}
	return s
}

// requireTokens is a test helper that fails with a readable diff.
func requireTokens(t *testing.T, got, want []RegexToken) {
	t.Helper()
	if !tokensEqual(got, want) {
		t.Errorf("token mismatch\n  got:  %s\n  want: %s", dumpTokens(got), dumpTokens(want))
	}
}

// hasAtom reports whether the token slice contains at least one atom with rune r.
func hasAtom(tokens []RegexToken, r rune) bool {
	for _, t := range tokens {
		if t.Kind == TokAtom && t.Atom == r {
			return true
		}
	}
	return false
}

// countKind counts tokens with the given kind.
func countKind(tokens []RegexToken, kind TokKind) int {
	n := 0
	for _, t := range tokens {
		if t.Kind == kind {
			n++
		}
	}
	return n
}

// countOp counts operator tokens with the given op rune.
func countOp(tokens []RegexToken, op rune) int {
	n := 0
	for _, t := range tokens {
		if t.Kind == TokOp && t.Op == op {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Normalizer tests
// ---------------------------------------------------------------------------

func TestNormalize_SimpleLiteral(t *testing.T) {
	tokens, err := Normalize("'a'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{atomTok('a')}
	requireTokens(t, tokens, want)
}

func TestNormalize_ConcatInsertion(t *testing.T) {
	// 'a' 'b' → atom(a) concat atom(b)
	tokens, err := Normalize("'a' 'b'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{atomTok('a'), opTok(ConcatOp), atomTok('b')}
	requireTokens(t, tokens, want)
}

func TestNormalize_AlternationNoConcatBetweenPipeAndOperands(t *testing.T) {
	// 'a' | 'b' → atom(a) | atom(b)   — NO concat around |
	tokens, err := Normalize("'a' | 'b'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{atomTok('a'), opTok('|'), atomTok('b')}
	requireTokens(t, tokens, want)
}

func TestNormalize_CharClassRange(t *testing.T) {
	// [a-c] → (a|b|c)
	tokens, err := Normalize("[a-c]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		openTok(),
		atomTok('a'), opTok('|'),
		atomTok('b'), opTok('|'),
		atomTok('c'),
		closeTok(),
	}
	requireTokens(t, tokens, want)
}

func TestNormalize_CharClassMultipleRanges(t *testing.T) {
	// [a-cA-C] → (a|b|c|A|B|C)
	tokens, err := Normalize("[a-cA-C]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		openTok(),
		atomTok('a'), opTok('|'),
		atomTok('b'), opTok('|'),
		atomTok('c'), opTok('|'),
		atomTok('A'), opTok('|'),
		atomTok('B'), opTok('|'),
		atomTok('C'),
		closeTok(),
	}
	requireTokens(t, tokens, want)
}

func TestNormalize_ComplementCharClass(t *testing.T) {
	// [^ 'a' 'b'] → alternation group of all alphabet chars EXCEPT a and b
	tokens, err := Normalize("[^ 'a' 'b']")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must NOT contain a or b atoms
	if hasAtom(tokens, 'a') {
		t.Error("complement class should NOT contain atom 'a'")
	}
	if hasAtom(tokens, 'b') {
		t.Error("complement class should NOT contain atom 'b'")
	}

	// Must contain other printable chars like 'c', 'z', '0'
	for _, r := range []rune{'c', 'z', '0', ' '} {
		if !hasAtom(tokens, r) {
			t.Errorf("complement class should contain atom %q", r)
		}
	}

	// Should be wrapped in a group
	if tokens[0].Kind != TokOpen || tokens[len(tokens)-1].Kind != TokClose {
		t.Error("complement class should be wrapped in ( ... )")
	}

	// Total alphabet = 2 (tab, cr) + 95 (32..126) = 97; minus 2 excluded = 95 atoms
	alphabet := buildAlphabet()
	expectedAtoms := len(alphabet) - 2
	gotAtoms := countKind(tokens, TokAtom)
	if gotAtoms != expectedAtoms {
		t.Errorf("expected %d atoms, got %d", expectedAtoms, gotAtoms)
	}
}

func TestNormalize_DoubleQuotedString(t *testing.T) {
	// "abc" → (a·b·c)
	tokens, err := Normalize(`"abc"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		openTok(),
		atomTok('a'), opTok(ConcatOp),
		atomTok('b'), opTok(ConcatOp),
		atomTok('c'),
		closeTok(),
	}
	requireTokens(t, tokens, want)
}

func TestNormalize_SingleCharDoubleQuoted(t *testing.T) {
	// "x" → single atom x, no grouping
	tokens, err := Normalize(`"x"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{atomTok('x')}
	requireTokens(t, tokens, want)
}

func TestNormalize_EmptyDoubleQuoted(t *testing.T) {
	// "" → no tokens
	tokens, err := Normalize(`""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected empty tokens, got %d: %s", len(tokens), dumpTokens(tokens))
	}
}

func TestNormalize_WildcardDot(t *testing.T) {
	// . → alternation group of entire alphabet (97 chars)
	tokens, err := Normalize(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alphabet := buildAlphabet()
	expectedAtoms := len(alphabet)
	gotAtoms := countKind(tokens, TokAtom)
	if gotAtoms != expectedAtoms {
		t.Errorf("dot wildcard: expected %d atoms, got %d", expectedAtoms, gotAtoms)
	}

	// Wrapped in group
	if tokens[0].Kind != TokOpen || tokens[len(tokens)-1].Kind != TokClose {
		t.Error("wildcard should be wrapped in ( ... )")
	}
}

func TestNormalize_WildcardUnderscore(t *testing.T) {
	// _ → same output as .
	dotToks, err := Normalize(".")
	if err != nil {
		t.Fatalf("unexpected error for dot: %v", err)
	}
	underToks, err := Normalize("_")
	if err != nil {
		t.Fatalf("unexpected error for underscore: %v", err)
	}
	if !tokensEqual(dotToks, underToks) {
		t.Errorf("_ and . should produce identical tokens\n  dot:   %s\n  under: %s",
			dumpTokens(dotToks), dumpTokens(underToks))
	}
}

func TestNormalize_EscapeSequences(t *testing.T) {
	tests := []struct {
		input string
		want  rune
	}{
		{`'\n'`, '\n'},
		{`'\t'`, '\t'},
		{`'\\'`, '\\'},
		{`'\''`, '\''},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			tokens, err := Normalize(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tokens) != 1 || tokens[0].Kind != TokAtom || tokens[0].Atom != tc.want {
				t.Errorf("expected single atom %q, got %s", tc.want, dumpTokens(tokens))
			}
		})
	}
}

func TestNormalize_KleeneStarNoConcatAfter(t *testing.T) {
	// 'a'* 'b' → a * · b
	tokens, err := Normalize("'a'* 'b'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'),
		opTok('*'),
		opTok(ConcatOp),
		atomTok('b'),
	}
	requireTokens(t, tokens, want)
}

func TestNormalize_PlusAndOptional(t *testing.T) {
	// 'a'+ 'b'? → a + · b ?
	tokens, err := Normalize("'a'+ 'b'?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'),
		opTok('+'),
		opTok(ConcatOp),
		atomTok('b'),
		opTok('?'),
	}
	requireTokens(t, tokens, want)
}

func TestNormalize_ComplexPattern(t *testing.T) {
	// [0-9]+ '.' [0-9]+ — FLOAT-like pattern
	// Expected structure:
	//   (0|1|...|9) + · atom('.') · (0|1|...|9) +
	tokens, err := Normalize("[0-9]+ '.' [0-9]+")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the overall structure by checking key tokens:
	// 1. starts with ( for first char class
	if tokens[0].Kind != TokOpen {
		t.Error("expected leading ( for char class")
	}

	// 2. there must be exactly 2 '+' operators (one for each [0-9]+)
	plusCount := countOp(tokens, '+')
	if plusCount != 2 {
		t.Errorf("expected 2 '+' ops, got %d", plusCount)
	}

	// 3. there must be a dot literal atom
	if !hasAtom(tokens, '.') {
		t.Error("expected literal '.' atom")
	}

	// 4. there must be concat operators between the three main parts
	concatCount := countOp(tokens, ConcatOp)
	if concatCount < 2 {
		t.Errorf("expected at least 2 concat ops, got %d", concatCount)
	}

	// 5. digits 0-9 must appear as atoms
	for r := '0'; r <= '9'; r++ {
		if !hasAtom(tokens, r) {
			t.Errorf("missing digit atom %q", r)
		}
	}
}

// ---------------------------------------------------------------------------
// Builder (BuildPostfix) tests
// ---------------------------------------------------------------------------

func TestBuildPostfix_SimpleConcat(t *testing.T) {
	// input: a · b → postfix: a b · then augmented with ( ... ) · #
	// BuildPostfix wraps as ( a · b ) · #
	// After shunting yard: a b · # ·
	input := []RegexToken{atomTok('a'), opTok(ConcatOp), atomTok('b')}
	got, err := BuildPostfix(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'), atomTok('b'), opTok(ConcatOp),
		atomTok(EndMarker), opTok(ConcatOp),
	}
	requireTokens(t, got, want)
}

func TestBuildPostfix_Alternation(t *testing.T) {
	// input: a | b → augmented: ( a | b ) · #
	// postfix: a b | # ·
	input := []RegexToken{atomTok('a'), opTok('|'), atomTok('b')}
	got, err := BuildPostfix(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'), atomTok('b'), opTok('|'),
		atomTok(EndMarker), opTok(ConcatOp),
	}
	requireTokens(t, got, want)
}

func TestBuildPostfix_StarPrecedence(t *testing.T) {
	// input: a | b * → augmented: ( a | b * ) · #
	// * is unary → emitted immediately after b
	// postfix: a b * | # ·
	input := []RegexToken{atomTok('a'), opTok('|'), atomTok('b'), opTok('*')}
	got, err := BuildPostfix(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'), atomTok('b'), opTok('*'), opTok('|'),
		atomTok(EndMarker), opTok(ConcatOp),
	}
	requireTokens(t, got, want)
}

func TestBuildPostfix_PlusPrecedence(t *testing.T) {
	// input: a · b + → augmented: ( a · b + ) · #
	// + is unary → emitted immediately after b
	// postfix: a b + · # ·
	input := []RegexToken{atomTok('a'), opTok(ConcatOp), atomTok('b'), opTok('+')}
	got, err := BuildPostfix(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'), atomTok('b'), opTok('+'), opTok(ConcatOp),
		atomTok(EndMarker), opTok(ConcatOp),
	}
	requireTokens(t, got, want)
}

func TestBuildPostfix_GroupedAlternation(t *testing.T) {
	// input: ( a | b ) · c → augmented: ( ( a | b ) · c ) · #
	// postfix: a b | c · # ·
	input := []RegexToken{
		openTok(), atomTok('a'), opTok('|'), atomTok('b'), closeTok(),
		opTok(ConcatOp), atomTok('c'),
	}
	got, err := BuildPostfix(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'), atomTok('b'), opTok('|'),
		atomTok('c'), opTok(ConcatOp),
		atomTok(EndMarker), opTok(ConcatOp),
	}
	requireTokens(t, got, want)
}

func TestBuildPostfix_UnmatchedOpenParen(t *testing.T) {
	// ( a with no closing )
	input := []RegexToken{openTok(), atomTok('a')}
	_, err := BuildPostfix(input)
	if err == nil {
		t.Fatal("expected error for unmatched open paren, got nil")
	}
}

func TestBuildPostfix_UnmatchedCloseParen(t *testing.T) {
	// a ) with no opening (
	input := []RegexToken{atomTok('a'), closeTok()}
	_, err := BuildPostfix(input)
	if err == nil {
		t.Fatal("expected error for unmatched close paren, got nil")
	}
}

func TestBuildPostfix_DoubleUnary(t *testing.T) {
	// input: a * + → augmented: ( a * + ) · #
	// Both unary → emitted immediately after their operand
	// postfix: a * + # ·
	input := []RegexToken{atomTok('a'), opTok('*'), opTok('+')}
	got, err := BuildPostfix(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []RegexToken{
		atomTok('a'), opTok('*'), opTok('+'),
		atomTok(EndMarker), opTok(ConcatOp),
	}
	requireTokens(t, got, want)
}

func TestBuildPostfix_EndMarkerPresent(t *testing.T) {
	// Every BuildPostfix result must end with atom(EndMarker) followed by Op(ConcatOp)
	inputs := []struct {
		name  string
		input []RegexToken
	}{
		{"single atom", []RegexToken{atomTok('x')}},
		{"alternation", []RegexToken{atomTok('a'), opTok('|'), atomTok('b')}},
		{"star", []RegexToken{atomTok('a'), opTok('*')}},
		{"grouped", []RegexToken{openTok(), atomTok('a'), closeTok()}},
	}
	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildPostfix(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			n := len(got)
			if n < 2 {
				t.Fatalf("output too short (%d tokens): %s", n, dumpTokens(got))
			}
			// Last token must be ConcatOp
			last := got[n-1]
			if last.Kind != TokOp || last.Op != ConcatOp {
				t.Errorf("last token should be ConcatOp, got %s", dumpTokens([]RegexToken{last}))
			}
			// Second-to-last must be EndMarker atom
			prev := got[n-2]
			if prev.Kind != TokAtom || prev.Atom != EndMarker {
				t.Errorf("second-to-last token should be EndMarker atom, got %s", dumpTokens([]RegexToken{prev}))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: Normalize → BuildPostfix round-trip
// ---------------------------------------------------------------------------

func TestIntegration_NormalizeThenBuild(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{"simple literal", "'a'", false},
		{"concat", "'a' 'b' 'c'", false},
		{"alternation", "'a' | 'b'", false},
		{"kleene star", "'a'*", false},
		{"plus", "'a'+", false},
		{"optional", "'a'?", false},
		{"char class", "[a-z]", false},
		{"double quoted", `"hello"`, false},
		{"float pattern", "[0-9]+ '.' [0-9]+", false},
		{"complex", "('a' | 'b')* 'c'+", false},
		{"wildcard dot", ".", false},
		{"wildcard underscore", "_", false},
		{"complement class", "[^ 'a']", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			normalized, err := Normalize(tc.pattern)
			if err != nil {
				if !tc.wantErr {
					t.Fatalf("Normalize error: %v", err)
				}
				return
			}
			postfix, err := BuildPostfix(normalized)
			if err != nil {
				if !tc.wantErr {
					t.Fatalf("BuildPostfix error: %v", err)
				}
				return
			}
			if tc.wantErr {
				t.Fatal("expected error, got none")
			}

			// Every successful result must end with EndMarker · ConcatOp
			n := len(postfix)
			if n < 2 {
				t.Fatalf("postfix too short: %d", n)
			}
			if postfix[n-1].Kind != TokOp || postfix[n-1].Op != ConcatOp {
				t.Errorf("last token not ConcatOp")
			}
			if postfix[n-2].Kind != TokAtom || postfix[n-2].Atom != EndMarker {
				t.Errorf("second-to-last not EndMarker")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Edge cases / error paths
// ---------------------------------------------------------------------------

func TestNormalize_UnclosedSingleQuote(t *testing.T) {
	_, err := Normalize("'a")
	if err == nil {
		t.Fatal("expected error for unclosed single quote")
	}
}

func TestNormalize_UnclosedDoubleQuote(t *testing.T) {
	_, err := Normalize(`"abc`)
	if err == nil {
		t.Fatal("expected error for unclosed double quote")
	}
}

func TestNormalize_EmptyCharClass(t *testing.T) {
	_, err := Normalize("[]")
	if err == nil {
		t.Fatal("expected error for empty char class")
	}
}

func TestBuildAlphabet_Size(t *testing.T) {
	alpha := buildAlphabet()
	// \t (9), \r (13), then 32..126 = 95 chars → total 97
	if len(alpha) != 97 {
		t.Errorf("expected 97 chars in alphabet, got %d", len(alpha))
	}
	// First two are \t and \r
	if alpha[0] != '\t' {
		t.Errorf("expected first char \\t, got %q", alpha[0])
	}
	if alpha[1] != '\r' {
		t.Errorf("expected second char \\r, got %q", alpha[1])
	}
}

func TestTokensToString(t *testing.T) {
	tokens := []RegexToken{
		atomTok('a'), opTok(ConcatOp), atomTok('b'), opTok('|'), atomTok('c'),
	}
	s := TokensToString(tokens)
	if len(s) == 0 {
		t.Error("TokensToString returned empty string")
	}
	// Should contain the concat symbol ·
	if !containsRune(s, '·') {
		t.Errorf("expected · in output, got %q", s)
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
