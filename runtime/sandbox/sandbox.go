// Package sandbox implements Linux process-isolation primitives used by the
// renderer and broker processes: namespaces, no-new-privileges, capability
// dropping, rlimits, Landlock filesystem rules, cgroups, and seccomp filters.
package sandbox

// Rlimit is one resource limit.
type Rlimit struct {
	Resource int
	Current  uint64
	Maximum  uint64
}

// Landlock access rights.
const (
	LandlockExecute    uint64 = 1 << 0
	LandlockWriteFile  uint64 = 1 << 1
	LandlockReadFile   uint64 = 1 << 2
	LandlockReadDir    uint64 = 1 << 3
	LandlockRemoveDir  uint64 = 1 << 4
	LandlockRemoveFile uint64 = 1 << 5
	LandlockMakeChar   uint64 = 1 << 6
	LandlockMakeDir    uint64 = 1 << 7
	LandlockMakeReg    uint64 = 1 << 8
	LandlockMakeSock   uint64 = 1 << 9
	LandlockMakeFIFO   uint64 = 1 << 10
	LandlockMakeBlock  uint64 = 1 << 11
	LandlockMakeSym    uint64 = 1 << 12
	LandlockRefer      uint64 = 1 << 13
	LandlockTruncate   uint64 = 1 << 14
)

// LandlockRule grants access beneath Path.
type LandlockRule struct {
	Path   string
	Access uint64
}

// LandlockConfig configures a filesystem ruleset.
type LandlockConfig struct {
	HandledAccess uint64
	Rules         []LandlockRule
}

// SeccompConfig configures a syscall allow-list. Installation is opt-in.
type SeccompConfig struct {
	AllowedSyscalls []uint32
}

// Config describes the isolation steps to apply.
type Config struct {
	NoNewPrivileges  bool
	DropCapabilities bool
	Rlimits          []Rlimit
	UnshareFlags     int
	UserNamespace    bool
	Landlock         *LandlockConfig
	CgroupPath       string
	MemoryMax        int64
	PidsMax          int64
	Seccomp          *SeccompConfig
}

// Apply applies the configured sandbox steps in order. Unsupported steps
// return a typed error; callers decide whether to abort or fall back.
func Apply(config Config) error {
	return applyPlatform(config)
}
