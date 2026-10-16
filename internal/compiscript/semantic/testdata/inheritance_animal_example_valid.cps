class Animal {
  let nombre: string;

  function constructor(nombre: string) {
    this.nombre = nombre;
  }

  function hacerSonido(): string {
    return "Algún sonido genérico";
  }

  function describir(): string {
    return this.nombre + " dice: " + this.hacerSonido();
  }
}

class Perro : Animal {
  let raza: string;

  function constructor(nombre: string, raza: string) {
    this.nombre = nombre;
    this.raza = raza;
  }

  function hacerSonido(): string {
    return "Guau";
  }
}

class Gato : Animal {
  function hacerSonido(): string {
    return "Miau";
  }
}

let miPerro: Animal = new Perro("Rex", "Labrador");
let miGato: Animal = new Gato("Michi");
