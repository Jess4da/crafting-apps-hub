//go:build linux

package hub

import (
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// extMime returns the MIME type the system knows for ext, or "".
func extMime(ext string) string {
	mt := mime.TypeByExtension("." + ext)
	if i := strings.IndexByte(mt, ';'); i >= 0 { // strip "; charset=..."
		mt = strings.TrimSpace(mt[:i])
	}
	return mt
}

// extMimes maps exts to known MIME types, deduplicated.
func extMimes(exts []string) (mimes, unknown []string) {
	for _, ext := range exts {
		mt := extMime(ext)
		if mt == "" {
			unknown = append(unknown, ext)
		} else if !slices.Contains(mimes, mt) {
			mimes = append(mimes, mt)
		}
	}
	return mimes, unknown
}

func setDefaultApp(app App, exts []string, in *Installed) error {
	dir, err := desktopDir()
	if err != nil {
		return err
	}
	mimes, unknown := extMimes(exts)
	if len(mimes) == 0 {
		return fmt.Errorf("no MIME type is known for %s", strings.Join(unknown, ", "))
	}
	file := desktopFile(app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(desktopEntry(app, in.Exe, mimes)), 0o644); err != nil {
		return err
	}
	args := append([]string{"default", file}, mimes...)
	if out, err := exec.Command("xdg-mime", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("xdg-mime: %v: %s", err, strings.TrimSpace(string(out)))
	}
	updateDesktopDatabase(dir)
	// ponytail: extensions with no installed MIME type are skipped (Craft's
	// portable tarballs don't ship their shared-mime-info XML); add a generated
	// mime package when double-clicking a native format on Linux matters.
	return nil
}

func desktopFile(app App) string { return "craftingapps-" + app.ID + ".desktop" }

// currentDefaults returns the desktop entry handling each ext's MIME type.
func currentDefaults(app App, exts []string) map[string]string {
	prev := map[string]string{}
	for _, ext := range exts {
		mt := extMime(ext)
		if mt == "" {
			continue
		}
		out, err := exec.Command("xdg-mime", "query", "default", mt).Output()
		if d := strings.TrimSpace(string(out)); err == nil && d != "" && d != desktopFile(app) {
			prev[ext] = d
		}
	}
	return prev
}

// restoreDefaults points each ext's MIME type back at its recorded entry, then
// removes ours (SetDefaultApp rewrites it for any types kept); types with no
// recorded entry fall back to the system default.
func restoreDefaults(app App, exts []string, prev map[string]string) error {
	var bad []string
	for _, ext := range exts {
		mt, d := extMime(ext), prev[ext]
		if mt == "" || d == "" {
			continue
		}
		if err := exec.Command("xdg-mime", "default", d, mt).Run(); err != nil {
			bad = append(bad, ext)
		}
	}
	if dir, err := desktopDir(); err == nil {
		os.Remove(filepath.Join(dir, desktopFile(app)))
		updateDesktopDatabase(dir)
	}
	if len(bad) > 0 {
		return fmt.Errorf("xdg-mime could not restore %s", strings.Join(bad, ", "))
	}
	return nil
}

func desktopEntry(app App, exe string, mimes []string) string {
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=" + app.Name + "\n" +
		"Exec=" + desktopQuote(exe) + " %f\n" +
		"MimeType=" + strings.Join(mimes, ";") + ";\n" +
		"NoDisplay=true\n" +
		"Terminal=false\n"
}

// desktopQuote quotes an Exec argument per the Desktop Entry spec: inside
// double quotes, backslash, backtick, dollar and quote are escaped.
func desktopQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`)
	return `"` + r.Replace(s) + `"`
}

func desktopDir() (string, error) {
	if x := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "applications"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "applications"), nil
}

// updateDesktopDatabase refreshes the desktop entry cache; it is advisory.
func updateDesktopDatabase(dir string) {
	exec.Command("update-desktop-database", dir).Run()
}
