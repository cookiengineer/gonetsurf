//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func applyPlatform(config Config) error {
	for _, limit := range config.Rlimits {
		if err := SetRlimit(limit); err != nil {
			return err
		}
	}
	if config.NoNewPrivileges {
		if err := NoNewPrivileges(); err != nil {
			return err
		}
	}
	if config.DropCapabilities {
		if err := DropCapabilities(); err != nil {
			return err
		}
	}
	unshareFlags := config.UnshareFlags
	if config.UserNamespace {
		unshareFlags |= unix.CLONE_NEWUSER
	}
	if unshareFlags != 0 {
		if err := Unshare(unshareFlags); err != nil {
			return err
		}
	}
	if config.Landlock != nil {
		if err := ApplyLandlock(*config.Landlock); err != nil {
			return err
		}
	}
	if config.CgroupPath != "" {
		if err := SetCgroupLimits(config.CgroupPath, config.MemoryMax, config.PidsMax); err != nil {
			return err
		}
	}
	if config.Seccomp != nil {
		if err := InstallSeccomp(*config.Seccomp); err != nil {
			return err
		}
	}
	return nil
}

// NoNewPrivileges sets PR_SET_NO_NEW_PRIVS.
func NoNewPrivileges() error {
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("sandbox: no_new_privs: %w", err)
	}
	return nil
}

// SetRlimit applies one rlimit.
func SetRlimit(limit Rlimit) error {
	resource := limit.Resource
	current := limit.Current
	maximum := limit.Maximum
	if err := unix.Setrlimit(resource, &unix.Rlimit{Cur: current, Max: maximum}); err != nil {
		return fmt.Errorf("sandbox: setrlimit %d: %w", resource, err)
	}
	return nil
}

// DropCapabilities clears the effective, permitted, inheritable, ambient, and
// bounding capability sets.
func DropCapabilities() error {
	header := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3, Pid: 0}
	data := [2]unix.CapUserData{}
	if err := unix.Capset(&header, &data[0]); err != nil {
		return fmt.Errorf("sandbox: capset: %w", err)
	}
	unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0)
	for capability := 0; capability <= unix.CAP_LAST_CAP; capability++ {
		unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(capability), 0, 0, 0)
	}
	return nil
}

// Unshare creates the requested namespaces. User namespaces are mapped to the
// current UID/GID. PID namespaces take effect for subsequently forked children.
func Unshare(flags int) error {
	userFlags := flags & unix.CLONE_NEWUSER
	remaining := flags &^ unix.CLONE_NEWUSER
	if userFlags != 0 {
		if err := unix.Unshare(unix.CLONE_NEWUSER); err != nil {
			return fmt.Errorf("sandbox: unshare user: %w", err)
		}
		if err := writeFile("/proc/self/setgroups", "deny"); err != nil {
			return err
		}
		uidMap := fmt.Sprintf("%d %d 1", os.Geteuid(), os.Geteuid())
		gidMap := fmt.Sprintf("%d %d 1", os.Getegid(), os.Getegid())
		if err := writeFile("/proc/self/uid_map", uidMap); err != nil {
			return err
		}
		if err := writeFile("/proc/self/gid_map", gidMap); err != nil {
			return err
		}
	}
	if remaining != 0 {
		if err := unix.Unshare(remaining); err != nil {
			return fmt.Errorf("sandbox: unshare: %w", err)
		}
	}
	return nil
}

func writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("sandbox: write %s: %w", path, err)
	}
	return nil
}

type landlockRulesetAttr struct {
	HandledAccessFS uint64
}

type landlockPathBeneathAttr struct {
	AllowedAccess uint64
	ParentFD      int32
	Pad           int32
}

// ApplyLandlock installs and enforces a filesystem ruleset.
func ApplyLandlock(config LandlockConfig) error {
	handled := config.HandledAccess
	if handled == 0 {
		handled = LandlockExecute | LandlockWriteFile | LandlockReadFile | LandlockReadDir |
			LandlockRemoveDir | LandlockRemoveFile | LandlockMakeChar | LandlockMakeDir |
			LandlockMakeReg | LandlockMakeSock | LandlockMakeFIFO | LandlockMakeBlock |
			LandlockMakeSym | LandlockRefer | LandlockTruncate
	}
	attr := landlockRulesetAttr{HandledAccessFS: handled}
	rulesetFD, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		return fmt.Errorf("sandbox: landlock_create_ruleset: %w", errno)
	}
	defer unix.Close(int(rulesetFD))
	for _, rule := range config.Rules {
		pathFD, err := unix.Open(rule.Path, unix.O_PATH|unix.O_CLOEXEC, 0)
		if err != nil {
			return fmt.Errorf("sandbox: open %s: %w", rule.Path, err)
		}
		ruleAttr := landlockPathBeneathAttr{AllowedAccess: rule.Access & handled, ParentFD: int32(pathFD)}
		_, _, errno := unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, rulesetFD, 1, uintptr(unsafe.Pointer(&ruleAttr)), 0, 0, 0)
		unix.Close(pathFD)
		if errno != 0 {
			return fmt.Errorf("sandbox: landlock_add_rule: %w", errno)
		}
	}
	if err := NoNewPrivileges(); err != nil {
		return err
	}
	if _, _, errno := unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, rulesetFD, 0, 0); errno != 0 {
		return fmt.Errorf("sandbox: landlock_restrict_self: %w", errno)
	}
	return nil
}

