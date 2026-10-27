-- Tienda: cuatro tablas, consultas con JOIN, agregaciones, subconsultas,
-- índices (con el plan de ejecución en el resultado) y UPDATE/DELETE.
CREATE TABLE clientes (id INT PRIMARY KEY, nombre VARCHAR(20) NOT NULL, ciudad VARCHAR(20), registro DATE, vip BOOLEAN);
CREATE TABLE productos (id INT PRIMARY KEY, nombre VARCHAR(20) NOT NULL, categoria VARCHAR(15), precio FLOAT NOT NULL, stock INT);
CREATE TABLE pedidos (id INT PRIMARY KEY, cliente_id INT NOT NULL, fecha DATE NOT NULL, estado VARCHAR(10) NOT NULL);
CREATE TABLE detalle (pedido_id INT NOT NULL, producto_id INT NOT NULL, cantidad INT NOT NULL, descuento FLOAT);

INSERT INTO clientes VALUES
  (1, 'Ana', 'Guatemala', '2023-01-15', TRUE),
  (2, 'Bruno', 'Antigua', '2023-03-02', FALSE),
  (3, 'Carla', 'Guatemala', '2022-11-30', TRUE),
  (4, 'Diego', 'Xela', '2024-02-29', NULL),
  (5, 'Elena', 'Antigua', '2024-05-10', FALSE),
  (6, 'Fabio', NULL, '2021-07-01', TRUE);
INSERT INTO productos VALUES
  (10, 'Teclado', 'Perifericos', 25.5, 40),
  (11, 'Mouse', 'Perifericos', 12.25, 100),
  (12, 'Monitor', 'Pantallas', 180, 15),
  (13, 'Laptop', 'Computo', 950, 5),
  (14, 'Cable USB', 'Accesorios', 3.75, 500),
  (15, 'Silla', 'Muebles', 120, 0),
  (16, 'Escritorio', 'Muebles', 210.5, NULL);
INSERT INTO pedidos VALUES
  (100, 1, '2024-01-10', 'entregado'),
  (101, 1, '2024-02-14', 'entregado'),
  (102, 2, '2024-02-20', 'pendiente'),
  (103, 3, '2024-03-05', 'entregado'),
  (104, 3, '2024-03-18', 'cancelado'),
  (105, 5, '2024-04-01', 'pendiente'),
  (106, 1, '2024-04-22', 'pendiente');
INSERT INTO detalle VALUES
  (100, 10, 1, NULL), (100, 11, 2, 0.1), (101, 12, 1, NULL), (102, 14, 10, NULL),
  (103, 13, 1, 0.05), (103, 11, 1, NULL), (104, 15, 2, NULL), (105, 10, 3, 0.2),
  (106, 16, 1, NULL), (106, 14, 4, 0.0);

SELECT c.nombre, p.fecha, p.estado FROM clientes c INNER JOIN pedidos p ON p.cliente_id = c.id
  WHERE p.estado <> 'cancelado' AND p.fecha >= '2024-02-01' ORDER BY p.fecha, c.nombre;
SELECT c.nombre, COUNT(*) AS pedidos FROM clientes c INNER JOIN pedidos p ON p.cliente_id = c.id
  GROUP BY c.nombre ORDER BY pedidos DESC, c.nombre;
SELECT pr.categoria, SUM(d.cantidad * pr.precio) AS ingreso, COUNT(*) AS lineas
  FROM detalle d INNER JOIN productos pr ON pr.id = d.producto_id
  GROUP BY pr.categoria HAVING SUM(d.cantidad * pr.precio) > 100 ORDER BY ingreso DESC;
SELECT MIN(precio), MAX(precio), AVG(precio), COUNT(stock), SUM(stock) FROM productos;
SELECT nombre FROM clientes WHERE id NOT IN (SELECT cliente_id FROM pedidos) ORDER BY nombre;
SELECT nombre FROM clientes c
  WHERE EXISTS (SELECT 1 FROM pedidos p WHERE p.cliente_id = c.id AND p.estado = 'pendiente') ORDER BY nombre;
SELECT nombre, precio FROM productos WHERE precio > (SELECT AVG(precio) FROM productos) ORDER BY precio DESC;
SELECT nombre, (SELECT COUNT(*) FROM pedidos p WHERE p.cliente_id = c.id) AS n FROM clientes c ORDER BY n DESC, nombre;
SELECT nombre FROM productos
  WHERE id NOT IN (SELECT producto_id FROM detalle WHERE descuento IS NOT NULL) ORDER BY nombre;
SELECT nombre FROM clientes WHERE id IN (
  SELECT cliente_id FROM pedidos WHERE id IN (
    SELECT pedido_id FROM detalle WHERE producto_id IN (
      SELECT id FROM productos WHERE categoria = 'Muebles'))) ORDER BY nombre;
SELECT nombre FROM clientes WHERE vip = FALSE OR ciudad = 'Guatemala' ORDER BY nombre;
SELECT nombre FROM clientes WHERE registro >= '2023-01-01' AND registro < '2024-01-01' ORDER BY nombre;
SELECT d.pedido_id, SUM(d.cantidad * pr.precio * (1 - d.descuento)) AS total
  FROM detalle d INNER JOIN productos pr ON pr.id = d.producto_id
  GROUP BY d.pedido_id ORDER BY d.pedido_id;

CREATE INDEX idx_cli_ciudad ON clientes (ciudad);
CREATE INDEX idx_ped_cliente ON pedidos (cliente_id);
CREATE INDEX idx_prod_precio ON productos (precio);

SELECT nombre FROM clientes WHERE ciudad = 'Antigua' ORDER BY nombre;
SELECT id, estado FROM pedidos WHERE cliente_id = 1 ORDER BY id;
SELECT nombre FROM productos WHERE precio >= 100 AND precio <= 250 ORDER BY precio;
SELECT c.nombre, p.id FROM clientes c INNER JOIN pedidos p ON p.cliente_id = c.id
  WHERE c.ciudad = 'Guatemala' AND p.cliente_id = 3 ORDER BY p.id;

UPDATE productos SET stock = stock - 5 WHERE id = 10;
UPDATE productos SET precio = precio + 10 WHERE precio >= 200;
UPDATE pedidos SET estado = 'entregado' WHERE cliente_id = 1 AND estado = 'pendiente';
DELETE FROM detalle WHERE pedido_id IN (SELECT id FROM pedidos WHERE estado = 'cancelado');
DELETE FROM pedidos WHERE estado = 'cancelado';
INSERT INTO pedidos VALUES (107, 1, '2024-05-02', 'pendiente');

SELECT id, estado FROM pedidos WHERE cliente_id = 1 ORDER BY id;
SELECT nombre, precio, stock FROM productos WHERE precio >= 150 ORDER BY precio;
SELECT COUNT(*) AS pedidos, (SELECT COUNT(*) FROM detalle) AS lineas FROM pedidos;
SELECT nombre FROM productos WHERE precio = 220.5;
