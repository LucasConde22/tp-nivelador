package client

import (
	"io"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

type SocketConnection struct {
	conn io.ReadWriter
}

func newSocketConnection(conn io.ReadWriter) *SocketConnection {
	return &SocketConnection{conn: conn}
}

func (s *SocketConnection) SendAll(bytes []byte) error {
	return safe_socket.SendAll(s.conn, bytes)
}

func (s *SocketConnection) RecvAll(size int) ([]byte, error) {
	return safe_socket.RecvAll(s.conn, size)
}
