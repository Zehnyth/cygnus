package collect

import (
	"bufio"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zehnyth/cygnus/types"
)

// ListProcs walks /proc once and returns every visible process.
// Run as root for complete PSS + FD data; degrades gracefully otherwise.
func ListProcs() ([]types.Proc, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	pageKB := uint64(os.Getpagesize() / 1024)
	uidCache := map[string]string{}

	var out []types.Proc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		if p, ok := readProc(filepath.Join("/proc", e.Name()), pid, pageKB, uidCache); ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// readProc collects one process's full record from its /proc/<pid>
// directory. Returns ok=false if the process vanished mid-walk (its
// comm file — the one thing that must exist — became unreadable).
func readProc(base string, pid int, pageKB uint64, uidCache map[string]string) (types.Proc, bool) {
	p := types.Proc{PID: pid, FDs: -1}

	b, err := os.ReadFile(filepath.Join(base, "comm"))
	if err != nil {
		return p, false // process vanished mid-walk; skip
	}
	p.Comm = strings.TrimSpace(string(b))

	if b, err := os.ReadFile(filepath.Join(base, "cmdline")); err == nil {
		p.Cmdline = strings.TrimRight(strings.ReplaceAll(string(b), "\x00", " "), " ")
	}

	st := readStatus(base, uidCache)
	p.User = st.User
	p.Threads = st.Threads
	// A process with no VmRSS line in status has no user address space
	// (e.g. kworker, ksoftirqd) — that's a kernel thread, not a failure.
	p.KernelThread = !st.HasVmRSS

	p.RssKB = readStatm(base, pageKB)

	// PSS + Swap from smaps_rollup (cheap single-file summary; needs
	// ptrace-level access, i.e. root for other users' processes).
	// Kernel threads have no address space, so this file is absent for
	// them — that's expected, not a permission failure.
	if !p.KernelThread {
		pss, swap, ok := readSmapsRollup(base)
		p.PssKB, p.SwapKB, p.PssOK = pss, swap, ok
		if !p.PssOK {
			p.PssKB = p.RssKB // genuinely degraded; flagged via PssOK=false
		}
	}

	p.FDs = readFDCount(base)

	return p, true
}

// readStatus parses /proc/<pid>/status for the uid (resolved to a
// username via uidCache), thread count, and whether a VmRSS line is
// present (absence marks a kernel thread).
func readStatus(base string, uidCache map[string]string) types.ProcStatus {
	var st types.ProcStatus

	f, err := os.Open(filepath.Join(base, "status"))
	if err != nil {
		return st
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "Uid:"):
			uid := strings.Fields(line)[1]
			if name, ok := uidCache[uid]; ok {
				st.User = name
			} else if u, err := user.LookupId(uid); err == nil {
				uidCache[uid] = u.Username
				st.User = u.Username
			} else {
				st.User = uid
			}
		case strings.HasPrefix(line, "Threads:"):
			st.Threads, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Threads:")))
		case strings.HasPrefix(line, "VmRSS:"):
			st.HasVmRSS = true
		}
	}
	return st
}

// readStatm returns RSS in KiB from /proc/<pid>/statm, which (unlike
// smaps_rollup) is always readable regardless of permissions.
func readStatm(base string, pageKB uint64) uint64 {
	b, err := os.ReadFile(filepath.Join(base, "statm"))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) < 2 {
		return 0
	}
	pages, _ := strconv.ParseUint(fields[1], 10, 64)
	return pages * pageKB
}

// readSmapsRollup returns PSS and swap in KiB from /proc/<pid>/smaps_rollup.
// ok is false when the file is unreadable (needs root for other users'
// processes) — callers fall back to RSS and flag the process as degraded.
func readSmapsRollup(base string) (pssKB, swapKB uint64, ok bool) {
	f, err := os.Open(filepath.Join(base, "smaps_rollup"))
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "Pss:"):
			pssKB = parseKB(line)
			ok = true
		case strings.HasPrefix(line, "Swap:"):
			swapKB = parseKB(line)
		}
	}
	return pssKB, swapKB, ok
}

// readFDCount returns the number of open file descriptors, or -1 if
// /proc/<pid>/fd is unreadable (needs root for other users' processes).
func readFDCount(base string) int {
	fds, err := os.ReadDir(filepath.Join(base, "fd"))
	if err != nil {
		return -1
	}
	return len(fds)
}

// parseKB extracts the numeric value from a "Label:   N kB" /proc line.
func parseKB(line string) uint64 {
	f := strings.Fields(line)
	if len(f) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(f[1], 10, 64)
	return v
}
