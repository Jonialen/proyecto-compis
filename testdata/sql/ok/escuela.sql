-- Escuela completa: tres tablas relacionadas, los cinco tipos, restricciones,
-- NULL, comillas escapadas, acentos, expresiones constantes y columnas parciales.
CREATE TABLE estudiantes (
  id INT PRIMARY KEY,
  nombre VARCHAR(30) NOT NULL,
  edad INT,
  promedio FLOAT,
  activo BOOLEAN,
  nacimiento DATE
);
CREATE TABLE cursos (
  id INT PRIMARY KEY,
  nombre VARCHAR(40) NOT NULL,
  creditos INT NOT NULL
);
CREATE TABLE inscripciones (
  estudiante_id INT NOT NULL,
  curso_id INT NOT NULL,
  nota FLOAT,
  fecha DATE
);

INSERT INTO estudiantes VALUES
  (1, 'Ana María', 20, 88.5, TRUE, '2005-03-14'),
  (2, 'Luis', 17, 71, TRUE, '2008-11-02'),
  (3, 'O''Brien', 22, NULL, FALSE, '2003-02-28'),
  (4, 'Zoë', 21, 95.25, TRUE, '2004-02-29'),
  (5, 'Pedro', -1 + 19, 60.0 * 1.5, NULL, NULL);

INSERT INTO cursos VALUES
  (1, 'Compiladores', 5),
  (2, 'Bases de Datos', 4),
  (3, 'Álgebra', 3),
  (4, 'Física', 2 * 3);

INSERT INTO inscripciones (curso_id, estudiante_id, nota, fecha) VALUES
  (1, 1, 90, '2026-02-01'),
  (1, 2, NULL, '2026-02-01');
INSERT INTO inscripciones (estudiante_id, curso_id) VALUES (3, 2), (4, 1), (4, 3), (5, 4);

CREATE TABLE temporal (x INT);
INSERT INTO temporal VALUES (1), (2), (3);
DROP TABLE temporal;
