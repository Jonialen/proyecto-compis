function choose(flag: boolean): integer {
  if (flag) { return 1; } else { return 2; }
}
function classify(value: integer): integer {
  switch (value > 0) { case true: return 1; default: return 0; }
}
while (true) { break; }
do { continue; } while (false);
for (;;) { break; }
for (let i: integer = 0; i < 3; i = i + 1) { if (i == 2) { continue; } }
while (true) { switch (true) { case true: continue; default: break; } }
