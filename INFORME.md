# TP Nivelador: Docker, Comunicaciones y Concurrencia
Lucas Ariel Conde Cardó - 112201

---

## 1. Introducción

El trabajo consiste en el desarrollo de un sistema distribuído simple en donde distintas agencias de lotería cargan participantes en un servidor central, que luego realiza el sorteo e informa los participantes que han ganado.

El sistema se compone de dos tipos de entidades independientes:
- **Clientes (Agencias de lotería):** Implementadas en **Golang**. Estas leen las apuestas desde un archivo `.CSV` de entrada (`INPUT_FILE`) y las envían en *batches* hacia el servidor. Luego,  esperan los resultados del sorteo y, finalmente, los persisten persisten en otro archivo `.CSV` de salida (`OUTPUT_FILE`).
- **Servidor (Central de lotería):** Implementada en **Python**. Esta entidad acepta múltiples conexiones concurrentes por parte de las agencias, persiste las apuestas (en un archivo `bets.csv`), y, tras recibir las apuestas tomadas por una cantidad (*quorum*) mínima de agencias (`AGENCY_QUORUM_MIN`), calcula los ganadores y responde a cada agencia con sus respectivos ganadores.

---

## 2. Protocolo de comunicación

Para cumplir con los solicitado en los ejercicios 5 y 6, definí e implementé un **protocolo de comunicación propio a nivel de aplicación, sobre TCP**.

### 2.1. Estructura general de los mensajes

Los paquetes transmitidos respetan un formato con encabezado de longitud fija, que permite a las entidades conocer qué tipo de mensaje llegó y cuantos bytes de payload (cuerpo del mensaje) debe leer:

1. **Longitud del payload (4 bytes):** Entero sin signo de 32 bits en formato *Big-Endian* que indica la cantidad exacta de bytes correspondientes al cuerpo del mensaje.
2. **Tipo de mensaje (1 byte):** Identificador numérico del propósito del mensaje.
3. **Payload (N bytes):** Contenido del mensaje codificado en UTF-8 utilizando el delimitador `|` para separar los distintos campos.

### 2.2. Tipos de mensajes

| Tipo de mensaje | Código | Dirección | Descripción y formato |
| :--- | :---: | :---: | :--- |
| `MSG_TYPE_BET` | `0` | Cliente $\to$ Servidor | Envía una única apuesta (usado para la resolución del ejercicio 5). Formato: `agency_id\|nombre\|apellido\|dni\|nacimiento\|numero` |
| `MSG_TYPE_MULTI_BETS` | `1` | Cliente $\to$ Servidor | Envía un *batch* de apuestas. Contiene un "segundo header" de 4 bytes adicionales, indicando la cantidad de apuestas en el *batch*: `[4B Longitud][1B Tipo=1][4B Cantidad de apuestas][Payload]`. El payload incluye el `agency_id` una sola vez como prefijo y concatena los datos de las apuestas: `agency_id\|nom1\|ape1\|dni1...\|nom2\|ape2\|dni2...` |
| `MSG_TYPE_REQUEST_WINNERS` | `2` | Cliente $\to$ Servidor | Lo emiten las agencias cuando concluye el envío de sus apuestas y solicita la lista de ganadores. Su payload tiene longitud `0`. |
| `MSG_TYPE_WINNER` | `3` | Servidor $\to$ Cliente | Mensaje enviado por el servidor por cada apuesta ganadora perteneciente a la agencia: `agency_id\|nombre\|apellido\|dni\|nacimiento\|numero`. |
| `MSG_TYPE_ACK` | `4` | Servidor $\to$ Cliente | Confirmación enviada por el servidor tras procesar y persistir exitosamente en disco la totalidad de las apuestas contenidas en un *batch*. Su payload también tiene longitud `0`. |

### 2.3. Señalización de fin de ganadores
El servidor envía los ganadores uno por uno mediante mensajes `MSG_TYPE_WINNER`. Al finalizar los correspondientes a esa agencia, el servidor cierra la conexión (`client_socket.close()`). En consecuencia, el cliente al intentar leer el siguiente encabezado, detecta el cierre mediante `io.EOF`, dando por finalizada la recepción.

