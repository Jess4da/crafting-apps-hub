package hub

// Owner is the GitHub account that publishes every Craft app.
const Owner = "storytold"

// App is a static catalog entry. Asset is the file prefix used in release
// assets and the executable name inside the archive (pdfcraft ships as
// "printcraft"). Data lists the per-user folders the app itself writes,
// removed by a clean uninstall. Portable names the folder the app keeps next
// to its executable in portable mode; it survives updates and plain uninstalls.
type App struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Repo        string    `json:"repo"`
	Asset       string    `json:"asset"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Data        []dataLoc `json:"-"`
	Portable    string    `json:"-"`
}

// dataLoc is one folder under a base dir, named per OS ("" = unused there).
// Bases: "config" (os.UserConfigDir), "data" (macOS/Windows: same as config;
// Linux: $XDG_DATA_HOME or ~/.local/share), "cache" (os.UserCacheDir, i.e.
// %LOCALAPPDATA% on Windows), "temp" (os.TempDir).
type dataLoc struct{ base, win, mac, nix string }

// Data folders are taken from each app's source (storytold/<repo>, Oct 2026).
// Deliberately NOT listed, because they hold user work: the LightCraft library
// (~/Pictures/LightCraft Library), EffectCraft presets (~/Documents/EffectCraft),
// and FilmCraft previews saved next to projects.
// Only PhotoCraft has a portable mode: its Windows zip ships portable.txt, which
// makes it store everything in <exe dir>/PhotoCraftData (app_dirs.rs). The other
// apps never write beside their executable.
var Catalog = []App{
	{ID: "photocraft", Name: "PhotoCraft", Repo: "photocraft", Asset: "photocraft", Category: "Image editing",
		Description: "Layers, masks, adjustments, brushes and real PSD files. A clean-room take on Photoshop.",
		Data:        []dataLoc{{"config", "Photocraft", "Photocraft", "photocraft"}},
		Portable:    "PhotoCraftData"},
	{ID: "vectorcraft", Name: "VectorCraft", Repo: "vectorcraft", Asset: "vectorcraft", Category: "Vector illustration",
		Description: "Paths, shapes, type and SVG editing. A clean-room take on Illustrator.",
		Data:        []dataLoc{{"config", "VectorCraft", "VectorCraft", "vectorcraft"}}},
	{ID: "filmcraft", Name: "FilmCraft", Repo: "filmcraft", Asset: "filmcraft", Category: "Video editing",
		Description: "Timeline editing, color and sound. A clean-room take on Premiere Pro.",
		Data: []dataLoc{
			{"data", "FilmCraft", "FilmCraft", "filmcraft"}, // autosave, speech models
			{"temp", "FilmCraft Previews", "FilmCraft Previews", "FilmCraft Previews"},
		}},
	{ID: "lightcraft", Name: "LightCraft", Repo: "lightcraft", Asset: "lightcraft", Category: "Photo library & raw",
		Description: "Photo library and raw development. A clean-room take on Lightroom.",
		Data:        []dataLoc{{"config", "LightCraft", "LightCraft", "lightcraft"}}},
	{ID: "pdfcraft", Name: "PdfCraft", Repo: "pdfcraft", Asset: "printcraft", Category: "PDF workbench",
		Description: "Read, organize, combine, split and secure PDFs. A clean-room take on Acrobat.",
		Data: []dataLoc{
			{"data", "PdfCraft", "PdfCraft", "pdfcraft"},       // eframe storage_dir + recovery
			{"cache", "PdfCraft", "", ""},                      // %LOCALAPPDATA%\PdfCraft\Recovery
			{"data", "PrintCraft", "PrintCraft", "printcraft"}, // pre-rename folders
			{"cache", "PrintCraft", "", ""},
		}},
	{ID: "effectcraft", Name: "EffectCraft", Repo: "effectcraft", Asset: "effectcraft", Category: "Motion & VFX",
		Description: "Compositions, layers, keyframes and effects. A clean-room take on After Effects.",
		Data: []dataLoc{
			{"config", "EffectCraft", "EffectCraft", "effectcraft"},
			{"cache", "EffectCraft", "EffectCraft", "effectcraft"}, // disk cache
		}},
	{ID: "designcraft", Name: "DesignCraft", Repo: "designcraft", Asset: "designcraft", Category: "Page layout",
		Description: "Page layout and publishing. A clean-room take on InDesign.",
		Data: []dataLoc{
			{"config", "DesignCraft", "DesignCraft", "designcraft"}, // prefs (+ recovery on macOS/Windows)
			{"data", "", "", "designcraft"},                         // Linux recovery
		}},
}

func findApp(id string) (App, bool) {
	for _, a := range Catalog {
		if a.ID == id {
			return a, true
		}
	}
	return App{}, false
}
