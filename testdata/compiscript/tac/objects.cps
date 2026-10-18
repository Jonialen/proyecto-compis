// All fields initialize before the nearest constructor, including virtual calls.
class Base {
  let first: integer = 1;
  function constructor() { this.set(); }
  function set() {}
}
class Child : Base {
  let later: integer = 7;
  function set() { this.later = 9; }
}
let child = new Child();
let observed = child.later;

function zero(): integer { return 0; }
let saved = zero;
class Counter {
  let value: integer;
  function constructor(seed: integer) { this.value = seed; }
  function next(): integer { value = value + 1; return value; }
  function install() {
    function next(): integer { return this.next(); }
    saved = next;
  }
  function name(): string { return "counter"; }
}
function install() {
  let counter = new Counter(10);
  counter.install();
}
install();
let first = saved();
let second = saved();
let separate = new Counter(20);
let bound = separate.next;
let independent = bound();
let alias = separate;
alias.value = 30;
let shared = separate.value;
let label = separate.name() + ":ready";
