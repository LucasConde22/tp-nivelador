package client

// Connection defines the interface for sending and receiving data over a connection,
// without short read and short write errors, and allowing for the use of different
// connection types (e.g., TCP, TLS).
type Connection interface {
	SendAll(bytes []byte) error
	RecvAll(size int) ([]byte, error)
}
