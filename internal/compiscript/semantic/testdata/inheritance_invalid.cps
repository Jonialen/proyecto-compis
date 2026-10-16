class Base {
  let etiqueta: string;
  function constructor(etiqueta: string) { this.etiqueta = etiqueta; }
  function leer(): string { return this.etiqueta; }
}

class FirmaDistinta : Base {
  function leer(valor: integer): string { return this.etiqueta; }
}

class RetornoDistinto : Base {
  function leer(): integer { return 1; }
}

class CampoRepetido : Base {
  let etiqueta: string;
}

class CampoVueltoMetodo : Base {
  function etiqueta(): string { return ""; }
}

let faltanArgumentos: Base = new Base();
