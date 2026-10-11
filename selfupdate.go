package main

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"craftinghub/internal/hub"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// updaterPublicKey verifies hub updates. The private half lives only in the
// UPDATER_SIGNING_KEY GitHub Actions secret (and the maintainer's backup).
//
//go:embed build/updater.pub
var updaterPublicKey []byte

// selfSource feeds Wails' updater from the hub's own release discovery
// (github.com web endpoints, no REST API). Wails downloads, verifies the
// Ed25519 signature, unpacks the zip/tar.gz and swaps the exe (.app on macOS)
// through a helper process on Restart.
type selfSource struct{}

func (selfSource) Name() string { return "github" }

func (selfSource) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	u, err := hub.CheckSelfUpdate(ctx, req.CurrentVersion, req.Platform, req.Arch)
	if err != nil || u == nil {
		return nil, err
	}
	return &updater.Release{
		Version:  u.Version,
		Artifact: updater.Artifact{Filename: u.Asset.Name, Platform: req.Platform, Arch: req.Arch},
		// Wails hashes the download with SHA-512 and checks this against updaterPublicKey; fails closed.
		Verification: &updater.Verification{DigestAlgo: "sha512", SignatureAlgo: "ed25519ph", Signature: u.Sig},
		Metadata:     map[string]any{"page": u.Page, "url": u.Asset.URL},
	}, nil
}

func (selfSource) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	return hub.DownloadSelf(ctx, &hub.SelfUpdate{Asset: hub.Asset{Name: r.Artifact.Filename, URL: r.Metadata["url"].(string)}}, dst, onProgress)
}

// HubUpdate is a newer hub release.
type HubUpdate struct {
	Version string `json:"version"`
	URL     string `json:"url"`     // release page
	InPlace bool   `json:"inPlace"` // false: the exe folder is not writable, so the UI links to the release page instead
}

// CheckHubUpdate reports a newer hub release on GitHub, or nil when current.
func (s *HubService) CheckHubUpdate(ctx context.Context) (*HubUpdate, error) {
	rel, err := application.Get().Updater.Check(ctx)
	if err != nil || rel == nil {
		return nil, err
	}
	return &HubUpdate{Version: rel.Version, URL: rel.Metadata["page"].(string), InPlace: canSwapSelf()}, nil
}

// InstallHubUpdate downloads, verifies and stages the update found by
// CheckHubUpdate. Progress arrives as wails:updater:download-progress events.
func (s *HubService) InstallHubUpdate(ctx context.Context) error {
	return application.Get().Updater.DownloadAndInstall(ctx)
}

// RestartHub quits and lets the Wails helper swap in the staged update,
// then relaunch. Refused while an app install/uninstall is running.
func (s *HubService) RestartHub(ctx context.Context) error {
	if s.Busy() {
		return errors.New("wait for the running install to finish, then restart")
	}
	return application.Get().Updater.Restart(ctx)
}

// canSwapSelf reports whether the helper can replace the running build:
// it renames the exe (or the .app on macOS) inside its folder, so that folder
// must be writable. AppImage mounts are read-only; server builds have no exe
// to swap.
func canSwapSelf() bool {
	if serverMode || os.Getenv("APPIMAGE") != "" {
		return false
	}
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return false
	}
	target := exe
	if runtime.GOOS == "darwin" {
		for d := filepath.Dir(exe); d != filepath.Dir(d); d = filepath.Dir(d) {
			if filepath.Ext(d) == ".app" {
				target = d
				break
			}
		}
	}
	return writable(filepath.Dir(target))
}
