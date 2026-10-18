// Intentional semantic error: a foreach-body binding escapes its scope.
class Accumulator {
  let total: integer = 0;
  function add(value: integer): integer {
    this.total = this.total + value;
    return this.total;
  }
}

let accumulator = new Accumulator();
function run(): integer {
  let sum = 0;
  foreach (value in [1, 2, 3]) {
    let hidden = value * 2;
    sum = sum + accumulator.add(hidden);
  }
  return sum + hidden;
}

let result = run();
print(result);
