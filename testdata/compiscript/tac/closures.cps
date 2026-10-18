function zero(): integer {
    return 0;
}

let saved = zero;

function install(seed: integer) {
    let count = seed;
    function next(): integer {
        count = count + 1;
        return count;
    }
    saved = next;
}

install(10);
let first = saved();
let second = saved();
let original = saved;
install(20);
let independent = saved();
let continued = original();
