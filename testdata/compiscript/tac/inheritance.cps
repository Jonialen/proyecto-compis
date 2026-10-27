// Inheritance demo: compile this file to inspect its TAC, not to execute it.
class Base {
    let first: integer = 1;

    function constructor() {
        // The receiver is still a Child when this constructor is inherited.
        this.set();
    }

    function set() {}

    function getFirst(): integer {
        return this.first;
    }
}

class Child : Base {
    let later: integer = 7;

    // Override the method inherited from Base.
    function set() {
        this.later = 9;
    }
}

// Fields initialize base-to-derived before the nearest constructor runs.
// Child inherits Base.constructor, whose this.set() selects Child.set().
let child = new Child();
let inheritedField = child.getFirst(); // Expected execution value: 1.
let overriddenEffect = child.later;   // Expected execution value: 9, not 7.

// A separate allocation has independent field cells.
let other = new Child();
other.later = 20;
let originalValue = child.later; // Expected execution value: 9.
let separateValue = other.later; // Expected execution value: 20.

// A bound method retains its original receiver.
let bound = other.set;
bound();
let boundEffect = other.later; // Expected execution value: 9.

// Assignment aliases the object; it does not copy its fields.
let alias = child;
alias.later = 30;
let sharedValue = child.later; // Expected execution value: 30.
