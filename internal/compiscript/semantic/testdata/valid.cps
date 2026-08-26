let global: integer = 1; function sum(left: integer, right: float): float { print(global); let global: float = left + right; { const label: string = "sum" + "!"; print(label); } return global / 2; }
