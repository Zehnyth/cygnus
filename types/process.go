package types

// Proc is one process's correlated footprint.
// PssKB is the honest attribution number (shared pages divided among
// sharers). RssKB is kept for comparison — summing RSS across processes
// double-counts shared memory, which is the classic `ps --sort=-rss` bug.
type Proc struct {
	PID     int    `json:"pid"`
	User    string `json:"user"`
	Comm    string `json:"comm"`
	Cmdline string `json:"cmdline"`
	PssKB   uint64 `json:"pss_kb"`
	RssKB   uint64 `json:"rss_kb"`
	SwapKB  uint64 `json:"swap_kb"`
	Threads int    `json:"threads"`
	FDs     int    `json:"fds"`    // -1 if unreadable (needs root for other users)
	PssOK   bool   `json:"pss_ok"` // false => smaps_rollup unreadable, PssKB copied from RssKB
	// KernelThread is true for processes with no user address space (e.g.
	// kworker, ksoftirqd). They legitimately have no PSS, so they must not
	// be counted as "PSS unreadable" — that inflated the degraded count.
	KernelThread bool `json:"kernel_thread"`
}

// ProcStatus is the subset of /proc/<pid>/status collect cares about:
// the resolved username, thread count, and whether a VmRSS line was
// present (its absence marks a kernel thread).
type ProcStatus struct {
	User     string
	Threads  int
	HasVmRSS bool
}
