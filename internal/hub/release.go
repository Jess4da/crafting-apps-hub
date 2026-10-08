package hub

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Asset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	URL  string `json:"browser_download_url"`
}

type Release struct {
	Tag         string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Version strips the leading "v" from the tag.
func (r Release) Version() string { return strings.TrimPrefix(r.Tag, "v") }

const sumsAsset = "SHA256SUMS.txt"

var apiClient = &http.Client{Timeout: 30 * time.Second}

// apiBase is a var so tests can point it at httptest.
var apiBase = "https://api.github.com"

func latestRelease(ctx context.Context, repo string) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, Owner, repo), nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "crafting-apps-hub")
	// Unauthenticated API allows 60 req/h; a token lifts that to 5000.
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	res, err := apiClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusTooManyRequests {
			return Release{}, errors.New("GitHub rate limit reached; try again later or set GITHUB_TOKEN")
		}
		return Release{}, fmt.Errorf("GitHub %s: %s", repo, res.Status)
	}
	var rel Release
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return Release{}, fmt.Errorf("decode %s release: %w", repo, err)
	}
	return rel, nil
}

// pickAsset returns the portable build for goos/goarch, or false when the
// platform has no build. Portable archives install per-user without admin
// rights, so MSI/deb/rpm are deliberately ignored.
func pickAsset(rel Release, prefix, goos, goarch string) (Asset, bool) {
	v := rel.Version()
	var candidates []string
	switch goos {
	case "windows":
		arch := map[string]string{"amd64": "x64", "386": "x86", "arm64": "arm64"}[goarch]
		candidates = append(candidates, fmt.Sprintf("%s-%s-windows-%s-portable.zip", prefix, v, arch))
		if goarch == "arm64" { // Windows on ARM runs x64 binaries under emulation.
			candidates = append(candidates, fmt.Sprintf("%s-%s-windows-x64-portable.zip", prefix, v))
		}
	case "linux":
		arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[goarch]
		candidates = append(candidates, fmt.Sprintf("%s-%s-linux-%s.tar.gz", prefix, v, arch))
	case "freebsd":
		candidates = append(candidates, fmt.Sprintf("%s-%s-freebsd-x86_64.tar.gz", prefix, v))
	case "darwin":
		candidates = append(candidates, fmt.Sprintf("%s-%s-macos-universal.dmg", prefix, v))
	}
	for _, want := range candidates {
		for _, a := range rel.Assets {
			if a.Name == want {
				return a, true
			}
		}
	}
	return Asset{}, false
}

// parseSums reads "<hex>  <name>" lines (sha256sum format, optional '*').
func parseSums(r io.Reader) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 {
			out[strings.TrimPrefix(f[1], "*")] = strings.ToLower(f[0])
		}
	}
	return out
}

// newerVersion reports whether a is a higher dotted version than b.
// Pre-release suffixes ("-beta.1") are ignored for ordering.
func newerVersion(a, b string) bool {
	pa, pb := versionParts(a), versionParts(b)
	for i := range max(len(pa), len(pb)) {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func versionParts(v string) []int {
	v = strings.TrimPrefix(v, "v")
	v, _, _ = strings.Cut(v, "-")
	var out []int
	for _, p := range strings.Split(v, ".") {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}
