package hub

import (
	"os"
	"path/filepath"
	"slices"
)

// baseDirs maps dataLoc bases to absolute folders for the current user.
type baseDirs struct{ config, data, cache, temp string }

func userBaseDirs(goos string) baseDirs {
	cfg, _ := os.UserConfigDir()
	cache, _ := os.UserCacheDir()
	b := baseDirs{config: cfg, data: cfg, cache: cache, temp: os.TempDir()}
	if goos != "windows" && goos != "darwin" {
		if x := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(x) {
			b.data = x
		} else if home, err := os.UserHomeDir(); err == nil {
			b.data = filepath.Join(home, ".local", "share")
		}
	}
	return b
}

// dataDirs returns the existing per-user folders app wrote on this machine,
// including its portable data folder inside the install (or kept) folder.
// Caller holds h.mu.
func (h *Hub) dataDirs(app App) []string {
	var out []string
	dir := filepath.Join(h.st.Settings.InstallRoot, app.ID)
	if in, ok := h.st.Installed[app.ID]; ok {
		dir = in.Dir
	}
	if p := findPortable(dir, app.Portable); p != "" {
		out = append(out, p)
	}
	for _, l := range app.Data {
		name := map[string]string{"windows": l.win, "darwin": l.mac}[h.goos]
		if h.goos != "windows" && h.goos != "darwin" {
			name = l.nix
		}
		root := map[string]string{"config": h.base.config, "data": h.base.data, "cache": h.base.cache, "temp": h.base.temp}[l.base]
		if name == "" || root == "" {
			continue
		}
		p := filepath.Join(root, name)
		if fi, err := os.Stat(p); err == nil && fi.IsDir() && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}
