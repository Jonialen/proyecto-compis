class Base {
    let n: integer = 2;
    function read(): integer { return this.n; }
}
class Child: Base {
    function read(): integer { return this.n + 8; }
}
let box: Base = new Child();
let other: Base = new Child();
print("objeto:");
print(box);
print("inicial:");
print(box.n);
let alias = box;
alias.n = 4;
print("alias:");
print(box.n);
print("independiente:");
print(other.n);
print("metodo sobrescrito:");
print(box.read());

function fib(n: integer): integer {
    if (n <= 1) return n;
    else return fib(n - 1) + fib(n - 2);
}
print("fib(8):");
print(fib(8));
