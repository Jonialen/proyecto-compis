-- Clave primaria, NOT NULL, varios INSERT y tablas con nombres en distinta capitalización.
CREATE TABLE Productos (codigo INT PRIMARY KEY, nombre VARCHAR(20) NOT NULL, precio FLOAT NOT NULL);
INSERT INTO productos VALUES (10, 'Lápiz', 1.5), (20, 'Cuaderno', 12.75);
INSERT INTO PRODUCTOS (precio, codigo, nombre) VALUES (3, 30, 'Regla');
INSERT INTO Productos (codigo, nombre, precio) VALUES (40, 'Borrador', 0.5 + 0.25), (50, 'Tijeras', 4 * 2.5);
CREATE TABLE Marcas (id INT PRIMARY KEY, nombre VARCHAR(10));
INSERT INTO marcas (id) VALUES (1), (2);
DROP TABLE marcas;
CREATE TABLE marcas (id INT, nombre VARCHAR(10) NOT NULL);
INSERT INTO marcas VALUES (1, 'A'), (1, 'A');
