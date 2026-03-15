package dfa

import (
	"testing"

	"genanalex/internal/regex"
)

// ---------------------------------------------------------------------------
// Helper: build a syntax tree from a .yal-style pattern string
// ---------------------------------------------------------------------------

// buildTreeFromPattern normalizes, converts to postfix, and builds a tree.
// Returns the root node and posToSymbol map. Fails the test on error.
func buildTreeFromPattern(t *testing.T, pattern string) (*Node, map[int]rune) {
	t.Helper()
	norm, err := regex.Normalize(pattern)
	if err != nil {
		t.Fatalf("Normalize(%q) failed: %v", pattern, err)
	}
	postfix, err := regex.BuildPostfix(norm)
	if err != nil {
		t.Fatalf("BuildPostfix(%q) failed: %v", pattern, err)
	}
	root, posToSymbol, err := BuildTree(postfix)
	if err != nil {
		t.Fatalf("BuildTree(%q) failed: %v", pattern, err)
	}
	return root, posToSymbol
}

// buildDFAFromPattern runs the full pipeline: pattern → DFA.
func buildDFAFromPattern(t *testing.T, pattern, tokenName string) *DFA {
	t.Helper()
	root, posToSymbol := buildTreeFromPattern(t, pattern)
	return BuildDFA(root, posToSymbol, tokenName)
}

// accepts simulates a DFA on the input string and returns true if it ends in
// an accepting state.
func accepts(d *DFA, input string) bool {
	state := d.Start
	for _, ch := range input {
		trans, ok := d.Transitions[state]
		if !ok {
			return false
		}
		next, ok := trans[ch]
		if !ok {
			return false
		}
		state = next
	}
	return d.Accepting[state]
}

