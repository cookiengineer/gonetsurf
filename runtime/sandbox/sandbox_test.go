//go:build linux

package sandbox

import (
	"os"
	"os/exec"
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
