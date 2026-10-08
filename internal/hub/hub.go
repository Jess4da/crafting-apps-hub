package hub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Progress is emitted while an install/update runs.
type Progress struct {
	ID       string `json:"id"`
	Stage    string `json:"stage"` // "download" | "verify" | "extract" | "done"
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
}

// Installed is the persisted record of one installed app.
type Installed struct {
	Version     string    `json:"version"`
	Dir         string    `json:"dir"`
	Exe         string    `json:"exe"`
	InstalledAt time.Time `json:"installedAt"`
}

// AppState is what the UI renders for one catalog entry.
type AppState struct {
	App
	Installed       *Installed `json:"installed"`
	LatestVersion   string     `json:"latestVersion"`
	PublishedAt     time.Time  `json:"publishedAt"`
	ReleaseNotes    string     `json:"releaseNotes"`
	ReleaseURL      string     `json:"releaseUrl"`
	DownloadSize    int64      `json:"downloadSize"`
	Supported       bool       `json:"supported"` // a build exists for this OS/arch
	UpdateAvailable bool       `json:"updateAvailable"`
	Error           string     `json:"error"`
	// DataDirs are existing per-user settings/cache folders a clean uninstall removes.
	DataDirs []string `json:"dataDirs"`
}

type Settings struct {
	InstallRoot string `json:"installRoot"`
}

type state struct {
	Settings  Settings              `json:"settings"`
	Installed map[string]*Installed `json:"installed"`
}

// Hub manages download, install, update, uninstall and launch of Craft apps.
// All exported methods are bound to the frontend by Wails.
type Hub struct {
	emit      func(Progress)
	statePath string
	goos      string
	goarch    string
	base      baseDirs

	mu       sync.Mutex
	st       state
	releases map[string]Release
	relErr   map[string]string
	busy     map[string]bool
}

// New loads persisted state from dataDir. defaultRoot is used for installs
// until the user picks another folder. emit receives progress updates.
func New(dataDir, defaultRoot string, emit func(Progress)) (*Hub, error) {
	h := &Hub{
		emit:      emit,
		statePath: filepath.Join(dataDir, "state.json"),
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		base:      userBaseDirs(runtime.GOOS),
		st:        state{Installed: map[string]*Installed{}},
		releases:  map[string]Release{},
		relErr:    map[string]string{},
		busy:      map[string]bool{},
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(h.statePath)
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &h.st); err != nil {
			return nil, fmt.Errorf("corrupt %s: %w", h.statePath, err)
		}
		if h.st.Installed == nil {
			h.st.Installed = map[string]*Installed{}
		}
	case !errors.Is(err, os.ErrNotExist):
		return nil, err
	}
	if h.st.Settings.InstallRoot == "" {
		h.st.Settings.InstallRoot = defaultRoot
	}
	// Drop records whose files were removed behind our back.
	for id, in := range h.st.Installed {
		if _, err := os.Stat(in.Exe); err != nil {
			delete(h.st.Installed, id)
		}
	}
	return h, nil
}

// saveLocked writes state atomically. Caller holds h.mu.
func (h *Hub) saveLocked() error {
	b, err := json.MarshalIndent(h.st, "", "  ")
	if err != nil {
		return err
	}
	tmp := h.statePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, h.statePath)
}

func (h *Hub) GetSettings() Settings {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.st.Settings
}

// SetInstallRoot changes where new installs go. Existing installs stay put.
func (h *Hub) SetInstallRoot(dir string) (Settings, error) {
	if !filepath.IsAbs(dir) {
		return Settings{}, errors.New("install folder must be an absolute path")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Settings{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.st.Settings.InstallRoot = dir
	return h.st.Settings, h.saveLocked()
}

// Platform reports the OS/arch the hub picks builds for, e.g. "windows/amd64".
func (h *Hub) Platform() string { return h.goos + "/" + h.goarch }

// List returns current state without touching the network.
func (h *Hub) List() []AppState {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]AppState, 0, len(Catalog))
	for _, a := range Catalog {
		out = append(out, h.stateLocked(a))
	}
	return out
}

func (h *Hub) stateLocked(a App) AppState {
	s := AppState{App: a, Error: h.relErr[a.ID], DataDirs: h.dataDirs(a)}
	if in, ok := h.st.Installed[a.ID]; ok {
		c := *in
		s.Installed = &c
	}
	rel, ok := h.releases[a.ID]
	if !ok {
		return s
	}
	s.LatestVersion = rel.Version()
	s.PublishedAt = rel.PublishedAt
	s.ReleaseNotes = rel.Body
	s.ReleaseURL = rel.HTMLURL
	if asset, ok := pickAsset(rel, a.Asset, h.goos, h.goarch); ok {
		s.Supported = true
		s.DownloadSize = asset.Size
	}
	s.UpdateAvailable = s.Installed != nil && s.Supported && newerVersion(s.LatestVersion, s.Installed.Version)
	return s
}

