//go:build !linux

package sandbox

import "fmt"

func applyPlatform(config Config) error {
	return fmt.Errorf("sandbox: Linux isolation is not available on this platform")
}

// InstallNetworkSocketGuard is a no-op on non-Linux platforms.
func InstallNetworkSocketGuard() error {
	return nil
}

// ApplyUnprivilegedBaseline is a no-op on non-Linux platforms.
func ApplyUnprivilegedBaseline() error {
	return nil
}

// NamespaceConfig selects namespaces for a newly spawned process.
type NamespaceConfig struct {
	User    bool
	Network bool
	Mount   bool
	PID     bool
}
