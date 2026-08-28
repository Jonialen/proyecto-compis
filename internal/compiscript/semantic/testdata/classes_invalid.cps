class Missing : Unknown {}
class Left : Right {}
class Right : Left {}
class Base { let value: integer; const fixed: integer = 1; function constructor(value: integer) { this.value = value; } }
class Broken : Base { let value: integer; let own: integer; function own() {} }
class Child : Base {}
let badType: Base = new Base("wrong");
let badArity: Base = new Base();
let inheritedConstructor: Child = new Child(1);
let absent = new Absent();
let member = new Child().missing;
print(this);
function ordinary() { print(this); }
let base = new Base(1);
base.missing = 1;
base.value = "wrong";
base.fixed = 2;
let leaked = new LocalChild();