// SetCgroupLimits creates cgroup v2 files for memory and pid limits.
func SetCgroupLimits(path string, memoryMax, pidsMax int64) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("sandbox: create cgroup: %w", err)
	}
	if memoryMax > 0 {
		if err := os.WriteFile(filepath.Join(path, "memory.max"), []byte(strconv.FormatInt(memoryMax, 10)), 0o644); err != nil {
			return fmt.Errorf("sandbox: memory.max: %w", err)
		}
	}
	if pidsMax > 0 {
		if err := os.WriteFile(filepath.Join(path, "pids.max"), []byte(strconv.FormatInt(pidsMax, 10)), 0o644); err != nil {
			return fmt.Errorf("sandbox: pids.max: %w", err)
		}
	}
	return nil
}

type seccompData struct {
	Number int32
	Arch   uint32
	IP     uint64
	Args   [6]uint64
}

type bpfInstruction struct {
	Code uint16
	JT   uint8
	JF   uint8
	K    uint32
}

type sockFilter struct {
	Length uint16
	Filter *bpfInstruction
}

const (
	bpfLoadWordAbsolute = 0x20
	bpfJumpEqual        = 0x15
	bpfReturn           = 0x06
	seccompAllow        = 0x7fff0000
	seccompKill         = 0x80000000
	seccompErrno        = 0x00050000
	archX86_64          = 0xc000003e
	archAArch64         = 0xc00000b7
)

// BuildSeccompFilter builds a syscall allow-list program for testing and
// installation.
func BuildSeccompFilter(config SeccompConfig) ([]byte, error) {
	architecture := uint32(0)
	switch currentArchitecture() {
	case "amd64":
		architecture = archX86_64
	case "arm64":
		architecture = archAArch64
	default:
		return nil, fmt.Errorf("sandbox: unsupported architecture")
	}
	instructions := []bpfInstruction{
		{Code: bpfLoadWordAbsolute, K: 4},
		{Code: bpfJumpEqual, K: architecture, JT: 0, JF: 1},
		{Code: bpfReturn, K: seccompKill},
		{Code: bpfLoadWordAbsolute, K: 0},
	}
	for _, syscallNumber := range config.AllowedSyscalls {
		instructions = append(instructions,
			bpfInstruction{Code: bpfJumpEqual, K: syscallNumber, JT: 0, JF: 1},
			bpfInstruction{Code: bpfReturn, K: seccompAllow},
		)
	}
	instructions = append(instructions, bpfInstruction{Code: bpfReturn, K: seccompErrno | uint32(syscall.EPERM)})
	program := make([]byte, len(instructions)*8)
	for index, instruction := range instructions {
		offset := index * 8
		program[offset] = byte(instruction.Code)
		program[offset+1] = byte(instruction.Code >> 8)
		program[offset+2] = instruction.JT
		program[offset+3] = instruction.JF
		program[offset+4] = byte(instruction.K)
		program[offset+5] = byte(instruction.K >> 8)
		program[offset+6] = byte(instruction.K >> 16)
		program[offset+7] = byte(instruction.K >> 24)
	}
	return program, nil
}

// InstallSeccomp installs a syscall allow-list. The process must already have
// no_new_privs or CAP_SYS_ADMIN.
func InstallSeccomp(config SeccompConfig) error {
	program, err := BuildSeccompFilter(config)
	if err != nil {
		return err
	}
	instructions := make([]bpfInstruction, len(program)/8)
	for index := range instructions {
		offset := index * 8
		instructions[index] = bpfInstruction{
			Code: uint16(program[offset]) | uint16(program[offset+1])<<8,
			JT:   program[offset+2],
			JF:   program[offset+3],
			K:    uint32(program[offset+4]) | uint32(program[offset+5])<<8 | uint32(program[offset+6])<<16 | uint32(program[offset+7])<<24,
		}
	}
	filter := sockFilter{Length: uint16(len(instructions)), Filter: &instructions[0]}
	if err := NoNewPrivileges(); err != nil {
		return err
	}
	if err := unix.Prctl(unix.PR_SET_SECCOMP, unix.SECCOMP_MODE_FILTER, uintptr(unsafe.Pointer(&filter)), 0, 0); err != nil {
		return fmt.Errorf("sandbox: seccomp: %w", err)
	}
	return nil
}

func currentArchitecture() string {
	return architectureName
}
