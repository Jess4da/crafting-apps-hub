package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"craftinghub/internal/hub"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Typed event so the binding generator emits a TS type for the payload.
	application.RegisterEvent[hub.Progress]("hub:progress")
}

func main() {
	dataDir, defaultRoot, err := dirs()
	if err != nil {
		log.Fatal(err)
	}

	var app *application.App
	h, err := hub.New(dataDir, defaultRoot, func(p hub.Progress) { app.Event.Emit("hub:progress", p) })
	if err != nil {
		log.Fatal(err)
	}

	app = application.New(application.Options{
		Name:        "Crafting Apps Hub",
		Description: "Install, update and launch the Craft apps",
		Services: []application.Service{
			application.NewService(&HubService{Hub: h}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: version,
		Providers:      []updater.Provider{selfSource{}},
		PublicKey:      updaterPublicKey,   // releases MUST carry <asset>.sig; unsigned or wrongly signed updates fail
		Window:         updater.WindowNone, // footer UI drives it
	}); err != nil {
		log.Fatal(err)
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Crafting Apps Hub",
		Width:     1180,
		Height:    760,
		MinWidth:  720,
		MinHeight: 520,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(18, 18, 22),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// dirs returns the hub's state dir and default install root. On Windows and
// Linux it is portable by default: both live next to the hub executable (found
// via os.Executable, so a shortcut with another working directory still lands
// here), falling back to per-user folders when that folder is not writable
// (e.g. under Program Files). macOS always uses per-user folders: the .app
// bundle must stay unmodified (code seal, self-update replaces it whole).
func dirs() (dataDir, installRoot string, err error) {
	exe, err := os.Executable()
	if p := os.Getenv("APPIMAGE"); p != "" { // AppImage runs from a read-only mount; use the .AppImage file
		exe = p
	}
	if err == nil && runtime.GOOS != "darwin" {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			if base := filepath.Dir(exe); writable(base) {
				return filepath.Join(base, "hub-data"), filepath.Join(base, "apps"), nil
			}
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dataDir = filepath.Join(cfg, "CraftingAppsHub")
	switch runtime.GOOS {
	case "windows":
		local, err := os.UserCacheDir() // %LOCALAPPDATA%
		if err != nil {
			return "", "", err
		}
		installRoot = filepath.Join(local, "Programs", "CraftingApps")
	case "darwin":
		installRoot = filepath.Join(home, "Applications", "Crafting Apps")
	default:
		installRoot = filepath.Join(home, ".local", "share", "crafting-apps")
	}
	return dataDir, installRoot, nil
}

// writable reports whether files can be created in dir.
func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".write-test-")
	if err != nil {
		return false
	}
	f.Close()
	return os.Remove(f.Name()) == nil
}
