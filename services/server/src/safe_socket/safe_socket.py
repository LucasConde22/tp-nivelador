import socket

def recv_all(sock: socket.socket, size: int) -> bytes:
    """
    Receives a specified number of bytes from a socket, ensuring that the exact amount is received.
    """
    received = bytearray()
    while len(received) < size:
        bytes_read = sock.recv(size - len(received))
        if not bytes_read:
            break
        received.extend(bytes_read)
    return bytes(received)


def send_all(socket: socket.socket, bytes):
    """
    Sends all bytes to a socket, ensuring that the exact amount is sent.
    """
    total_sent = 0
    while total_sent < len(bytes):
        sent = socket.send(bytes[total_sent:])
        total_sent += sent
    return total_sent
