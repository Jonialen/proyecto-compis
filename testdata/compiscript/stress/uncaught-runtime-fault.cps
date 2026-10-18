// Compilation succeeds; execution must stop at the second guarded division.
let attempts = 0;
function divide(denominator): float {
  attempts = attempts + 1;
  return 100 / denominator;
}

let denominators = [2, 0, 5];
let total: float = 0;
foreach (denominator in denominators) {
  let part = divide(denominator);
  total = total + part;
  print(total);
}

// Unreachable after the fault: the third denominator must never be consumed.
print("unexpected completion");
