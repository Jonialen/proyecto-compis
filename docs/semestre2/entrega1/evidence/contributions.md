# Evidencia de contribuciones

Esta evidencia registra únicamente la autoría observable en el historial del repositorio. No infiere un contribuyente adicional para satisfacer el requisito del curso.

## Reproducción del conteo

Ejecute desde la raíz del repositorio:

```bash
git shortlog -sne --all
```

Observado el 2026-08-30 en la revisión base `97c394b` (los commits de documentación posteriores quedan intencionalmente fuera de esta instantánea):

```text
53  jonialen <jonathanalej09@gmail.com>
17  Jonialen <jonathanalej09@gmail.com>
 9  DijanU <pad23663@uvg.edu.gt>
 1  Jonathan Díaz <77712004+Jonialen@users.noreply.github.com>
```

El comando reporta 80 commits distribuidos en cuatro firmas de autor. Las dos primeras firmas comparten el mismo correo electrónico y difieren solo en el uso de mayúsculas, por lo que son alias de una misma identidad de contribuyente observable (70 commits en total). La firma noreply de GitHub contiene el mismo identificador de cuenta `Jonialen`; se documenta como otro alias de Jonathan/Jonialen (1 commit), y no como una tercera persona. `DijanU <pad23663@uvg.edu.gt>` es la segunda identidad de contribuyente observable (9 commits).

## Limitación del requisito

Por lo tanto, el historial de Git respalda dos identidades de contribuyentes, no tres. Esta evidencia no puede demostrar el cumplimiento de un requisito de contribución de tres integrantes. Agregar un nombre o una atribución sin un commit correspondiente sería inexacto; cualquier actualización futura debe regenerarse a partir del historial real de Git.
