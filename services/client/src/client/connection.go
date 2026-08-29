package client

type Connection interface {
	SendAll(bytes []byte) error
	RecvAll(size int) ([]byte, error)
}
