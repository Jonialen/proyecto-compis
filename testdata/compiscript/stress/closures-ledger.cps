// Virtual methods, inherited construction, escaped cells and bound receivers.
class Account {
  let balance: integer;

  function constructor(start: integer) {
    this.balance = start;
  }

  function adjust(delta: integer): integer {
    this.balance = this.balance + delta;
    return this.balance;
  }
}

class BonusAccount: Account {
  let bonus: integer = 2;

  function adjust(delta: integer): integer {
    this.balance = this.balance + delta + this.bonus;
    return this.balance;
  }
}

function zero(): integer { return 0; }
let tick = zero;

function attach(account: Account) {
  let visits = 0;
  function next(): integer {
    visits = visits + 1;
    return account.adjust(visits);
  }
  tick = next;
}

let account: Account = new BonusAccount(10);
attach(account);
let first = tick();
let second = tick();
let saved = tick;

let other: Account = new BonusAccount(100);
attach(other);
let third = saved();
let fourth = tick();

let totals = [first, second, third, fourth];
let sum = 0;
foreach (value in totals) { sum = sum + value; }

let direct = account.adjust;
let finalBalance = direct(5);
let result = sum + finalBalance;
print(first);
print(second);
print(third);
print(fourth);
print(finalBalance);
print(result);
