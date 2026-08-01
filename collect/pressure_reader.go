package collect

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/zehnyth/cygnus/types"
)

// ReadMemPressure reads and parses /proc/pressure/memory. Absence (old
// kernel, or PSI disabled) is not an error — it returns an unsupported PSI.
func ReadMemPressure() *types.PSI {
	f, err := os.Open("/proc/pressure/memory")
	if err != nil {
		return &types.PSI{} // PSI absent — report unsupported, not fatal
	}
	defer func() { _ = f.Close() }()
	return parsePSI(f)
}

// parsePSI is the pure parsing half of ReadMemPressure, split out so it
// can be exercised with fixture content instead of a real /proc file.
func parsePSI(r io.Reader) *types.PSI {
	p := &types.PSI{Supported: true}

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		var a10, a60 *float64
		switch fields[0] {
		case "some":
			a10, a60 = &p.SomeAvg10, &p.SomeAvg60
		case "full":
			a10, a60 = &p.FullAvg10, &p.FullAvg60
		default:
			continue
		}
		for _, kv := range fields[1:] {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				continue
			}
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			switch k {
			case "avg10":
				*a10 = val
			case "avg60":
				*a60 = val
			}
		}
	}
	return p
}
