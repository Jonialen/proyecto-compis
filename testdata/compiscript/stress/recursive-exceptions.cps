// Untyped recursion, guarded division and faults raised inside catch bodies.
let calls = 0;
function fold(n): integer {
  calls = calls + 1;
  if (n == 0) { return 1; }
  return n * fold(n - 1);
}

function divide(value, denominator): float {
  return value / denominator;
}

function read(values, index): integer { return values[index]; }

let denominators = [4, 0, 2, 0, 5];
let total: float = 0;
let caught = 0;
let steps = 0;
foreach (denominator in denominators) {
  try {
    steps = steps + 1;
    let part = divide(20, denominator);
    total = total + part;
  } catch (error) {
    caught = caught + 1;
  }
}

let nested = 0;
try {
  try {
    let empty: integer[] = [];
    print(read(empty, 1));
  } catch (inner) {
    nested = nested + 1;
    // The inner handler must be gone before this second fault occurs.
    let zero = 0;
    let impossible = 1 / zero;
  }
} catch (outer) {
  nested = nested + 10;
}

let factorial = fold(6);
let result = factorial + caught + nested + steps;
print(total);
print(caught);
print(nested);
print(factorial);
print(calls);
print(result);
