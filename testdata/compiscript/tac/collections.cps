let rows: integer[][] = [[1, 2], [], [3]];
let alias = rows;
alias[0][0] = 9;
let shared = rows[0][0];
let sum = 0;
foreach (row in rows) {
    foreach (value in row) {
        if (value == 2) continue;
        sum = sum + value;
    }
}

function zero(): integer { return 0; }
let left = zero;
let right = zero;
foreach (value in [2, 5]) {
    function next(): integer {
        value = value + 1;
        return value;
    }
    if (value == 2) left = next;
    else right = next;
}
let first = left();
let second = right();
let continued = left();
