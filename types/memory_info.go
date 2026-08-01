package types

// MemInfo is the system-wide memory picture, all values in KiB.
type MemInfo struct {
	TotalKB        uint64  `json:"total_kb"`
	FreeKB         uint64  `json:"free_kb"`
	AvailableKB    uint64  `json:"available_kb"`
	BuffersKB      uint64  `json:"buffers_kb"`
	CachedKB       uint64  `json:"cached_kb"`
	SReclaimKB     uint64  `json:"slab_reclaimable_kb"`
	SUnreclaimKB   uint64  `json:"slab_unreclaimable_kb"`
	ShmemKB        uint64  `json:"shmem_kb"`
	SwapTotalKB    uint64  `json:"swap_total_kb"`
	SwapFreeKB     uint64  `json:"swap_free_kb"`
	DirtyKB        uint64  `json:"dirty_kb"`
	AvailablePct   float64 `json:"available_pct"`
	ReclaimablePct float64 `json:"reclaimable_pct"` // cache+buffers+sreclaim as % of total
}
