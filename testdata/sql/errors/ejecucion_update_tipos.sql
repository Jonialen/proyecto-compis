-- mode: execute
CREATE TABLE t (id INT PRIMARY KEY, nombre VARCHAR(5) NOT NULL);
INSERT INTO t VALUES (1, 'a'), (2, 'b');
UPDATE t SET nombre = 'demasiado largo' WHERE id = 2;
