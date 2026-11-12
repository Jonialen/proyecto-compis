CREATE TABLE clientes (id INT PRIMARY KEY, nombre VARCHAR(20));
CREATE TABLE pedidos (id INT PRIMARY KEY, cliente_id INT, FOREIGN KEY (cliente_id) REFERENCES clientes(id));
INSERT INTO clientes VALUES (1, 'Ana'), (2, 'Luis');
INSERT INTO pedidos VALUES (10, 1);
