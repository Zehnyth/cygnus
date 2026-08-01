package types

// PSI mirrors /proc/pressure/memory. "Some" = at least one task stalled,
// "Full" = all non-idle tasks stalled (the genuinely bad one).
type PSI struct {
	SomeAvg10 float64 `json:"some_avg10"`
	SomeAvg60 float64 `json:"some_avg60"`
	FullAvg10 float64 `json:"full_avg10"`
	FullAvg60 float64 `json:"full_avg60"`
	Supported bool    `json:"supported"`
}
