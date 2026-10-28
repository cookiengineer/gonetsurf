//go:build !linux

package x11

import "fmt"

func createShmSegment(size int) (int, []byte, error) {
	return 0, nil, fmt.Errorf("x11: MIT-SHM is only supported on Linux")
}

func destroyShmSegment(identifier int, data []byte) {}
