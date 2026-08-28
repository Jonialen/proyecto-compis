function bad(flag: boolean, amount: float): integer {
  if (1) { return "wrong"; }
}
bad(1.5, "wrong", 3);
function empty(): integer { return; }
function maybeWhile(flag: boolean): integer { while (flag) { return 1; } }
function maybeFor(flag: boolean): integer { for (; flag; ) { return 1; } }
function partialForever(flag: boolean): integer { while (true) { if (flag) { return 1; } } }
return 1;
