let numbers = [1, 2, 3];
let ragged: integer[][] = [[1, 2], [3], []];
let empty: integer[] = [];
let nestedEmpty = [[], [1]];
let trailingEmpty = [[1], []];
let first: integer = numbers[0];
let row: integer[] = ragged[1];
let value: integer = ragged[0][1];
numbers[1] = 4;
ragged[1] = [5, 6];
let uncertain: integer = numbers[first];
function take(values: integer[]): integer { return values[999]; }
let contextualCall = take([]);
let original = [[1]];
let alias = original;
alias[0] = [1, 2];
let uncertainAlias = original[0][1];
