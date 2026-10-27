-- Casos límite de cada tipo y de la codificación en CSV.
CREATE TABLE tipos (
  n INT,
  f FLOAT,
  s VARCHAR(40),
  b BOOLEAN,
  d DATE
);

INSERT INTO tipos VALUES
  (9223372036854775807, 3.14159, '', TRUE, '2000-02-29'),
  (-9223372036854775807 - 1, 0.0, 'línea con , coma', FALSE, '1999-12-31'),
  (0, 1 / 2, 'comilla "doble" y ''simple''', NULL, '1900-01-01'),
  (7 * 6 - 2, 10 / 4.0, '\N', TRUE, NULL),
  (NULL, NULL, NULL, NULL, NULL),
  (-5, -2.5, '\barra', FALSE, '2024-12-31'),
  (1, 1, 'salto
de línea', TRUE, '2023-01-01');
