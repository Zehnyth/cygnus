// Package utils holds small, stateless helpers with no dependency on
// types or any other project package — pure functions only.
package utils

import "fmt"

// GB converts KiB to a GiB float, e.g. for "12.3 GB" style output.
func GB(kb uint64) float64 { return float64(kb) / 1048576 }

// MB converts KiB to a MiB float, e.g. for per-process "340 MB" output.
func MB(kb uint64) float64 { return float64(kb) / 1024 }

// FDs renders an FD count, or "?" when unreadable (FDs == -1, e.g. not root).
func FDs(n int) string {
	if n < 0 {
		return "?"
	}
	return fmt.Sprintf("%d", n)
}
