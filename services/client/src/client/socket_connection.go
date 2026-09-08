package client

import (
	"io"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

// SocketConnection is a wrapper around an io.ReadWriter that implements the Connection interface,
// providing methods to send and receive data over a socket connection without short read and short write errors.
type SocketConnection struct {
	conn io.ReadWriter
}

// newSocketConnection creates a new SocketConnection instance with the given io.ReadWriter.
func newSocketConnection(conn io.ReadWriter) *SocketConnection {
	return &SocketConnection{conn: conn}
}

// SendAll sends all bytes over the socket connection, ensuring that all data is sent without short write errors.
func (s *SocketConnection) SendAll(bytes []byte) error {
	return safe_socket.SendAll(s.conn, bytes)
}

// RecvAll receives the specified number of bytes from the socket connection, ensuring that all data is received
// without short read errors.
func (s *SocketConnection) RecvAll(size int) ([]byte, error) {
	return safe_socket.RecvAll(s.conn, size)
}
