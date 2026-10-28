package ipc

// SharedBuffer is a shared-memory region that can be passed over a Unix socket.
type SharedBuffer interface {
	Bytes() []byte
	Fd() int
	Close() error
}

// CreateSharedBuffer allocates a shared-memory buffer of size bytes.
func CreateSharedBuffer(size int) (SharedBuffer, error) {
	return createSharedBuffer(size)
}
