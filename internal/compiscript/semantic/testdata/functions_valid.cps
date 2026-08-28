function entry(n: integer): integer { return later(n); }
function later(n: integer): integer { if (n <= 1) { return 1; } else { return n * later(n - 1); } }
function outer(seed: integer): integer {
  function callLater(value: integer): integer { return later(value) + seed; }
  function later(seed: integer): integer { return seed; }
  return callLater(seed);
}
let widened: float = entry(3);
let nested: integer = outer(2);
function foreverWhile(): integer { while (true) { return 1; } }
function foreverFor(): integer { for (;;) { return 1; } }
