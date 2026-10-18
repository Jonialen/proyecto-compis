let total = 0;
for (let i = 0; i < 4; i = i + 1) {
    switch (i < 2) {
        case true:
            total = total + 1;
            continue;
        case false:
            total = total + 2;
            break;
        default:
            total = 99;
    }
}
