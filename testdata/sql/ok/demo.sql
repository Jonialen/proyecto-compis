-- Entrega 3: CREATE, INSERT y DROP se ejecutan; SELECT/UPDATE/DELETE solo se analizan.
CREATE TABLE estudiantes (
  id INT PRIMARY KEY,
  nombre VARCHAR(50) NOT NULL,
  edad INT,
  promedio FLOAT,
  activo BOOLEAN,
  nacimiento DATE
);

CREATE TABLE cursos (id INT PRIMARY KEY, nombre VARCHAR(40));

INSERT INTO estudiantes VALUES
  (1, 'Ana', 20, 88.5, TRUE, '2005-03-14'),
  (2, 'Luis', 17, 71.0, TRUE, '2008-11-02');

INSERT INTO cursos (id, nombre) VALUES (3, 'Compiladores');
