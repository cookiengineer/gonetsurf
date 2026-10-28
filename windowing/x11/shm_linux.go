//go:build linux

package x11

import "golang.org/x/sys/unix"

func createShmSegment(size int) (int, []byte, error) {
	identifier, err := unix.SysvShmGet(unix.IPC_PRIVATE, size, unix.IPC_CREAT|0o600)
	if err != nil {
		return 0, nil, err
	}
	data, err := unix.SysvShmAttach(identifier, 0, 0)
	if err != nil {
		unix.SysvShmCtl(identifier, unix.IPC_RMID, nil)
		return 0, nil, err
	}
	return identifier, data, nil
}

func destroyShmSegment(identifier int, data []byte) {
	if data != nil {
		unix.SysvShmDetach(data)
	}
	if identifier != 0 {
		unix.SysvShmCtl(identifier, unix.IPC_RMID, nil)
	}
}
