-- mode: execute
CREATE TABLE t (id INT PRIMARY KEY, nombre VARCHAR(5) NOT NULL, n INT);
INSERT INTO t VALUES (1, 'a', 1), (2, 'b', 2);
UPDATE t SET n = n + 1 WHERE id = 1;
UPDATE t SET id = 1 WHERE id = 2;
