-- mode: execute
CREATE TABLE t (id INT, d INT);
INSERT INTO t VALUES (1, 5), (2, 0);
SELECT id, 10 / d FROM t;
