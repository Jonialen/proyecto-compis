function factorial(n: integer): integer {
    if (n <= 1) return 1;
    else return n * factorial(n - 1);
}

function fibonacci(n: integer): integer {
    if (n <= 1) return n;
    else return fibonacci(n - 1) + fibonacci(n - 2);
}

let product = factorial(5);
let sequence = fibonacci(8);
