import socket


def recv_all(socket: socket.socket, size):
    """
    bytes = b''
    remaining = size
    while remaining != 0:
        new_bytes = socket.recv(remaining)

        if len(new_bytes) == 0:
            break # Levantar error?

        remaining -= len(new_bytes)
        bytes += new_bytes
    return bytes
    """
    return socket.recv(size)


def send_all(socket: socket.socket, bytes):
    """
    total_sent = 0
    while total_sent < len(bytes):
        sent = socket.send(bytes[total_sent:])

        if sent == 0:
            return total_sent # Levantar error?

        total_sent += sent
    return total_sent
    """
    return socket.send(bytes)
