package cmd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Release endpoints. Variables, not constants, so tests can point them at a
// local server.
var (
	releaseWebBase = "https://github.com"
	releaseAPIBase = "https://api.github.com"
)

// maxReleaseDownload caps an archive download. Release archives are ~10MB; the
// cap only stops a misbehaving server from filling memory.
const maxReleaseDownload = 200 << 20

// errNoChecksum is returned when a release publishes no checksum for an asset.
var errNoChecksum = errors.New("no checksum published")

// latestRelease returns the latest release version (without a leading "v").
//
// It reads the redirect of github.com/<repo>/releases/latest, which is not
// subject to the GitHub API's 60-requests-per-hour limit for unauthenticated
// callers (a whole office or CI fleet behind one IP shares that limit). The API
// is the fallback when the redirect cannot be read.
func latestRelease(ctx context.Context) (string, error) {
	v, err := latestFromRedirect(ctx)
	if err == nil {
		return v, nil
	}
	v, apiErr := latestFromAPI(ctx)
	if apiErr == nil {
		return v, nil
	}
	return "", fmt.Errorf("%v (API fallback: %v)", err, apiErr)
}

func latestFromRedirect(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, releaseWebBase+"/"+updateRepo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "encrata-cli")
	client := &http.Client{
		Timeout:       15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	if resp.StatusCode < 300 || resp.StatusCode > 399 || loc == "" {
		return "", fmt.Errorf("releases/latest did not redirect (HTTP %d)", resp.StatusCode)
	}
	i := strings.LastIndex(loc, "/tag/")
	if i < 0 {
		return "", fmt.Errorf("unexpected release redirect %q", loc)
	}
	tag := normalizeVersion(loc[i+len("/tag/"):])
	if _, ok := parseVersion(tag); !ok {
		return "", fmt.Errorf("unexpected release tag %q", tag)
	}
	return tag, nil
}

func latestFromAPI(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPIBase+"/repos/"+updateRepo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "encrata-cli")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("GitHub API rate limit reached; try again later")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no release tag found")
	}
	return normalizeVersion(rel.TagName), nil
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// ── semantic versions ───────────────────────────────────────────────────────

type semver struct {
	core [3]int
	pre  string // "" for a release; releases sort above their pre-releases
}

// parseVersion parses "1.2.3", "v1.2.3" or "1.2.3-rc.1" (build metadata after
// "+" is ignored). ok is false for anything else, including "dev".
func parseVersion(s string) (semver, bool) {
	s = normalizeVersion(s)
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var v semver
	core := s
	if i := strings.IndexByte(s, '-'); i >= 0 {
		core, v.pre = s[:i], s[i+1:]
		if v.pre == "" {
			return semver{}, false
		}
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, false
		}
		v.core[i] = n
	}
	return v, true
}

// compareVersions returns -1, 0 or 1 as a is older than, equal to, or newer
// than b. ok is false when either is not a semantic version.
func compareVersions(a, b string) (cmp int, ok bool) {
	va, okA := parseVersion(a)
	vb, okB := parseVersion(b)
	if !okA || !okB {
		return 0, false
	}
	for i := 0; i < 3; i++ {
		if va.core[i] != vb.core[i] {
			if va.core[i] < vb.core[i] {
				return -1, true
			}
			return 1, true
		}
	}
	switch {
	case va.pre == vb.pre:
		return 0, true
	case va.pre == "":
		return 1, true
	case vb.pre == "":
		return -1, true
	case va.pre < vb.pre:
		return -1, true
	default:
		return 1, true
	}
}

// ── downloads and checksums ─────────────────────────────────────────────────

// releaseAssetURL is the download URL of one asset of release v<version>.
func releaseAssetURL(version, asset string) string {
	return fmt.Sprintf("%s/%s/releases/download/v%s/%s", releaseWebBase, updateRepo, version, asset)
}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "encrata-cli")
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxReleaseDownload {
		return nil, fmt.Errorf("download is larger than %d MB", maxReleaseDownload>>20)
	}
	return data, nil
}

// checksumFor returns the hex SHA-256 listed for asset in a GoReleaser
// checksums.txt ("<sha256>  <file name>" per line).
func checksumFor(checksums []byte, asset string) (string, error) {
	sc := bufio.NewScanner(bytes.NewReader(checksums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			sum := strings.ToLower(fields[0])
			if len(sum) != sha256.Size*2 {
				return "", fmt.Errorf("malformed checksum for %s", asset)
			}
			return sum, nil
		}
	}
	return "", fmt.Errorf("%w for %s", errNoChecksum, asset)
}

// verifyChecksum checks data against the release's checksums.txt. A release
// without one fails closed: an unverified binary is never installed.
func verifyChecksum(ctx context.Context, version, asset string, data []byte) error {
	sums, err := download(ctx, releaseAssetURL(version, "checksums.txt"))
	if err != nil {
		return fmt.Errorf("could not download checksums.txt for v%s: %w", version, err)
	}
	want, err := checksumFor(sums, asset)
	if err != nil {
		return err
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("checksum mismatch for %s: the download is corrupt or was tampered with, nothing was changed", asset)
	}
	return nil
}