// Refresh fetches the latest release of every app in parallel.
func (h *Hub) Refresh(ctx context.Context) []AppState {
	var wg sync.WaitGroup
	for _, a := range Catalog {
		wg.Go(func() {
			rel, err := latestRelease(ctx, a.Repo)
			h.mu.Lock()
			defer h.mu.Unlock()
			if err != nil {
				h.relErr[a.ID] = err.Error()
				return
			}
			delete(h.relErr, a.ID)
			h.releases[a.ID] = rel
		})
	}
	wg.Wait()
	return h.List()
}

// Install downloads, verifies and installs the latest release of id.
// Calling it on an installed app updates it in place.
func (h *Hub) Install(ctx context.Context, id string) (AppState, error) {
	app, ok := findApp(id)
	if !ok {
		return AppState{}, fmt.Errorf("unknown app %q", id)
	}
	h.mu.Lock()
	if h.busy[id] {
		h.mu.Unlock()
		return AppState{}, fmt.Errorf("%s is already being installed", app.Name)
	}
	h.busy[id] = true
	root := h.st.Settings.InstallRoot
	prev := h.st.Installed[id]
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.busy, id)
		h.mu.Unlock()
	}()

	rel, err := latestRelease(ctx, app.Repo)
	if err != nil {
		return AppState{}, err
	}
	asset, ok := pickAsset(rel, app.Asset, h.goos, h.goarch)
	if !ok {
		return AppState{}, fmt.Errorf("%s %s has no build for %s", app.Name, rel.Version(), h.Platform())
	}
	sum, err := h.expectedSum(ctx, rel, asset.Name)
	if err != nil {
		return AppState{}, err
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return AppState{}, err
	}
	work, err := os.MkdirTemp(root, ".tmp-"+id+"-")
	if err != nil {
		return AppState{}, err
	}
	defer os.RemoveAll(work)

	archive := filepath.Join(work, asset.Name)
	if err := h.download(ctx, id, asset, archive, sum); err != nil {
		return AppState{}, err
	}
	h.emit(Progress{ID: id, Stage: "extract"})
	staged := filepath.Join(work, "pkg")
	if err := os.Mkdir(staged, 0o755); err != nil {
		return AppState{}, err
	}
	if err := extract(archive, staged); err != nil {
		return AppState{}, fmt.Errorf("extract: %w", err)
	}

	// Swap the new tree into place; keep the old one until the swap succeeds.
	dir := filepath.Join(root, id)
	if prev != nil {
		dir = prev.Dir
	}
	backup := filepath.Join(work, "old")
	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, backup); err != nil {
			return AppState{}, fmt.Errorf("cannot replace %s (close it if it is running): %w", app.Name, err)
		}
	}
	if err := os.Rename(staged, dir); err != nil {
		os.Rename(backup, dir)
		return AppState{}, err
	}
	exe, err := findExecutable(dir, app.Asset, h.goos)
	if err == nil {
		err = keepPortable(app, backup, filepath.Dir(exe))
	}
	if err != nil {
		os.RemoveAll(dir)
		os.Rename(backup, dir)
		return AppState{}, err
	}
	// Past this point nothing can fail and roll back, so moved settings are safe.

	h.mu.Lock()
	defer h.mu.Unlock()
	h.releases[id] = rel
	h.st.Installed[id] = &Installed{Version: rel.Version(), Dir: dir, Exe: exe, InstalledAt: time.Now()}
	if err := h.saveLocked(); err != nil {
		return AppState{}, err
	}
	h.emit(Progress{ID: id, Stage: "done"})
	return h.stateLocked(app), nil
}

