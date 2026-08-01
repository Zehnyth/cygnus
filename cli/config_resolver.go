package cli

import (
	"os"
	"path/filepath"

	"github.com/zehnyth/cygnus/services"
	"github.com/zehnyth/cygnus/types"
)

// resolveConfigPath picks the config file to load, in priority order:
// an explicit --config flag, then a per-user XDG override, then a
// system-wide file (the kind an RPM install would drop at /etc/cygnus).
// Returns "" if none exist, which makes LoadConfig fall back to defaults.
func resolveConfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}

	candidates := []string{}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "cygnus", "config.json"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".config", "cygnus", "config.json"))
	}
	candidates = append(candidates, "/etc/cygnus/config.json")

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// loadConfig resolves and loads the effective Config for this run.
func loadConfig() (types.Config, error) {
	return services.LoadConfig(resolveConfigPath(configPath))
}
