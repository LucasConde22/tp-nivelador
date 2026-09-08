package safe_socket

import (
	"io"
)

// SendAll sends all bytes over the provided io.Writer, ensuring that all data is sent without short write errors.
func SendAll(socket io.Writer, bytes []byte) error {
	totalSent := 0

	for totalSent < len(bytes) {
		sent, err := socket.Write(bytes[totalSent:])

		if err != nil {
			return err
		}
		totalSent += sent
	}

	return nil
}

// RecvAll receives the specified number of bytes from the provided io.Reader, ensuring that all data is received
// without short read errors.
func RecvAll(socket io.Reader, size int) ([]byte, error) {
	buff := make([]byte, size)
	totalRecv := 0

	for totalRecv < size {
		recv, err := socket.Read(buff[totalRecv:])
		totalRecv += recv

		if err != nil {
			return nil, err
		}
	}

	return buff, nil
}
