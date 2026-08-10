package cmd

import (
	"regexp"
	"strings"

	"github.com/Encratahq/cli/internal/password"
	"github.com/spf13/cobra"
)

var sha1Regex = regexp.MustCompile(`^[A-F0-9]{40}$`)

// gatherSHA1s collects de-duplicated, validated SHA-1 hashes from --sha1s and
// --sha1-file. It never accepts plaintext passwords.
func gatherSHA1s(cmd *cobra.Command) ([]string, error) {
	sha1s, _ := cmd.Flags().GetStringSlice("sha1s")
	filePath, _ := cmd.Flags().GetString("sha1-file")

	seen := map[string]bool{}
	out := make([]string, 0)
	appendHash := func(hash string) error {
		hash = strings.ToUpper(strings.TrimSpace(hash))
		if hash == "" {
			return nil
		}
		if !sha1Regex.MatchString(hash) {
			return friendlyFormatError(cmd, "invalid SHA-1 hash; expected 40 hex characters")
		}
		if !seen[hash] {
			seen[hash] = true
			out = append(out, hash)
		}
		return nil
	}

	for _, hash := range sha1s {
		if err := appendHash(hash); err != nil {
			return nil, err
		}
	}
	if filePath != "" {
		lines, err := readLines(filePath)
		if err != nil {
			return nil, err
		}
		for _, hash := range lines {
			if err := appendHash(hash); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// gatherPasswordSHA1s collects SHA-1 hashes from --sha1s/--sha1-file plus any
// plaintext passwords from --password-file (hashed locally, never uploaded).
func gatherPasswordSHA1s(cmd *cobra.Command) ([]string, error) {
	out, err := gatherSHA1s(cmd)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(out))
	for _, h := range out {
		seen[h] = true
	}

	pwFile, _ := cmd.Flags().GetString("password-file")
	if pwFile != "" {
		lines, err := readLines(pwFile)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			buf := []byte(line)
			h := password.Hash(buf)
			password.Zero(buf)
			if !seen[h] {
				seen[h] = true
				out = append(out, h)
			}
		}
	}
	return out, nil
}
