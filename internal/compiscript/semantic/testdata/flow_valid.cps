function choose(flag: boolean): integer {
  if (flag) { return 1; } else { return 2; }
}
function classify(value: integer): integer {
  switch (value) { case 1: return 1; case 2.5: return 2; default: return 0; }
}
while (true) { break; }
do { continue; } while (false);
for (;;) { break; }
for (let i: integer = 0; i < 3; i = i + 1) { if (i == 2) { continue; } }
while (true) { switch (1) { case 1: continue; default: break; } }
