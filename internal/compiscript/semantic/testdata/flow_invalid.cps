if (1) { print(1); }
while (1) { break; }
do { print(1); } while ("no");
for (let i: integer = 0; 1; i = i + 1) { print(i); }
switch (true) {
  case 1: print(1);
  case true: continue;
  case true: print(2);
}
switch (1) { case 1: print(1); case 1.0: print(2); }
switch (0) { case 0: print(1); case -0: print(2); }
switch ("a") { case "a": print(1); case "\u0061": print(2); }
break;
continue;
return 1;
function dead(): integer { return 1; print(2); let x: integer = 3; }
function branches(flag: boolean): integer {
  if (flag) { return 1; } else { return 2; }
  print(3);
}
function cases(value: integer): integer {
  switch (value) { case 1: return 1; print(1); default: return 0; }
}
