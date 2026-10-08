package hub

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		"printcraft-0.3.0-windows-x64-portable.zip", "printcraft-0.3.0-windows-x64.msi",
		"printcraft-0.3.0-windows-arm64-portable.zip", "printcraft-0.3.0-windows-x86-portable.zip",
		"printcraft-0.3.0-linux-x86_64.tar.gz", "printcraft-0.3.0-linux-aarch64.tar.gz",
		"printcraft-0.3.0-linux-x86_64.deb", "printcraft-0.3.0-macos-universal.dmg",
		"printcraft-cli-0.3.0-macos-universal.zip",
	)
	noArm := rel("v0.2.1", "x-0.2.1-windows-x64-portable.zip")
	cases := []struct {
		r          Release
		prefix, os string
		arch, want string
	}{
		{full, "printcraft", "windows", "amd64", "printcraft-0.3.0-windows-x64-portable.zip"},
		{full, "printcraft", "windows", "arm64", "printcraft-0.3.0-windows-arm64-portable.zip"},
		{full, "printcraft", "windows", "386", "printcraft-0.3.0-windows-x86-portable.zip"},
		{full, "printcraft", "linux", "amd64", "printcraft-0.3.0-linux-x86_64.tar.gz"},
		{full, "printcraft", "linux", "arm64", "printcraft-0.3.0-linux-aarch64.tar.gz"},
		{full, "printcraft", "darwin", "arm64", "printcraft-0.3.0-macos-universal.dmg"},
		{noArm, "x", "windows", "arm64", "x-0.2.1-windows-x64-portable.zip"}, // x64 emulation fallback
		{full, "printcraft", "freebsd", "amd64", ""},                         // not in this release
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

// fakeGitHub serves a PdfCraft release API + assets. sum overrides the
// checksum file contents ("" = correct sum).
func fakeGitHub(t *testing.T, version string, zipBytes []byte, sum string) {
	fakeRelease(t, "pdfcraft", "printcraft", version, zipBytes, sum)
}

func fakeRelease(t *testing.T, repo, prefix, version string, zipBytes []byte, sum string) {
	t.Helper()
	asset := fmt.Sprintf("%s-%s-windows-x64-portable.zip", prefix, version)
	if sum == "" {
		h := sha256.Sum256(zipBytes)
		sum = hex.EncodeToString(h[:])
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/storytold/" + repo + "/releases/latest":
			json.NewEncoder(w).Encode(Release{Tag: "v" + version, Assets: []Asset{
				{Name: asset, Size: int64(len(zipBytes)), URL: srv.URL + "/dl/" + asset},
				{Name: sumsAsset, URL: srv.URL + "/dl/sums"},
			}})
		case "/dl/" + asset:
			w.Write(zipBytes)
		case "/dl/sums":
			fmt.Fprintf(w, "%s  %s\n", sum, asset)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	old := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = old })
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

	fakeGitHub(t, "0.1.0", writeZip(t, "", map[string]string{"printcraft-0.1.0-windows-x64-portable/printcraft.exe": "v1"}), "")
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
	fakeGitHub(t, "0.2.0", writeZip(t, "", map[string]string{"printcraft-0.2.0-windows-x64-portable/printcraft.exe": "v2"}), "")
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
	fakeGitHub(t, "0.1.0", writeZip(t, "", map[string]string{"p/printcraft.exe": "v1"}), "")
	if _, err := h.Install(ctx, "pdfcraft"); err != nil {
		t.Fatal(err)
	}
	fakeGitHub(t, "0.2.0", writeZip(t, "", map[string]string{"p/printcraft.exe": "evil"}), strings.Repeat("0", 64))
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

func TestCleanUninstallRemovesOnlyAppData(t *testing.T) {
	data := t.TempDir()
	h := newTestHub(t, data)
	ctx := context.Background()
	fakeGitHub(t, "0.1.0", writeZip(t, "", map[string]string{"p/printcraft.exe": "v1"}), "")
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

func find(list []AppState, id string) AppState {
	for _, s := range list {
		if s.ID == id {
			return s
		}
	}
	return AppState{}
}
