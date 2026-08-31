import socket

def recv_all(sock: socket.socket, size: int) -> bytes:
    received = bytearray()
    while len(received) < size:
        bytes_read = sock.recv(size - len(received))
        if not bytes_read:
            break
        received.extend(bytes_read)
    return bytes(received)


def send_all(socket: socket.socket, bytes):
    total_sent = 0
    while total_sent < len(bytes):
        sent = socket.send(bytes[total_sent:])
        total_sent += sent
    return total_sent