func (h *Hub) expectedSum(ctx context.Context, rel Release, name string) (string, error) {
	for _, a := range rel.Assets {
		if a.Name != sumsAsset {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
		if err != nil {
			return "", err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("checksums: %s", res.Status)
		}
		if sum := parseSums(res.Body)[name]; sum != "" {
			return sum, nil
		}
		return "", fmt.Errorf("%s is not listed in %s", name, sumsAsset)
	}
	// Refuse unverifiable downloads rather than install something unchecked.
	return "", fmt.Errorf("release %s has no %s", rel.Tag, sumsAsset)
}

// download streams asset to dst, hashing as it goes, and fails on mismatch.
func (h *Hub) download(ctx context.Context, id string, asset Asset, dst, wantSum string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", asset.Name, res.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	total := res.ContentLength
	if total <= 0 {
		total = asset.Size
	}
	hash := sha256.New()
	p := &progressWriter{h: h, id: id, total: total}
	if _, err := io.Copy(io.MultiWriter(f, hash, p), res.Body); err != nil {
		return fmt.Errorf("download %s: %w", asset.Name, err)
	}
	h.emit(Progress{ID: id, Stage: "verify", Received: p.n, Total: total})
	if got := hex.EncodeToString(hash.Sum(nil)); got != wantSum {
		return fmt.Errorf("checksum mismatch for %s", asset.Name)
	}
	return f.Close()
}

type progressWriter struct {
	h     *Hub
	id    string
	n     int64
	total int64
	last  time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.n += int64(len(b))
	if now := time.Now(); now.Sub(p.last) > 100*time.Millisecond {
		p.last = now
		p.h.emit(Progress{ID: p.id, Stage: "download", Received: p.n, Total: p.total})
	}
	return len(b), nil
}

// Uninstall removes the app's install folder. With clean, it also deletes the
// app's own settings, recovery and cache folders (see App.Data); this also
// works for apps that are no longer installed but left data behind.
func (h *Hub) Uninstall(id string, clean bool) (AppState, error) {
	app, ok := findApp(id)
	if !ok {
		return AppState{}, fmt.Errorf("unknown app %q", id)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.busy[id] {
		return AppState{}, fmt.Errorf("%s is being installed", app.Name)
	}
	if in, ok := h.st.Installed[id]; ok {
		if err := h.removeInstall(app, in.Dir, clean); err != nil {
			return AppState{}, fmt.Errorf("cannot remove %s (close it if it is running): %w", app.Name, err)
		}
		delete(h.st.Installed, id)
		if err := h.saveLocked(); err != nil {
			return AppState{}, err
		}
	}
	if clean {
		for _, d := range h.dataDirs(app) {
			if err := os.RemoveAll(d); err != nil {
				return h.stateLocked(app), fmt.Errorf("app removed, but %s could not be deleted: %w", d, err)
			}
		}
		os.Remove(filepath.Join(h.st.Settings.InstallRoot, id)) // now-empty leftover holder; fails harmlessly if not empty
	}
	return h.stateLocked(app), nil
}

// removeInstall deletes an install folder. Unless clean, the app's portable
// data folder is kept at <dir>/<Portable> so a reinstall picks it up again.
func (h *Hub) removeInstall(app App, dir string, clean bool) error {
	keep := ""
	if !clean {
		keep = findPortable(dir, app.Portable)
	}
	if keep == "" {
		return os.RemoveAll(dir)
	}
	tmp := dir + ".keep"
	os.RemoveAll(tmp)
	if err := os.Rename(keep, tmp); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		os.MkdirAll(filepath.Dir(keep), 0o755)
		os.Rename(tmp, keep) // put settings back where the (still installed) app expects them
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, app.Portable))
}

// keepPortable moves the app's portable data folder from the previous install
// tree (from) next to the new executable (exeDir). The versioned folder name in
// each archive means the exe directory changes on every update.
func keepPortable(app App, from, exeDir string) error {
	src := findPortable(from, app.Portable)
	if src == "" {
		return nil
	}
	dst := filepath.Join(exeDir, app.Portable)
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("keep %s settings: %w", app.Name, err)
	}
	return nil
}

// findPortable finds dir/<name> or dir/*/<name> ("" when absent or name is "").
func findPortable(dir, name string) string {
	if name == "" {
		return ""
	}
	for _, pat := range []string{filepath.Join(dir, name), filepath.Join(dir, "*", name)} {
		if m, _ := filepath.Glob(pat); len(m) > 0 {
			if fi, err := os.Stat(m[0]); err == nil && fi.IsDir() {
				return m[0]
			}
		}
	}
	return ""
}

// Launch starts the installed app detached from the hub.
func (h *Hub) Launch(id string) error {
	h.mu.Lock()
	in, ok := h.st.Installed[id]
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("%s is not installed", id)
	}
	cmd := exec.Command(in.Exe)
	if h.goos == "darwin" {
		cmd = exec.Command("open", in.Exe)
	}
	cmd.Dir = filepath.Dir(in.Exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// InstallDir returns the folder of an installed app ("" when not installed).
func (h *Hub) InstallDir(id string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if in, ok := h.st.Installed[id]; ok {
		return in.Dir
	}
	return ""
}
