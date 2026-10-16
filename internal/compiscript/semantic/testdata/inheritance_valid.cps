class Animal {
  let nombre: string;
  function constructor(nombre: string) { this.nombre = nombre; }
  function hablar(): string { return this.nombre + " hace ruido."; }
}

class Perro : Animal {
  function hablar(): string { return this.nombre + " ladra."; }
}

class Cachorro : Perro {
  let edad: integer;
  function hablar(): string { return this.nombre + " ladra bajito."; }
}

let perro: Perro = new Perro("Toby");
let sonido: string = perro.hablar();
let cachorro: Cachorro = new Cachorro("Lola");
