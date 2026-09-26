package version

import "fmt"

// Set via -ldflags at build/release time, e.g.:
//
//	-X github.com/aknEvrnky/pgway/internal/platform/version.Version=0.1.0
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns Version only (for OTel / agent registration).
func String() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

// Line formats a human-readable version line for CLI / -version flags.
func Line(binary string) string {
	return fmt.Sprintf("%s version %s (commit %s, built %s)", binary, String(), Commit, Date)
}
