package hub

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Asset struct {
	Name string
	Size int64 // 0 when unknown
	URL  string
}

// Release is what the hub knows about a repo's latest release. It is built
// from github.com web endpoints only (no REST API, so no 60/h rate limit):
// the /releases/latest redirect gives the tag, and the release's
// SHA256SUMS.txt lists every asset together with its checksum.
type Release struct {
	Tag     string
	HTMLURL string
	Assets  []Asset
	Sums    map[string]string // asset name → lowercase hex sha256
}

// Version strips the leading "v" from the tag.
func (r Release) Version() string { return strings.TrimPrefix(r.Tag, "v") }

const sumsAsset = "SHA256SUMS.txt"

// ghBase is a var so tests can point it at httptest.
var ghBase = "https://github.com"

// noRedirect reads the /releases/latest redirect instead of following it.
var noRedirect = &http.Client{
	Timeout:       30 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

var webClient = &http.Client{Timeout: 30 * time.Second}

func latestRelease(ctx context.Context, repo string) (Release, error) {
	base := fmt.Sprintf("%s/%s/%s/releases", ghBase, Owner, repo)
	res, err := get(ctx, noRedirect, http.MethodGet, base+"/latest")
	if err != nil {
		return Release{}, err
	}
	res.Body.Close()
	if res.StatusCode != http.StatusFound && res.StatusCode != http.StatusMovedPermanently {
		return Release{}, fmt.Errorf("%s: no published release (%s)", repo, res.Status)
	}
	loc, err := res.Location()
	if err != nil {
		return Release{}, fmt.Errorf("%s: %w", repo, err)
	}
	_, tag, ok := strings.Cut(loc.Path, "/releases/tag/")
	if !ok || tag == "" {
		return Release{}, fmt.Errorf("%s: unexpected latest-release redirect %s", repo, loc)
	}
	if tag, err = url.PathUnescape(tag); err != nil {
		return Release{}, err
	}

	dl := base + "/download/" + url.PathEscape(tag) + "/"
	res, err = get(ctx, webClient, http.MethodGet, dl+sumsAsset)
	if err != nil {
		return Release{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		// Refuse unverifiable releases rather than install something unchecked.
		return Release{}, fmt.Errorf("%s %s has no %s (%s)", repo, tag, sumsAsset, res.Status)
	}
	rel := Release{Tag: tag, HTMLURL: base + "/tag/" + url.PathEscape(tag), Sums: parseSums(io.LimitReader(res.Body, 1<<20))}
	for name := range rel.Sums {
		rel.Assets = append(rel.Assets, Asset{Name: name, URL: dl + url.PathEscape(name)})
	}
	return rel, nil
}

// assetSize asks for the download size without downloading (0 on failure;
// the size is only shown in the UI).
func assetSize(ctx context.Context, a Asset) int64 {
	res, err := get(ctx, webClient, http.MethodHead, a.URL)
	if err != nil {
		return 0
	}
	res.Body.Close()
	return max(res.ContentLength, 0)
}

func get(ctx context.Context, c *http.Client, method, u string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "crafting-apps-hub")
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusTooManyRequests {
		res.Body.Close()
		return nil, errors.New("GitHub is throttling requests; try again in a few minutes")
	}
	return res, nil
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
