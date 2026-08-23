package safe_socket

import (
	"io"
)

//TODO: Complete with a short-read/short-write tolerant implementation

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
