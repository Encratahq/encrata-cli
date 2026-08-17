// Package version is the single source of truth for the CLI's build metadata.
// These vars are overridden at release time via -ldflags, e.g.:
//
//	-X github.com/Encratahq/cli/internal/version.Version={{.Version}}
//	-X github.com/Encratahq/cli/internal/version.Commit={{.Commit}}
//	-X github.com/Encratahq/cli/internal/version.Date={{.Date}}
package version

var (
	// Version is the semantic version (without a leading "v"). Bumped per
	// release; a new feature is a MINOR bump.
	Version = "0.11.1"
	// Commit is the short git SHA the binary was built from.
	Commit = "none"
	// Date is the build timestamp (RFC3339) set by the release pipeline.
	Date = "unknown"
)
