//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestBuildSeccompFilter(t *testing.T) {
	program, err := BuildSeccompFilter(SeccompConfig{AllowedSyscalls: []uint32{0, 1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if len(program)%8 != 0 {
		t.Fatalf("program length = %d", len(program))
	}
	instructions := len(program) / 8
	if instructions < 4+2*3+1 {
		t.Errorf("instructions = %d", instructions)
	}
}

func TestSandboxHelper(t *testing.T) {
	if os.Getenv("GONETSURF_SANDBOX_HELPER") == "" {
		return
	}
	switch os.Getenv("GONETSURF_SANDBOX_MODE") {
	case "no-new-privs":
		if err := NoNewPrivileges(); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	case "rlimit":
		if err := SetRlimit(Rlimit{Resource: unix.RLIMIT_NOFILE, Current: 64, Maximum: 64}); err != nil {
			os.Exit(2)
		}
		var limit unix.Rlimit
		if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil || limit.Cur > 64 {
			os.Exit(3)
		}
		os.Exit(0)
	case "baseline":
		home, _ := os.UserHomeDir()
		probePath := filepath.Join(home, ".bashrc")
		_, probeExisted := os.Stat(probePath)
		if err := ApplyUnprivilegedBaseline(); err != nil {
			fmt.Fprintln(os.Stderr, "baseline:", err)
			os.Exit(2)
		}
		if _, err := os.ReadFile("/etc/passwd"); err != nil {
			os.Exit(3)
		}
		if err := os.WriteFile("/tmp/gonetsurf-sandbox-write-test", []byte("x"), 0o600); err == nil {
			os.Remove("/tmp/gonetsurf-sandbox-write-test")
			os.Exit(4)
		}
		if probeExisted == nil {
			if _, err := os.ReadFile(probePath); err == nil {
				os.Exit(5)
			}
		}
		os.Exit(0)
	case "unshare":
		if err := Unshare(unix.CLONE_NEWUSER | unix.CLONE_NEWNET | unix.CLONE_NEWNS); err != nil {
			fmt.Fprintln(os.Stderr, "unshare:", err)
			os.Exit(2)
		}
		os.Exit(0)
	case "netguard":
		if err := InstallNetworkSocketGuard(); err != nil {
			os.Exit(2)
		}
		inetFD, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
		if err == nil {
			unix.Close(inetFD)
			os.Exit(3)
		}
		unixFD, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
		if err != nil {
			os.Exit(4)
		}
		unix.Close(unixFD)
		os.Exit(0)
	case "landlock":
		err := ApplyLandlock(LandlockConfig{
			HandledAccess: LandlockReadFile | LandlockReadDir,
			Rules:         []LandlockRule{{Path: "/usr", Access: LandlockReadFile | LandlockReadDir}},
		})
		if err != nil {
			os.Exit(2)
		}
		if _, err := os.ReadFile("/etc/passwd"); err == nil {
			os.Exit(3)
		}
		os.Exit(0)
	}
	os.Exit(4)
}

func runHelper(t *testing.T, mode string) int {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=TestSandboxHelper")
	command.Env = append(os.Environ(), "GONETSURF_SANDBOX_HELPER=1", "GONETSURF_SANDBOX_MODE="+mode)
	err := command.Run()
	if err == nil {
		return 0
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	t.Fatalf("helper %s: %v", mode, err)
	return -1
}

func TestNoNewPrivilegesHelper(t *testing.T) {
	if code := runHelper(t, "no-new-privs"); code != 0 {
		t.Errorf("no-new-privs helper exit = %d", code)
	}
}

func TestRlimitHelper(t *testing.T) {
	if code := runHelper(t, "rlimit"); code != 0 {
		t.Errorf("rlimit helper exit = %d", code)
	}
}

func TestUnprivilegedBaselineHelper(t *testing.T) {
	code := runHelper(t, "baseline")
	switch code {
	case 0:
	case 2:
		t.Skip("baseline sandbox is unavailable in this environment")
	case 3:
		t.Error("baseline sandbox denied a read-only path")
	case 4:
		t.Error("baseline sandbox allowed a filesystem write")
	case 5:
		t.Error("baseline sandbox allowed a home-directory read")
	default:
		t.Errorf("baseline helper exit = %d", code)
	}
}

func TestProcAttrNamespaces(t *testing.T) {
	command := exec.Command("/bin/true")
	command.SysProcAttr = ProcAttr(NamespaceConfig{User: true, Network: true, Mount: true})
	if err := command.Run(); err != nil {
		t.Skipf("namespace spawn unavailable: %v", err)
	}
}

func TestUnshareHelper(t *testing.T) {
	code := runHelper(t, "unshare")
	switch code {
	case 0:
	case 2:
		t.Skip("user/network/mount namespaces are unavailable in this environment")
	default:
		t.Errorf("unshare helper exit = %d", code)
	}
}

func TestNetworkSocketGuardHelper(t *testing.T) {
	code := runHelper(t, "netguard")
	switch code {
	case 0:
	case 2:
		t.Skip("seccomp is unavailable in this environment")
	default:
		t.Errorf("netguard helper exit = %d", code)
	}
}

func TestLandlockHelper(t *testing.T) {
	code := runHelper(t, "landlock")
	switch code {
	case 0:
	case 2:
		t.Skip("Landlock is unavailable in this environment")
	case 3:
		t.Error("Landlock allowed reading /etc/passwd")
	default:
		t.Errorf("landlock helper exit = %d", code)
	}
}

func TestUnsupportedArchitectureFilter(t *testing.T) {
	if currentArchitecture() == "amd64" || currentArchitecture() == "arm64" {
		if _, err := BuildSeccompFilter(SeccompConfig{}); err != nil {
			t.Fatal(err)
		}
	}
}
