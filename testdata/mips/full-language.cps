function fib(n: integer): integer {
    if (n <= 1) return n;
    else return fib(n - 1) + fib(n - 2);
}
print(fib(8));

class Base {
    let n: integer = 2;
    function read(): integer { return this.n; }
}
class Child: Base {
    function read(): integer { return this.n + 8; }
}
let box: Base = new Child();
let alias = box;
alias.n = 4;
print(box.read());

let values = [1, 2, 3];
values[1] = 5;
let total = 0;
foreach (value in values) { total = total + value; }
print(total);

function zero(): integer { return 0; }
let saved = zero;
function install(seed: integer) {
    let count = seed;
    function next(): integer { count = count + 1; return count; }
    saved = next;
}
install(10);
print(saved());
print(saved());

function show(a, b) { print(a + b); }
show(1, 2.5);
try { values[99] = 0; }
catch (err) { print(err); }
print(values);
print("Compiscript → MIPS");
