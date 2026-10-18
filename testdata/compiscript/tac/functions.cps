let total = 1;

function increment(value: integer): integer {
    value = value + 1;
    return value;
}

function accumulate(count: integer) {
    for (let i = 0; i < count; i = i + 1) {
        total = total + increment(i);
    }
    return;
}

accumulate(3);
let result: float = increment(total);
