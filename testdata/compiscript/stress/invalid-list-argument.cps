// Intentional semantic error after valid declarations and a valid first call.
class Vault {
  let balance: integer = 10;
  function debit(amount: integer): integer {
    this.balance = this.balance - amount;
    return this.balance;
  }
}

function total(values: integer[]): integer {
  let sum = 0;
  foreach (value in values) { sum = sum + value; }
  return sum;
}

let vault = new Vault();
let safe = total([1, 2, 3]);
let remaining = vault.debit(safe);
let invalid = total(["wrong"]);
print(remaining + invalid);
