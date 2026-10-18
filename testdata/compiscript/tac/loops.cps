let total = 0;
let i = 0;
while (i < 4) {
    i = i + 1;
    if (i == 2) continue;
    total = total + i;
}
do {
    total = total + 1;
} while (false);
for (let j = 0; ; j = j + 1) {
    if (j == 4) break;
    if (j == 2) continue;
    total = total + j;
}
