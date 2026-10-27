-- statements: 27
-- Consultas complejas sobre un esquema sin datos: prueban solo el análisis (léxico, sintaxis, AST y semántico).
CREATE TABLE estudiantes (id INT PRIMARY KEY, nombre VARCHAR(30) NOT NULL, edad INT, promedio FLOAT, activo BOOLEAN, nacimiento DATE);
CREATE TABLE cursos (id INT PRIMARY KEY, nombre VARCHAR(40) NOT NULL, creditos INT NOT NULL);
CREATE TABLE inscripciones (estudiante_id INT NOT NULL, curso_id INT NOT NULL, nota FLOAT, fecha DATE);

SELECT nombre FROM estudiantes WHERE edad >= 18;
SELECT * FROM cursos ORDER BY creditos DESC, nombre ASC;
SELECT e.nombre AS estudiante, c.nombre AS curso, i.nota
  FROM estudiantes e
  INNER JOIN inscripciones i ON i.estudiante_id = e.id
  INNER JOIN cursos c ON c.id = i.curso_id
  WHERE (i.nota >= 60 OR i.nota IS NULL) AND NOT e.activo = FALSE
  ORDER BY estudiante, i.nota DESC;
SELECT c.nombre, COUNT(*) AS inscritos, AVG(i.nota) AS promedio, MIN(i.nota), MAX(i.nota), SUM(c.creditos)
  FROM cursos c INNER JOIN inscripciones i ON i.curso_id = c.id
  GROUP BY c.nombre
  HAVING COUNT(*) >= 2 AND AVG(i.nota) > 70
  ORDER BY inscritos DESC;
SELECT activo, COUNT(*) FROM estudiantes GROUP BY activo;
SELECT COUNT(*), AVG(promedio) FROM estudiantes;
SELECT nombre FROM estudiantes
  WHERE id IN (SELECT estudiante_id FROM inscripciones WHERE curso_id = 3);
SELECT nombre FROM estudiantes
  WHERE id NOT IN (SELECT estudiante_id FROM inscripciones);
SELECT nombre FROM estudiantes e
  WHERE EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = e.id AND i.nota > 80);
SELECT nombre FROM cursos c
  WHERE NOT EXISTS (SELECT 1 FROM inscripciones i WHERE i.curso_id = c.id);
SELECT nombre FROM estudiantes WHERE promedio > (SELECT AVG(promedio) FROM estudiantes);
SELECT nombre FROM estudiantes
  WHERE id IN (SELECT estudiante_id FROM inscripciones
               WHERE curso_id IN (SELECT id FROM cursos WHERE creditos > (SELECT MIN(creditos) FROM cursos)));
SELECT nombre FROM estudiantes WHERE id IN (1, 2, 3) AND nacimiento >= '2004-01-01' AND edad * 2 - 1 > 30 / 2;
SELECT nombre FROM estudiantes WHERE nombre IS NOT NULL AND (edad < 18 OR edad > 21) ORDER BY edad;
UPDATE estudiantes SET promedio = promedio + 5, activo = TRUE WHERE id IN (SELECT estudiante_id FROM inscripciones WHERE nota >= 90);
UPDATE inscripciones SET nota = NULL, fecha = '2026-03-01' WHERE nota < 30;
UPDATE cursos SET creditos = creditos * 2;
DELETE FROM inscripciones WHERE estudiante_id NOT IN (SELECT id FROM estudiantes);
DELETE FROM estudiantes WHERE NOT EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = estudiantes.id) AND activo = FALSE;
DELETE FROM cursos;
CREATE INDEX idx_est_nombre ON estudiantes (nombre);
CREATE INDEX idx_ins_curso ON inscripciones (curso_id);
DROP INDEX idx_est_nombre;
DROP TABLE inscripciones;
