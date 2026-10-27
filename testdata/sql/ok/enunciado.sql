-- Ejemplos del enunciado, con sus tablas (estudiantes, cursos, inscripciones)
-- y las dos consultas tal como aparecen en el PDF.
CREATE TABLE estudiantes (id INT PRIMARY KEY, nombre VARCHAR(50) NOT NULL, edad INT NOT NULL);
CREATE TABLE cursos (id INT PRIMARY KEY, nombre VARCHAR(50) NOT NULL);
CREATE TABLE inscripciones (estudiante_id INT NOT NULL, curso_id INT NOT NULL);

INSERT INTO estudiantes VALUES (1, 'Ana', 20), (2, 'Luis', 17), (3, 'Marta', 22), (4, 'Zoe', 18), (5, 'Pedro', 16);
INSERT INTO cursos VALUES (1, 'Compiladores'), (2, 'Bases de Datos'), (3, 'Algebra');
INSERT INTO inscripciones VALUES (1, 3), (2, 1), (3, 3), (4, 2), (5, 3), (1, 1);

-- Ejemplo del enunciado: solo los nombres que cumplen la condición.
SELECT nombre
FROM estudiantes
WHERE edad >= 18;

-- Subconsulta del enunciado.
SELECT nombre
FROM estudiantes
WHERE id IN (SELECT estudiante_id
             FROM inscripciones
             WHERE curso_id = 3);

-- Las otras formas de subconsulta que pide el enunciado.
SELECT nombre FROM estudiantes
WHERE id NOT IN (SELECT estudiante_id FROM inscripciones WHERE curso_id = 3);
SELECT nombre FROM estudiantes e
WHERE EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = e.id AND i.curso_id = 1);
SELECT nombre FROM estudiantes e
WHERE NOT EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = e.id AND i.curso_id = 1);
SELECT nombre FROM estudiantes WHERE edad > (SELECT AVG(edad) FROM estudiantes);

-- JOIN, agregaciones, GROUP BY y HAVING.
SELECT c.nombre, COUNT(*) AS inscritos, MIN(e.edad), MAX(e.edad), AVG(e.edad), SUM(e.edad)
FROM cursos c
INNER JOIN inscripciones i ON i.curso_id = c.id
INNER JOIN estudiantes e ON e.id = i.estudiante_id
GROUP BY c.nombre
HAVING COUNT(*) >= 2
ORDER BY inscritos DESC, c.nombre;

-- Con un índice, la misma consulta del enunciado lee solo los registros necesarios.
CREATE INDEX idx_edad ON estudiantes (edad);
SELECT nombre
FROM estudiantes
WHERE edad >= 18;

INSERT INTO estudiantes VALUES (6, 'Sofia', 30);
UPDATE estudiantes SET edad = 19 WHERE id = 5;
DELETE FROM estudiantes WHERE edad = 17;
SELECT nombre, edad FROM estudiantes WHERE edad >= 18 ORDER BY edad DESC, nombre;
