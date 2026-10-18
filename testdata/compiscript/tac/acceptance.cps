class Basket {
    let rows: integer[][];
    function constructor(rows: integer[][]) { this.rows = rows; }
    function sum(): integer {
        let total = 0;
        foreach (row in rows) {
            foreach (value in row) {
                switch (value == 2) {
                    case true: continue;
                    default: total = total + value;
                }
            }
        }
        return total;
    }
}

function zero(): integer { return 0; }
let next = zero;
function install(basket: Basket, seed: integer) {
    let count = seed;
    function step(): integer {
        count = count + 1;
        basket.rows[0][0] = basket.rows[0][0] + count;
        return basket.sum() + count;
    }
    next = step;
}
function triangular(n: integer): integer {
    if (n == 0) return 0;
    else return n + triangular(n - 1);
}

let rows: integer[][] = [[1, 2], [], [3, 4]];
let basket = new Basket(rows);
install(basket, 10);
let first = next();
let second = next();
let saved = next;
let otherBasket = new Basket([[5]]);
install(otherBasket, 0);
let independent = next();
let continued = saved();
let total = 0;
for (let i = 0; i < 2; i = i + 1) { total = total + triangular(2); }
while (total < 8) { total = total + 1; }
do { total = total + 2; } while (total < 10);
let shared = rows[0][0];
let result = first + second + independent + continued + total;
