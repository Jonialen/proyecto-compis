// Ragged lists, shared mutation and transfers across nested control structures.
let matrix: integer[][] = [[1, 2, 3], [], [4, 5], [6]];
let alias = matrix;
alias[0][1] = 20;

let sum = 0;
let visited = 0;
foreach (row in matrix) {
  foreach (value in row) {
    visited = visited + 1;
    if (value % 2 == 0) { continue; }
    sum = sum + value;
  }
}

let trace = 0;
for (let i = 0; i < 5; i = i + 1) {
  switch (i % 2 == 0) {
    case true:
      // This break exits the switch, not the surrounding for loop.
      if (i == 4) { break; }
      trace = trace + i;
    case false:
      // Continue targets the for post, crossing the switch.
      continue;
  }
  trace = trace + 10;
}

let rounds = 0;
do {
  rounds = rounds + 1;
  sum = sum + rounds;
} while (rounds < 3);

let counter = 3;
while (counter > 0) {
  sum = sum + counter;
  counter = counter - 1;
}

let result = sum + trace + visited;
print(visited);
print(matrix[0][1]);
print(sum);
print(trace);
print(result);
