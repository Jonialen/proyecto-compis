class Animal {
  let name: string;
  function constructor(name: string) { this.name = name; }
  function speak(): string { return this.name; }
}
class Dog : Animal { let age: integer; function constructor(age: integer) { this.age = age; } }
class Puppy : Dog {}
let animal: Animal = new Dog(2);
let puppy: Puppy = new Puppy();
let inheritedName: string = puppy.name;
let inheritedMethod: string = puppy.speak();