// setsEqual returns true if two map[int]bool sets contain the same keys.
func setsEqual(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// setContains returns true if set contains all positions in want.
func setContains(set map[int]bool, want ...int) bool {
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Manually construct tree nodes for unit tests on positions/followpos
// ---------------------------------------------------------------------------

func makeLeaf(sym rune, pos int) *Node {
	return &Node{Kind: NodeLeaf, Symbol: sym, Pos: pos}
}

func makeCat(left, right *Node) *Node {
	return &Node{Kind: NodeCat, Left: left, Right: right}
}

func makeOr(left, right *Node) *Node {
	return &Node{Kind: NodeOr, Left: left, Right: right}
}

func makeStar(child *Node) *Node {
	return &Node{Kind: NodeStar, Left: child}
}

func makePlus(child *Node) *Node {
	return &Node{Kind: NodePlus, Left: child}
}

func makeOpt(child *Node) *Node {
	return &Node{Kind: NodeOpt, Left: child}
}

// ===========================
// Tests for tree.go
// ===========================

func TestBuildTree_SingleChar(t *testing.T) {
	// Pattern 'a' augmented → cat(leaf(a,1), leaf(#,2))
	root, pos := buildTreeFromPattern(t, "'a'")

	if root.Kind != NodeCat {
		t.Fatalf("expected root kind NodeCat, got %d", root.Kind)
	}
	if root.Left == nil || root.Left.Kind != NodeLeaf {
		t.Fatal("expected left child to be a leaf")
	}
	if root.Left.Symbol != 'a' {
		t.Errorf("expected left leaf symbol 'a', got %q", root.Left.Symbol)
	}
	if root.Right == nil || root.Right.Kind != NodeLeaf {
		t.Fatal("expected right child to be end-marker leaf")
	}
	if root.Right.Symbol != regex.EndMarker {
		t.Errorf("expected right leaf to be EndMarker, got %q", root.Right.Symbol)
	}
	// posToSymbol should have 2 positions
	if len(pos) != 2 {
		t.Errorf("expected 2 positions, got %d", len(pos))
	}
}

func TestBuildTree_Alternation(t *testing.T) {
	// 'a'|'b' augmented → cat(or(leaf(a), leaf(b)), leaf(#))
	root, _ := buildTreeFromPattern(t, "'a'|'b'")

	if root.Kind != NodeCat {
		t.Fatalf("expected root kind NodeCat, got %d", root.Kind)
	}
	if root.Left == nil || root.Left.Kind != NodeOr {
		t.Fatal("expected left child to be an or node")
	}
	orNode := root.Left
	if orNode.Left == nil || orNode.Left.Kind != NodeLeaf || orNode.Left.Symbol != 'a' {
		t.Error("expected or left child = leaf(a)")
	}
	if orNode.Right == nil || orNode.Right.Kind != NodeLeaf || orNode.Right.Symbol != 'b' {
		t.Error("expected or right child = leaf(b)")
	}
}

func TestBuildTree_Concatenation(t *testing.T) {
	// 'a''b' augmented → cat(cat(leaf(a), leaf(b)), leaf(#))
	root, _ := buildTreeFromPattern(t, "'a''b'")

	if root.Kind != NodeCat {
		t.Fatalf("expected root kind NodeCat (augmented), got %d", root.Kind)
	}
	if root.Left == nil || root.Left.Kind != NodeCat {
		t.Fatal("expected left child to be a cat node")
	}
	catNode := root.Left
	if catNode.Left == nil || catNode.Left.Symbol != 'a' {
		t.Error("expected inner cat left = leaf(a)")
	}
	if catNode.Right == nil || catNode.Right.Symbol != 'b' {
		t.Error("expected inner cat right = leaf(b)")
	}
}

func TestBuildTree_KleeneStar(t *testing.T) {
	// 'a'* augmented → cat(star(leaf(a)), leaf(#))
	root, _ := buildTreeFromPattern(t, "'a'*")

	if root.Kind != NodeCat {
		t.Fatalf("expected root kind NodeCat, got %d", root.Kind)
	}
	if root.Left == nil || root.Left.Kind != NodeStar {
		t.Fatal("expected left child to be a star node")
	}
	if root.Left.Left == nil || root.Left.Left.Symbol != 'a' {
		t.Error("expected star child = leaf(a)")
	}
}

func TestBuildTree_Plus(t *testing.T) {
	// 'a'+ augmented → cat(plus(leaf(a)), leaf(#))
	root, _ := buildTreeFromPattern(t, "'a'+")

	if root.Kind != NodeCat {
		t.Fatalf("expected root kind NodeCat, got %d", root.Kind)
	}
	if root.Left == nil || root.Left.Kind != NodePlus {
		t.Fatal("expected left child to be a plus node")
	}
	if root.Left.Left == nil || root.Left.Left.Symbol != 'a' {
		t.Error("expected plus child = leaf(a)")
	}
}

func TestBuildTree_Optional(t *testing.T) {
	// 'a'? augmented → cat(opt(leaf(a)), leaf(#))
	root, _ := buildTreeFromPattern(t, "'a'?")

	if root.Kind != NodeCat {
		t.Fatalf("expected root kind NodeCat, got %d", root.Kind)
	}
	if root.Left == nil || root.Left.Kind != NodeOpt {
		t.Fatal("expected left child to be an opt node")
	}
	if root.Left.Left == nil || root.Left.Left.Symbol != 'a' {
		t.Error("expected opt child = leaf(a)")
	}
}

// ===========================
// Tests for positions.go (Nullable, FirstPos, LastPos)
// ===========================

func TestNullable_Leaf(t *testing.T) {
	n := makeLeaf('a', 1)
	if Nullable(n) {
		t.Error("leaf should not be nullable")
	}
}

func TestNullable_Star(t *testing.T) {
	n := makeStar(makeLeaf('a', 1))
	if !Nullable(n) {
		t.Error("star should be nullable")
	}
}

func TestNullable_Plus_NonNullableChild(t *testing.T) {
	// plus(leaf) → nullable(leaf) = false → nullable(plus) = false
	n := makePlus(makeLeaf('a', 1))
	if Nullable(n) {
		t.Error("plus(leaf) should not be nullable when child is not nullable")
	}
}

func TestNullable_Plus_NullableChild(t *testing.T) {
	// plus(opt(leaf)) → nullable(opt) = true → nullable(plus) = true
	n := makePlus(makeOpt(makeLeaf('a', 1)))
	if !Nullable(n) {
		t.Error("plus(opt(leaf)) should be nullable since opt is nullable")
	}
}

func TestNullable_Opt(t *testing.T) {
	n := makeOpt(makeLeaf('a', 1))
	if !Nullable(n) {
		t.Error("opt should be nullable")
	}
}

func TestNullable_Or(t *testing.T) {
	// or(leaf, leaf) → false || false → false
	n1 := makeOr(makeLeaf('a', 1), makeLeaf('b', 2))
	if Nullable(n1) {
		t.Error("or(leaf, leaf) should not be nullable")
	}
	// or(leaf, star(leaf)) → false || true → true
	n2 := makeOr(makeLeaf('a', 1), makeStar(makeLeaf('b', 2)))
	if !Nullable(n2) {
		t.Error("or(leaf, star(leaf)) should be nullable")
	}
}

func TestNullable_Cat(t *testing.T) {
	// cat(leaf, leaf) → false && false → false
	n1 := makeCat(makeLeaf('a', 1), makeLeaf('b', 2))
	if Nullable(n1) {
		t.Error("cat(leaf, leaf) should not be nullable")
	}
	// cat(star(leaf), star(leaf)) → true && true → true
	n2 := makeCat(makeStar(makeLeaf('a', 1)), makeStar(makeLeaf('b', 2)))
	if !Nullable(n2) {
		t.Error("cat(star, star) should be nullable")
	}
}

func TestNullable_NilNode(t *testing.T) {
	if Nullable(nil) {
		t.Error("nil node should not be nullable")
	}
}

func TestFirstPos_Leaf(t *testing.T) {
	n := makeLeaf('a', 1)
	fp := FirstPos(n)
	expected := map[int]bool{1: true}
	if !setsEqual(fp, expected) {
		t.Errorf("FirstPos(leaf(1)) = %v, want {1}", fp)
	}
}

func TestFirstPos_Or(t *testing.T) {
	n := makeOr(makeLeaf('a', 1), makeLeaf('b', 2))
	fp := FirstPos(n)
	expected := map[int]bool{1: true, 2: true}
	if !setsEqual(fp, expected) {
		t.Errorf("FirstPos(or(1,2)) = %v, want {1,2}", fp)
	}
}

func TestFirstPos_Cat_NonNullableLeft(t *testing.T) {
	// cat(leaf(1), leaf(2)) → left not nullable → {1}
	n := makeCat(makeLeaf('a', 1), makeLeaf('b', 2))
	fp := FirstPos(n)
	expected := map[int]bool{1: true}
	if !setsEqual(fp, expected) {
		t.Errorf("FirstPos(cat(leaf,leaf)) = %v, want {1}", fp)
	}
}

func TestFirstPos_Cat_NullableLeft(t *testing.T) {
	// cat(star(leaf(1)), leaf(2)) → left nullable → {1, 2}
	n := makeCat(makeStar(makeLeaf('a', 1)), makeLeaf('b', 2))
	fp := FirstPos(n)
	expected := map[int]bool{1: true, 2: true}
	if !setsEqual(fp, expected) {
		t.Errorf("FirstPos(cat(star(leaf),leaf)) = %v, want {1,2}", fp)
	}
}

func TestLastPos_Leaf(t *testing.T) {
	n := makeLeaf('a', 1)
	lp := LastPos(n)
	expected := map[int]bool{1: true}
	if !setsEqual(lp, expected) {
		t.Errorf("LastPos(leaf(1)) = %v, want {1}", lp)
	}
}

func TestLastPos_Cat_NonNullableRight(t *testing.T) {
	// cat(leaf(1), leaf(2)) → right not nullable → {2}
	n := makeCat(makeLeaf('a', 1), makeLeaf('b', 2))
	lp := LastPos(n)
	expected := map[int]bool{2: true}
	if !setsEqual(lp, expected) {
		t.Errorf("LastPos(cat(leaf,leaf)) = %v, want {2}", lp)
	}
}

func TestLastPos_Cat_NullableRight(t *testing.T) {
	// cat(leaf(1), star(leaf(2))) → right nullable → {1, 2}
	n := makeCat(makeLeaf('a', 1), makeStar(makeLeaf('b', 2)))
	lp := LastPos(n)
	expected := map[int]bool{1: true, 2: true}
	if !setsEqual(lp, expected) {
		t.Errorf("LastPos(cat(leaf,star(leaf))) = %v, want {1,2}", lp)
	}
}

// ===========================
// Tests for followpos.go
// ===========================

func TestFollowPos_SimpleConcat(t *testing.T) {
	// 'a''b' augmented: tree is cat(cat(leaf(a,1), leaf(b,2)), leaf(#,3))
	// followpos(1) should contain 2 (because cat(a,b) → lastpos(a)={1}, firstpos(b)={2})
	root, _ := buildTreeFromPattern(t, "'a''b'")
	fp := ComputeFollowPos(root)

	// Find position of 'a' — it's position 1
	if !fp[1][2] {
		t.Errorf("followpos(pos_a=1) should contain pos_b=2, got %v", fp[1])
	}
}

func TestFollowPos_Star(t *testing.T) {
	// 'a'* augmented: tree is cat(star(leaf(a,1)), leaf(#,2))
	// followpos(1) should contain 1 (star loop)
	root, _ := buildTreeFromPattern(t, "'a'*")
	fp := ComputeFollowPos(root)

	if !fp[1][1] {
		t.Errorf("followpos(pos_a=1) should contain 1 (star loop), got %v", fp[1])
	}
}

func TestFollowPos_Plus(t *testing.T) {
	// 'a'+ augmented: tree is cat(plus(leaf(a,1)), leaf(#,2))
	// followpos(1) should contain 1 (plus loop, same as star)
	root, _ := buildTreeFromPattern(t, "'a'+")
	fp := ComputeFollowPos(root)

	if !fp[1][1] {
		t.Errorf("followpos(pos_a=1) should contain 1 (plus loop), got %v", fp[1])
	}
}

func TestFollowPos_Opt_NoLoop(t *testing.T) {
	// 'a'? augmented: tree is cat(opt(leaf(a,1)), leaf(#,2))
	// followpos(1) should NOT contain 1 (opt does not create a loop)
	root, _ := buildTreeFromPattern(t, "'a'?")
	fp := ComputeFollowPos(root)

	if fp[1] != nil && fp[1][1] {
		t.Errorf("followpos(pos_a=1) should NOT contain 1 for optional, got %v", fp[1])
	}
}

// ===========================
// Tests for builder.go (BuildDFA)
// ===========================

func TestBuildDFA_SingleChar(t *testing.T) {
	// 'a' → DFA: start --a--> accept (2 states)
	d := buildDFAFromPattern(t, "'a'", "CHAR_A")

	if len(d.States) != 2 {
		t.Errorf("expected 2 states, got %d", len(d.States))
	}
	if len(d.Accepting) != 1 {
		t.Errorf("expected 1 accepting state, got %d", len(d.Accepting))
	}
	// Start state should not be accepting
	if d.Accepting[d.Start] {
		t.Error("start state should not be accepting for 'a'")
	}
}

func TestBuildDFA_Alternation(t *testing.T) {
	// 'a'|'b' → start --a--> accept, start --b--> accept
	d := buildDFAFromPattern(t, "'a'|'b'", "AB")

	// Should have transitions from start for both 'a' and 'b'
	trans := d.Transitions[d.Start]
	if trans == nil {
		t.Fatal("start state should have transitions")
	}
	if _, ok := trans['a']; !ok {
		t.Error("start should have transition on 'a'")
	}
	if _, ok := trans['b']; !ok {
		t.Error("start should have transition on 'b'")
	}
	// Both targets should be accepting
	if !d.Accepting[trans['a']] {
		t.Error("state after 'a' should be accepting")
	}
	if !d.Accepting[trans['b']] {
		t.Error("state after 'b' should be accepting")
	}
}

func TestBuildDFA_KleeneStar(t *testing.T) {
	// 'a'* → start is accepting (matches empty); start --a--> state that accepts and loops
	d := buildDFAFromPattern(t, "'a'*", "STAR_A")

	// Start state should be accepting (a* matches "")
	if !d.Accepting[d.Start] {
		t.Error("start state should be accepting for 'a'*")
	}
}

func TestBuildDFA_Digits(t *testing.T) {
	// [0-9]+ → 2 states, transitions for '0'-'9', state B is accept and loops
	d := buildDFAFromPattern(t, "[0-9]+", "DIGITS")

	// Start should not be accepting (requires at least one digit)
	if d.Accepting[d.Start] {
		t.Error("start state should NOT be accepting for [0-9]+")
	}

	// Start should have transitions for digits 0-9
	trans := d.Transitions[d.Start]
	for ch := '0'; ch <= '9'; ch++ {
		if _, ok := trans[ch]; !ok {
			t.Errorf("start should have transition on %q", ch)
		}
	}

	// All digit transitions from start should go to the same accepting state
	targetState := trans['0']
	if !d.Accepting[targetState] {
		t.Error("target state for digit should be accepting")
	}
	for ch := '1'; ch <= '9'; ch++ {
		if trans[ch] != targetState {
			t.Errorf("expected all digits to go to same state, but '0' → %d and %q → %d",
				targetState, ch, trans[ch])
		}
	}

	// The accepting state should loop back to itself on digits
	acceptTrans := d.Transitions[targetState]
	for ch := '0'; ch <= '9'; ch++ {
		if acceptTrans[ch] != targetState {
			t.Errorf("accepting state should loop on %q", ch)
		}
	}
}

func TestBuildDFA_Concat(t *testing.T) {
	// 'a''b' → 3 states: start --a--> mid --b--> accept
	d := buildDFAFromPattern(t, "'a''b'", "AB")

	if len(d.States) != 3 {
		t.Errorf("expected 3 states, got %d", len(d.States))
	}

	// start --a--> mid
	mid, ok := d.Transitions[d.Start]['a']
	if !ok {
		t.Fatal("start should have transition on 'a'")
	}
	if d.Accepting[mid] {
		t.Error("mid state should not be accepting")
	}

	// mid --b--> accept
	acc, ok := d.Transitions[mid]['b']
	if !ok {
		t.Fatal("mid should have transition on 'b'")
	}
	if !d.Accepting[acc] {
		t.Error("final state should be accepting")
	}
}

func TestBuildDFA_BooleanTF(t *testing.T) {
	// 'T'|'F' → match exactly: 2 states
	d := buildDFAFromPattern(t, "'T'|'F'", "BOOLEAN")

	if len(d.States) != 2 {
		t.Errorf("expected 2 states for 'T'|'F', got %d", len(d.States))
	}

	// Start has transitions for T and F
	if _, ok := d.Transitions[d.Start]['T']; !ok {
		t.Error("start should have transition on 'T'")
	}
	if _, ok := d.Transitions[d.Start]['F']; !ok {
		t.Error("start should have transition on 'F'")
	}
}

// ===========================
// Tests for minimizer.go
// ===========================

func TestMinimize_AlreadyMinimal(t *testing.T) {
	// Build a 2-state DFA that's already minimal (start, accept)
	d := buildDFAFromPattern(t, "'a'", "A")
	before := len(d.States)

	m := Minimize(d)
	if len(m.States) != before {
		t.Errorf("minimized states = %d, expected %d (already minimal)", len(m.States), before)
	}
}

func TestMinimize_RedundantStates(t *testing.T) {
	// Construct a DFA with two equivalent non-accepting states that have
	// identical transitions → they should get merged.
	//
	// State 0 (start) --a--> 1, --b--> 2
	// State 1 --c--> 3 (accept)
	// State 2 --c--> 3 (accept)
	// State 3 (accept)
	//
	// States 1 and 2 are equivalent (same transitions, same accepting status).
	d := &DFA{
		States: map[int]map[int]bool{
			0: {0: true},
			1: {1: true},
			2: {2: true},
			3: {3: true},
		},
		Transitions: map[int]map[rune]int{
			0: {'a': 1, 'b': 2},
			1: {'c': 3},
			2: {'c': 3},
		},
		Start:     0,
		Accepting: map[int]bool{3: true},
		StateToken: map[int]string{
			3: "TOK",
		},
	}

	m := Minimize(d)
	// States 1 and 2 should be merged → 3 states total
	if len(m.States) != 3 {
		t.Errorf("expected 3 states after minimization, got %d", len(m.States))
	}
}

func TestMinimize_PreservesAccepting(t *testing.T) {
	d := buildDFAFromPattern(t, "[0-9]+", "DIGITS")
	m := Minimize(d)

	// The minimized DFA should still accept "42"
	if !accepts(m, "42") {
		t.Error("minimized DFA should still accept '42'")
	}
	// And reject ""
	if accepts(m, "") {
		t.Error("minimized DFA should reject empty string for [0-9]+")
	}
}

func TestMinimize_EmptyDFA(t *testing.T) {
	// 0-state DFA: should not crash
	d := &DFA{
		States:      map[int]map[int]bool{},
		Transitions: map[int]map[rune]int{},
		Start:       0,
		Accepting:   map[int]bool{},
		StateToken:  map[int]string{},
	}
	m := Minimize(d) // should not panic
	if m == nil {
		t.Error("Minimize should return non-nil DFA")
	}

	// 1-state DFA: should not crash
	d1 := &DFA{
		States:      map[int]map[int]bool{0: {1: true}},
		Transitions: map[int]map[rune]int{},
		Start:       0,
		Accepting:   map[int]bool{0: true},
		StateToken:  map[int]string{0: "TOK"},
	}
	m1 := Minimize(d1)
	if len(m1.States) != 1 {
		t.Errorf("expected 1 state, got %d", len(m1.States))
	}
	if !m1.Accepting[m1.Start] {
		t.Error("single accepting state should remain accepting")
	}
}

// ===========================
// DFA acceptance tests (full pipeline)
// ===========================

func TestDFA_AcceptsCorrectStrings(t *testing.T) {
	// [0-9]+
	d := buildDFAFromPattern(t, "[0-9]+", "NUMBER")

	cases := []struct {
		input string
		want  bool
	}{
		{"42", true},
		{"0", true},
		{"007", true},
		{"", false},
		{"abc", false},
		{"1a", false},
	}
	for _, tc := range cases {
		got := accepts(d, tc.input)
		if got != tc.want {
			t.Errorf("accepts(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDFA_BooleanAccepts(t *testing.T) {
	// 'T'|'F'
	d := buildDFAFromPattern(t, "'T'|'F'", "BOOLEAN")

	cases := []struct {
		input string
		want  bool
	}{
		{"T", true},
		{"F", true},
		{"TF", false},
		{"X", false},
		{"", false},
	}
	for _, tc := range cases {
		got := accepts(d, tc.input)
		if got != tc.want {
			t.Errorf("accepts(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDFA_NilAccepts(t *testing.T) {
	// "Nil"
	d := buildDFAFromPattern(t, `"Nil"`, "NIL")

	cases := []struct {
		input string
		want  bool
	}{
		{"Nil", true},
		{"nil", false},
		{"N", false},
		{"Ni", false},
		{"Nill", false},
	}
	for _, tc := range cases {
		got := accepts(d, tc.input)
		if got != tc.want {
			t.Errorf("accepts(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDFA_OptionalAccepts(t *testing.T) {
	// 'a'?
	d := buildDFAFromPattern(t, "'a'?", "OPT_A")

	cases := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"a", true},
		{"aa", false},
	}
	for _, tc := range cases {
		got := accepts(d, tc.input)
		if got != tc.want {
			t.Errorf("accepts(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDFA_StarAccepts(t *testing.T) {
	// 'a'*
	d := buildDFAFromPattern(t, "'a'*", "STAR_A")

	cases := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"a", true},
		{"aaa", true},
		{"b", false},
	}
	for _, tc := range cases {
		got := accepts(d, tc.input)
		if got != tc.want {
			t.Errorf("accepts(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
