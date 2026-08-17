//go:build !windows

package cmd

// detachedNpmUpdate is a no-op off Windows: npm can replace and delete a
// running binary in place there, so the normal synchronous update is used.
func detachedNpmUpdate(_ string) (bool, error) {
	return false, nil
}
