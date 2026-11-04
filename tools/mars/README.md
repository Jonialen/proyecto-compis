# Ejecutar MIPS con MARS 4.5

MARS queda instalado localmente en `tools/mars/Mars4_5.jar`; el JAR y las preferencias de Java no se versionan ni se instalan globalmente. Se requiere Java en `PATH`, `bash`, `curl` y `shasum`.

## Desde la raíz del repositorio

```sh
bash scripts/install-mars.sh
bash scripts/test-mars.sh
bash scripts/mars.sh testdata/mips/smoke.asm
```

La prueba debe terminar con `MARS smoke: execution and invalid assembly passed`. El último comando ejecuta ensamblador escrito a mano e imprime `MARS OK`; **esto verifica el simulador, no un backend Compiscript terminado**.

Para ejecutar otro programa: `bash scripts/mars.sh ruta/programa.asm`. El directorio de trabajo no cambia, por lo que las rutas relativas se interpretan desde donde invocás el comando. El wrapper fuerza modo sin interfaz gráfica, omite el banner, separa mensajes del simulador por stderr y devuelve código 1 por errores de ensamblado o 2 por errores de simulación. En una ejecución correcta, MARS agrega un separador de una línea vacía a stderr; stdout contiene solo la salida del programa. Las preferencias de Java se guardan en `tools/mars/.prefs/`.

## Procedencia y verificación

- [Repositorio original](https://github.com/dpetersanderson/MARS).
- [Release oficial MARS v4.5, etiqueta `v.4.5.1`](https://github.com/dpetersanderson/MARS/releases/tag/v.4.5.1).
- Asset: `Mars4_5.jar`, 4.169.142 bytes.
- SHA-256 fijado en `SHA256SUMS`: `ac340b676ba2b62246b9df77e62f81ad4447bcfd329ab539716bcd09950b7096`.

El release no publica un digest independiente: el checksum se calculó después de la primera descarga HTTPS del asset oficial. Protege contra cambios posteriores; no constituye una firma del autor. El instalador verifica antes de activar el JAR y el wrapper vuelve a verificar antes de ejecutarlo. Una instalación existente con checksum incorrecto se rechaza, no se reemplaza silenciosamente.

El instalador requiere red solo cuando falta el JAR; la ejecución y las pruebas siguientes funcionan sin red. Java y las herramientas del sistema no se descargan automáticamente. No se modifica el módulo Go ni sus dependencias.
