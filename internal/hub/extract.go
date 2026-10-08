package hub

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// extract unpacks archive into dst (which must exist), dispatching on extension.
func extract(archive, dst string) error {
	switch {
	case strings.HasSuffix(archive, ".zip"):
		return extractZip(archive, dst)
	case strings.HasSuffix(archive, ".tar.gz"):
		return extractTarGz(archive, dst)
	case strings.HasSuffix(archive, ".dmg"):
		return extractDmg(archive, dst)
	}
	return fmt.Errorf("unsupported archive %s", filepath.Base(archive))
}

// safeJoin rejects entries that would escape dst (zip-slip).
func safeJoin(dst, name string) (string, error) {
	p := filepath.Join(dst, filepath.FromSlash(name))
	if p != dst && !strings.HasPrefix(p, dst+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes target: %q", name)
	}
	return p, nil
}

func extractZip(archive, dst string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		p, err := safeJoin(dst, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(p, 0o755); err != nil {
				return err
			}
			continue
		}
		if !f.Mode().IsRegular() {
			continue // ponytail: symlinks skipped; Craft zips contain none.
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = writeFile(p, rc, f.Mode().Perm()|0o600)
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(archive, dst string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		p, err := safeJoin(dst, h.Name)
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(p, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeFile(p, tr, fs.FileMode(h.Mode).Perm()|0o600); err != nil {
				return err
			}
		}
		// ponytail: symlinks/hardlinks skipped; Craft tarballs contain none.
	}
}

func writeFile(p string, r io.Reader, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// extractDmg mounts the image read-only and copies every .app bundle out.
func extractDmg(archive, dst string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("dmg images can only be installed on macOS")
	}
	mnt, err := os.MkdirTemp("", "crafthub-dmg-")
	if err != nil {
		return err
	}
	defer os.Remove(mnt)
	if out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen", "-mountpoint", mnt, archive).CombinedOutput(); err != nil {
		return fmt.Errorf("hdiutil attach: %v: %s", err, out)
	}
	defer exec.Command("hdiutil", "detach", "-force", mnt).Run()
	apps, _ := filepath.Glob(filepath.Join(mnt, "*.app"))
	if len(apps) == 0 {
		return errors.New("no .app bundle in disk image")
	}
	for _, a := range apps {
		// ditto preserves bundle symlinks, xattrs and code signatures.
		if out, err := exec.Command("ditto", a, filepath.Join(dst, filepath.Base(a))).CombinedOutput(); err != nil {
			return fmt.Errorf("ditto: %v: %s", err, out)
		}
	}
	return nil
}

// findExecutable locates the GUI binary for prefix inside an install dir.
func findExecutable(root, prefix, goos string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return err
		}
		name := strings.ToLower(d.Name())
		switch goos {
		case "darwin":
			if d.IsDir() && strings.HasSuffix(name, ".app") {
				found = p
				return fs.SkipAll
			}
		case "windows":
			if !d.IsDir() && name == prefix+".exe" {
				found = p
			}
		default:
			if !d.IsDir() && name == prefix && filepath.Base(filepath.Dir(p)) == "bin" {
				found = p
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("%s executable not found in package", prefix)
	}
	return found, nil
}