---

## 3. Concurrencia y sincronización en el servidor

El servidor procesa múltiples conexiones en simultáneo ejecutando cada cliente en un hilo independiente. La sincronización entre estos hilos y el acceso a los recursos compartidos (entre ellos, el archivo sobre el que lee y escribe la clase `Lottery`), se diseñó con las siguientes consideraciones:

### 3.1. Sincronización del quorum mínimo con *condition variable*

La consigna exige que el servidor no realice el cálculo y envío de ganadores hasta que un número mínimo de agencias (`AGENCY_QUORUM_MIN`) hayan notificado el fin de la carga de sus apuestas.

Para resolver esto, inicialmente implementé la utilización de una *barrera*. Sin embargo, debido a su naturaleza cíclica (diseñada para un número exacto de hilos, volviendo luego a 0), me veía obligado a introducir "trucos raros", para evitar que agencias posteriores al quorum quedaran atrapadas en un segundo ciclo.

Es por ello que, en su lugar, decidí utilizar una *condition varible* (`threading.Condition`). De esta forma, la sincornización se realiza de la siguiente manera:
1. Cada hilo, al recibir `MSG_TYPE_REQUEST_WINNERS`, ingresa al método `_wait_for_quorum()`.
2. Incrementa `self.agencies_ready` (protegido mediante un lock).
3. Si `self.agencies_ready >= self.agency_quorum_min`, establece `self.quorum_reached = True` y emite un `notify_all()` para despertar a todos los hilos en espera.
4. Los hilos que llegaron antes del quorum esperan en un bucle `while not self.quorum_reached and self.running: self.quorum_condition.wait()`.
5. **Hilos que llegan a este punto post-quorum:** Cualquier agencia que finalice su carga de apuestas luego de que el quorum ya haya sido alcanzado adquiere la condición, observa `self.quorum_reached == True` y continúa de inmediato sin bloquearse.

### 3.2. Protección de secciones críticas y optimización del lock de Lotería

Múltiples agencias pueden enviar lotes de apuestas en cualquier orden. Para evitar escrituras intercaladas o corrompimientos en el archivo `.CSV` de almacenamiento (`bets.csv`), los llamdos a `self.lottery.store_bets(bets)` y `self.lottery.load_bets()` se encuentran protegidos bajo un lock de acceso exclusivo.

---

## 4. Eficiencia de memoria

Para cumplir con los requerimientos no funcionales de escalabilidad y evitar agotar la memoria ante un gran volumen de registros (como los evaluados en la prueba *memory profile*), el archivo de entrada no se carga por completo en memoria. Se utiliza `bufio.Scanner` en [`bets_io_handler.go`](services/client/src/client/bets_io_handler.go) para leer línea por línea, construyendo *batches* de tamaño acotado (`BATCH_SIZE`) antes de emitir cada mensaje.

---

## 5. Graceful shutdown

Tanto el cliente como el servidor contemplan la recepción de las señales de terminación del sistema operativo (`SIGTERM` y `SIGINT`), garantizando la liberación ordenada de recursos antes de finalizar:

- **En el servidor (`main.py` / `server.py`):**
  - Un manejador de señal invoca a `server.stop()`.
  - Se activa `self.running = False`.
  - Se invoca `self.quorum_condition.notify_all()` para desbloquear cualquier hilo esperando el quorum.
  - Se cierra el socket de escucha (`server_socket`), interrumpiendo bloqueos en `accept()`.
  - Se realiza el `shutdown` y `close` de todos los sockets de clientes activos en `self.client_sockets`.
  - Se espera la finalización de los hilos de atención mediante `thread.join(timeout=THREADS_TIMEOUT_TIME)`.
- **En el cliente (`main.go` / `client.go`):**
  - Se emplea `signal.NotifyContext` para gestionar la cancelación a lo largo del árbol de llamadas.
  - Los sockets y descriptores de archivos de entrada y salida se cierran mediante el llamado a `defer`.