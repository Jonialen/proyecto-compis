let divisor = 0;
let quotient: float = 0;
let safe = divisor != 0 && 10 / divisor > 1;
if (safe) {
    quotient = 10 / divisor;
} else {
    quotient = -1;
}
let accepted = safe || quotient == -1;
