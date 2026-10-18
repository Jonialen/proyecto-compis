// Freeze receivers/indices and earlier operands before later mutation effects.
let trace = 0;
function mark(n: integer): integer {
  trace = trace * 10 + n;
  return n;
}

class Cell {
  let value;
  function constructor(value) { this.value = value; }
}

let a = new Cell(10);
let b = new Cell(20);
let chosen = a;
function select(): Cell {
  mark(1);
  return chosen;
}
function rhs(): integer {
  mark(2);
  chosen = b;
  return 7;
}

select().value = rhs();

let index = 0;
let items = [1, 2];
function move(): integer {
  mark(3);
  index = 1;
  return 9;
}
items[index] = move();

let selected = true ? mark(4) : mark(9);
let x = 1;
let y = x + (x = 5);
let sum = items[0] + items[1] + y;

// Unannotated fields retain their unknown contract instead of their first tag.
a.value = "changed";
let finalText = a.value;
print(trace);
print(b.value);
print(sum);
print(finalText);
