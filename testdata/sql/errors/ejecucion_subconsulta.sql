-- mode: execute
CREATE TABLE t (id INT PRIMARY KEY, v INT);
INSERT INTO t VALUES (1, 10), (2, 20), (3, 30);
SELECT id FROM t WHERE v = (SELECT v FROM t WHERE id > 1);
SELECT id FROM t;
