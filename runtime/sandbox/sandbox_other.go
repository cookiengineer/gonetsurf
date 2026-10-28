//go:build !linux

package sandbox

import "fmt"

func applyPlatform(config Config) error {
	return fmt.Errorf("sandbox: Linux isolation is not available on this platform")
}
