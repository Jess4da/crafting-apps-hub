package hub

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func rel(tag string, names ...string) Release {
	r := Release{Tag: tag}
	for _, n := range names {
		r.Assets = append(r.Assets, Asset{Name: n})
	}
	return r
}

func TestPickAsset(t *testing.T) {
	full := rel("v0.3.0",
		"pdfcraft-0.3.0-windows-x64-portable.zip", "pdfcraft-0.3.0-windows-x64.msi",
		"pdfcraft-0.3.0-windows-arm64-portable.zip", "pdfcraft-0.3.0-windows-x86-portable.zip",
		"pdfcraft-0.3.0-linux-x86_64.tar.gz", "pdfcraft-0.3.0-linux-aarch64.tar.gz",
		"pdfcraft-0.3.0-linux-x86_64.deb", "pdfcraft-0.3.0-macos-universal.dmg",
		"pdfcraft-cli-0.3.0-macos-universal.zip",
	)
	noArm := rel("v0.2.1", "x-0.2.1-windows-x64-portable.zip")
	cases := []struct {
		r          Release
		prefix, os string
		arch, want string
	}{
		{full, "pdfcraft", "windows", "amd64", "pdfcraft-0.3.0-windows-x64-portable.zip"},
		{full, "pdfcraft", "windows", "arm64", "pdfcraft-0.3.0-windows-arm64-portable.zip"},
		{full, "pdfcraft", "windows", "386", "pdfcraft-0.3.0-windows-x86-portable.zip"},
		{full, "pdfcraft", "linux", "amd64", "pdfcraft-0.3.0-linux-x86_64.tar.gz"},
		{full, "pdfcraft", "linux", "arm64", "pdfcraft-0.3.0-linux-aarch64.tar.gz"},
		{full, "pdfcraft", "darwin", "arm64", "pdfcraft-0.3.0-macos-universal.dmg"},
		{noArm, "x", "windows", "arm64", "x-0.2.1-windows-x64-portable.zip"}, // x64 emulation fallback
		{full, "pdfcraft", "freebsd", "amd64", ""},                           // not in this release
		{full, "photocraft", "windows", "amd64", ""},                         // wrong prefix
	}
	for _, c := range cases {
		got, ok := pickAsset(c.r, c.prefix, c.os, c.arch)
		if got.Name != c.want || ok != (c.want != "") {
			t.Errorf("%s/%s %s: got %q,%v want %q", c.os, c.arch, c.prefix, got.Name, ok, c.want)
		}
	}
}

func TestNewerVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.3.0", "0.2.9", true},
		{"0.10.0", "0.9.0", true}, // numeric, not lexical
		{"1.0", "1.0.0", false},
		{"1.0.1", "1.0", true},
		{"v0.2.1", "0.2.1", false},
		{"0.2.0", "0.2.1", false},
		{"0.3.0-beta.1", "0.2.0", true},
	}
	for _, c := range cases {
		if got := newerVersion(c.a, c.b); got != c.want {
			t.Errorf("newerVersion(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestParseSums(t *testing.T) {
	m := parseSums(strings.NewReader("ABC  a.zip\ndef *b.tar.gz\n\ngarbage\n"))
	if m["a.zip"] != "abc" || m["b.tar.gz"] != "def" || len(m) != 2 {
		t.Fatalf("got %v", m)
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "evil.zip")
	writeZip(t, p, map[string]string{"../escape.txt": "x"})
	dst := filepath.Join(dir, "out")
	os.Mkdir(dst, 0o755)
	if err := extract(p, dst); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := os.Stat(filepath.Join(dir, "escape.txt")); err == nil {
		t.Fatal("file escaped target dir")
	}
}

func writeZip(t *testing.T, path string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	zw.Close()
	if path != "" {
		os.WriteFile(path, buf.Bytes(), 0o644)
	}
	return buf.Bytes()
}

// fakeGitHub serves a PdfCraft release from a fake github.com. sum overrides
// the checksum file contents ("" = correct sum).
func fakeGitHub(t *testing.T, version string, zipBytes []byte, sum string) *fakeGH {
	return fakeRelease(t, "pdfcraft", "pdfcraft", version, zipBytes, sum)
}

type fakeGH struct{ heads, sums atomic.Int32 }

// fakeRelease mimics the github.com web endpoints the hub uses: the
// /releases/latest redirect, SHA256SUMS.txt and the asset (GET and HEAD).
func fakeRelease(t *testing.T, repo, prefix, version string, zipBytes []byte, sum string) *fakeGH {
	t.Helper()
	asset := fmt.Sprintf("%s-%s-windows-x64-portable.zip", prefix, version)
	if sum == "" {
		h := sha256.Sum256(zipBytes)
		sum = hex.EncodeToString(h[:])
	}
	f := &fakeGH{}
	base := "/storytold/" + repo + "/releases"
	dl := base + "/download/v" + version + "/"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case base + "/latest":
			http.Redirect(w, r, base+"/tag/v"+version, http.StatusFound)
		case dl + sumsAsset:
			f.sums.Add(1)
			fmt.Fprintf(w, "%s  %s\n%s  other-file.zip\n", sum, asset, strings.Repeat("a", 64))
		case dl + asset:
			if r.Method == http.MethodHead {
				f.heads.Add(1)
			}
			http.ServeContent(w, r, asset, time.Time{}, bytes.NewReader(zipBytes))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	old := ghBase
	ghBase = srv.URL
	t.Cleanup(func() { ghBase = old })
	return f
}

func newTestHub(t *testing.T, data string) *Hub {
	t.Helper()
	h, err := New(data, filepath.Join(data, "apps"), func(Progress) {})
	if err != nil {
		t.Fatal(err)
	}
	h.goos, h.goarch = "windows", "amd64"
	// Never touch the real user profile from tests.
	u := filepath.Join(data, "user")
	h.base = baseDirs{config: filepath.Join(u, "Roaming"), data: filepath.Join(u, "Roaming"), cache: filepath.Join(u, "Local"), temp: filepath.Join(u, "Temp")}
	return h
}

func TestInstallUpdateUninstall(t *testing.T) {
	data := t.TempDir()
	h := newTestHub(t, data)
	ctx := context.Background()

	fakeGitHub(t, "0.1.0", writeZip(t, "", map[string]string{"pdfcraft-0.1.0-windows-x64-portable/pdfcraft.exe": "v1"}), "")
	s, err := h.Install(ctx, "pdfcraft")
	if err != nil {
		t.Fatal(err)
	}
	if s.Installed == nil || s.Installed.Version != "0.1.0" || s.UpdateAvailable {
		t.Fatalf("after install: %+v", s)
	}
	exe := s.Installed.Exe
	if b, _ := os.ReadFile(exe); string(b) != "v1" {
		t.Fatalf("exe content %q", b)
	}

	// New release appears: Refresh flags the update, Install replaces files.
	fakeGitHub(t, "0.2.0", writeZip(t, "", map[string]string{"pdfcraft-0.2.0-windows-x64-portable/pdfcraft.exe": "v2"}), "")
	h.Refresh(ctx)
	if st := find(h.List(), "pdfcraft"); !st.UpdateAvailable || st.LatestVersion != "0.2.0" {
		t.Fatalf("expected update flag: %+v", st)
	}
	s, err = h.Install(ctx, "pdfcraft")
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(s.Installed.Exe); string(b) != "v2" || s.UpdateAvailable {
		t.Fatalf("after update: %q %+v", b, s)
	}
	if _, err := os.Stat(exe); err == nil && exe != s.Installed.Exe {
		t.Fatal("old version left behind")
	}

	// State survives a restart.
	if st := find(newTestHub(t, data).List(), "pdfcraft"); st.Installed == nil || st.Installed.Version != "0.2.0" {
		t.Fatalf("not persisted: %+v", st)
	}

	if _, err := h.Uninstall("pdfcraft", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Installed.Dir); !os.IsNotExist(err) {
		t.Fatal("install dir not removed")
	}
	if find(h.List(), "pdfcraft").Installed != nil {
		t.Fatal("still marked installed")
	}
}

func TestChecksumMismatchKeepsPreviousInstall(t *testing.T) {
	data := t.TempDir()
	h := newTestHub(t, data)
	ctx := context.Background()
	fakeGitHub(t, "0.1.0", writeZip(t, "", map[string]string{"p/pdfcraft.exe": "v1"}), "")
	if _, err := h.Install(ctx, "pdfcraft"); err != nil {
		t.Fatal(err)
	}
	fakeGitHub(t, "0.2.0", writeZip(t, "", map[string]string{"p/pdfcraft.exe": "evil"}), strings.Repeat("0", 64))
	if _, err := h.Install(ctx, "pdfcraft"); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("want checksum error, got %v", err)
	}
	st := find(h.List(), "pdfcraft")
	if b, _ := os.ReadFile(st.Installed.Exe); st.Installed.Version != "0.1.0" || string(b) != "v1" {
		t.Fatalf("previous install damaged: %+v %q", st.Installed, b)
	}
	if left, _ := filepath.Glob(filepath.Join(data, "apps", ".tmp-*")); len(left) != 0 {
		t.Fatalf("temp dirs left: %v", left)
	}
}

// A portable hub folder (hub-data + apps side by side) keeps working after it
// is moved, e.g. a USB stick mounted under another drive letter.
func TestPortableFolderCanMove(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "hub")
	open := func(root string) *Hub {
		h, err := New(filepath.Join(root, "hub-data"), filepath.Join(root, "apps"), func(Progress) {})
		if err != nil {
			t.Fatal(err)
		}
		h.goos, h.goarch = "windows", "amd64"
		return h
	}
	fakeGitHub(t, "0.1.0", writeZip(t, "", map[string]string{"p/pdfcraft.exe": "v1"}), "")
	if _, err := open(root).Install(context.Background(), "pdfcraft"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "hub-data", "state.json")); strings.Contains(string(b), strings.ReplaceAll(tmp, `\`, `\\`)) {
		t.Fatalf("state.json holds absolute paths:\n%s", b)
	}

	moved := filepath.Join(tmp, "elsewhere")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	h := open(moved)
	in := find(h.List(), "pdfcraft").Installed
	if in == nil || !strings.HasPrefix(in.Exe, moved) {
		t.Fatalf("after move: %+v", in)
	}
	if got := h.GetSettings().InstallRoot; got != filepath.Join(moved, "apps") {
		t.Fatalf("install root %q", got)
	}
}

func TestCleanUninstallRemovesOnlyAppData(t *testing.T) {
	data := t.TempDir()
	h := newTestHub(t, data)
	ctx := context.Background()
	fakeGitHub(t, "0.1.0", writeZip(t, "", map[string]string{"p/pdfcraft.exe": "v1"}), "")
	if _, err := h.Install(ctx, "pdfcraft"); err != nil {
		t.Fatal(err)
	}
	mk := func(p string) string {
		os.MkdirAll(p, 0o755)
		os.WriteFile(filepath.Join(p, "f"), []byte("x"), 0o644)
		return p
	}
	ours := []string{
		mk(filepath.Join(h.base.config, "PdfCraft", "data")), // eframe storage
		mk(filepath.Join(h.base.cache, "PdfCraft", "Recovery")),
		mk(filepath.Join(h.base.cache, "PrintCraft")), // legacy name
	}
	keep := []string{
		mk(filepath.Join(h.base.config, "Photocraft")),      // another app
		mk(filepath.Join(h.base.config, "PdfCraftBackups")), // prefix lookalike
	}

	// Plain uninstall keeps settings; DataDirs still reports them.
	s, err := h.Uninstall("pdfcraft", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.DataDirs) != 3 {
		t.Fatalf("DataDirs = %v", s.DataDirs)
	}
	for _, p := range ours {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("plain uninstall removed %s", p)
		}
	}

	// Clean uninstall of a no-longer-installed app clears the leftovers.
	s, err = h.Uninstall("pdfcraft", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.DataDirs) != 0 {
		t.Fatalf("DataDirs after clean = %v", s.DataDirs)
	}
	for _, p := range ours {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s not removed", p)
		}
	}
	for _, p := range keep {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("clean uninstall removed unrelated %s", p)
		}
	}
}

// PhotoCraft's Windows zip ships portable.txt, so the app keeps all settings in
// <exe dir>/PhotoCraftData. Each release has a differently named top folder,
// so a naive folder swap would throw those settings away.
func TestPortableDataSurvivesUpdateAndUninstall(t *testing.T) {
	data := t.TempDir()
	h := newTestHub(t, data)
	ctx := context.Background()
	pkg := func(v string) []byte {
		top := "photocraft-" + v + "-windows-x64-portable/"
		return writeZip(t, "", map[string]string{top + "photocraft.exe": v, top + "portable.txt": ""})
	}
	settings := func() string {
		in := find(h.List(), "photocraft").Installed
		return filepath.Join(filepath.Dir(in.Exe), "PhotoCraftData", "preferences.json")
	}
	mustRead := func(p string) string {
		t.Helper()
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("settings lost: %v", err)
		}
		return string(b)
	}

	fakeRelease(t, "photocraft", "photocraft", "0.1.0", pkg("0.1.0"), "")
	if _, err := h.Install(ctx, "photocraft"); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(settings()), 0o755)
	os.WriteFile(settings(), []byte("mine"), 0o644)
	if d := find(h.List(), "photocraft").DataDirs; len(d) != 1 || filepath.Base(d[0]) != "PhotoCraftData" {
		t.Fatalf("DataDirs = %v", d)
	}

	// Update: settings move into the new versioned folder.
	fakeRelease(t, "photocraft", "photocraft", "0.2.0", pkg("0.2.0"), "")
	if _, err := h.Install(ctx, "photocraft"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(settings(), "0.2.0") || mustRead(settings()) != "mine" {
		t.Fatalf("after update: %s", settings())
	}

	// Failed update (bad checksum) leaves settings where the old exe expects them.
	fakeRelease(t, "photocraft", "photocraft", "0.3.0", pkg("0.3.0"), strings.Repeat("0", 64))
	if _, err := h.Install(ctx, "photocraft"); err == nil {
		t.Fatal("want checksum error")
	}
	if mustRead(settings()) != "mine" {
		t.Fatal("settings changed by failed update")
	}

	// Plain uninstall keeps settings; reinstall picks them up.
	dir := find(h.List(), "photocraft").Installed.Dir
	s, err := h.Uninstall("photocraft", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.DataDirs) != 1 || mustRead(filepath.Join(s.DataDirs[0], "preferences.json")) != "mine" {
		t.Fatalf("plain uninstall: %v", s.DataDirs)
	}
	if _, err := os.Stat(filepath.Join(dir, "photocraft-0.2.0-windows-x64-portable")); !os.IsNotExist(err) {
		t.Fatal("program files not removed")
	}
	fakeRelease(t, "photocraft", "photocraft", "0.2.0", pkg("0.2.0"), "")
	if _, err := h.Install(ctx, "photocraft"); err != nil {
		t.Fatal(err)
	}
	if mustRead(settings()) != "mine" {
		t.Fatal("reinstall lost kept settings")
	}

	// Clean uninstall removes everything, including the install folder itself.
	s, err = h.Uninstall("photocraft", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) || len(s.DataDirs) != 0 {
		t.Fatalf("clean uninstall left %s / %v", dir, s.DataDirs)
	}
}

// Release discovery uses only github.com web endpoints (redirect + checksum
// file), never the rate-limited REST API.
func TestRefreshFromWebEndpoints(t *testing.T) {
	h := newTestHub(t, t.TempDir())
	ctx := context.Background()
	zip := writeZip(t, "", map[string]string{"p/printcraft.exe": "v1"})
	f := fakeGitHub(t, "0.2.0", zip, "")

	st := find(h.Refresh(ctx), "pdfcraft")
	if st.LatestVersion != "0.2.0" || !st.Supported || st.DownloadSize != int64(len(zip)) {
		t.Fatalf("got %+v", st)
	}
	if !strings.HasSuffix(st.ReleaseURL, "/storytold/pdfcraft/releases/tag/v0.2.0") {
		t.Fatalf("release url %q", st.ReleaseURL)
	}
	// Apps the fake doesn't serve report an error instead of a version.
	if other := find(h.List(), "filmcraft"); other.Error == "" || other.LatestVersion != "" {
		t.Fatalf("filmcraft: %+v", other)
	}
	// Same release on the next check: size is reused, not fetched again.
	h.Refresh(ctx)
	if n := f.heads.Load(); n != 1 {
		t.Fatalf("HEAD requests = %d, want 1", n)
	}
}

func TestReleaseWithoutChecksumsIsRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.Redirect(w, r, "/storytold/pdfcraft/releases/tag/v1.0.0", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	old := ghBase
	ghBase = srv.URL
	defer func() { ghBase = old }()

	if _, err := latestRelease(context.Background(), Owner, "pdfcraft"); err == nil || !strings.Contains(err.Error(), sumsAsset) {
		t.Fatalf("want missing-checksums error, got %v", err)
	}
	h := newTestHub(t, t.TempDir())
	if _, err := h.Install(context.Background(), "pdfcraft"); err == nil {
		t.Fatal("installed without checksums")
	}
}

func TestCheckSelfUpdate(t *testing.T) {
	body := []byte("new hub")
	sig := bytes.Repeat([]byte{7}, 64) // Wails verifies it against the pinned key; here only transport is tested
	dl := "/" + selfOwner + "/" + selfRepo + "/releases/download/v0.3.0/crafting-apps-hub-v0.3.0-"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := "/" + selfOwner + "/" + selfRepo + "/releases"
		switch r.URL.Path {
		case base + "/latest":
			http.Redirect(w, r, base+"/tag/v0.3.0", http.StatusFound)
		case base + "/download/v0.3.0/" + sumsAsset:
			for _, s := range []string{"windows-x64.zip", "macos-universal.zip", "linux-x64.tar.gz"} {
				fmt.Fprintf(w, "%s  crafting-apps-hub-v0.3.0-%s\n", strings.Repeat("a", 64), s)
			}
		case dl + "windows-x64.zip":
			w.Write(body)
		case dl + "windows-x64.zip.sig", dl + "macos-universal.zip.sig":
			fmt.Fprintln(w, base64.StdEncoding.EncodeToString(sig))
		case dl + "linux-x64.tar.gz.sig":
			fmt.Fprintln(w, "not a signature")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := ghBase
	ghBase = srv.URL
	defer func() { ghBase = old }()
	ctx := context.Background()

	for _, c := range []struct{ goos, goarch, asset string }{
		{"windows", "amd64", "crafting-apps-hub-v0.3.0-windows-x64.zip"},
		{"windows", "arm64", "crafting-apps-hub-v0.3.0-windows-x64.zip"}, // x64 under emulation
		{"darwin", "arm64", "crafting-apps-hub-v0.3.0-macos-universal.zip"},
	} {
		u, err := CheckSelfUpdate(ctx, "0.2.1", c.goos, c.goarch)
		if err != nil || u == nil || u.Version != "0.3.0" || u.Asset.Name != c.asset || !bytes.Equal(u.Sig, sig) {
			t.Fatalf("%s/%s: got %+v, %v", c.goos, c.goarch, u, err)
		}
	}
	// A garbled .sig means no update, never an unsigned one.
	if u, err := CheckSelfUpdate(ctx, "0.2.1", "linux", "amd64"); err == nil || u != nil {
		t.Fatalf("linux with bad sig: got %+v, %v", u, err)
	}
	for _, cur := range []string{"0.3.0", "0.10.0"} {
		if u, err := CheckSelfUpdate(ctx, cur, "windows", "amd64"); err != nil || u != nil {
			t.Fatalf("hub %s: want no update, got %+v, %v", cur, u, err)
		}
	}

	u, _ := CheckSelfUpdate(ctx, "0.2.1", "windows", "amd64")
	var got bytes.Buffer
	var last int64
	if err := DownloadSelf(ctx, u, &got, func(n, _ int64) { last = n }); err != nil || got.String() != string(body) || last != int64(len(body)) {
		t.Fatalf("download: %q progress=%d err=%v", got.String(), last, err)
	}
}

func find(list []AppState, id string) AppState {
	for _, s := range list {
		if s.ID == id {
			return s
		}
	}
	return AppState{}
}
