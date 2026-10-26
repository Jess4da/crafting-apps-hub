package hub

// Owner is the GitHub account that publishes every Craft app.
const Owner = "storytold"

// App is a static catalog entry. Asset is the file prefix used in release
// assets and the executable name inside the archive (pdfcraft shipped as
// "printcraft" before v0.4.0). Group is the UI section; Catalog keeps each
// group contiguous.
// Data lists the per-user folders the app itself writes,
// removed by a clean uninstall. Portable names the folder the app keeps next
// to its executable in portable mode; it survives updates and plain uninstalls.
// Extensions are the document types the app opens, lowercase and without the
// leading dot; SetDefaultApp registers the app as their OS handler.
type App struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Repo        string    `json:"repo"`
	Asset       string    `json:"asset"`
	Group       string    `json:"group"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Extensions  []string  `json:"extensions"`
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
// GridCraft, WordCraft, DeckCraft and SoundCraft keep ui.json in a config
// folder; DeckCraft adds Recovery (inside it on macOS/Windows, under the data
// dir on Linux). CADCraft persists nothing. FigCraft keeps prefs, Recovery and
// Plugins in a "FigCraft" config folder on every OS (services.rs).
// Only PhotoCraft has a portable mode: its Windows zip ships portable.txt, which
// makes it store everything in <exe dir>/PhotoCraftData (app_dirs.rs). The other
// apps never write beside their executable.
//
// Extensions mirror what each app declares for OS association
// (packaging/macos/Info.plist.in, packaging/linux/*.desktop MimeType) plus the
// formats its own open dialog/argv path reads (storytold/<repo>, Oct 2026).
// Deliberately left out: system formats every app parses (json, xml, gz) and
// preset libraries imported into a document (PhotoCraft .abr/.grd/.aco/.ase).
var Catalog = []App{
	{ID: "photocraft", Name: "PhotoCraft", Repo: "photocraft", Asset: "photocraft", Group: "Creative", Category: "Image editing",
		Description: "Layers, masks, adjustments, brushes and real PSD files. A clean-room take on Photoshop.",
		Extensions:  []string{"pcraft", "psd", "psb", "psdt", "ora", "kra", "krz", "pdn", "qoi", "png", "jpg", "jpeg", "tif", "tiff", "webp", "gif", "bmp", "tga", "ico", "pbm", "pgm", "ppm", "pnm", "pam", "exr", "hdr", "avif"},
		Data:        []dataLoc{{"config", "Photocraft", "Photocraft", "photocraft"}},
		Portable:    "PhotoCraftData"},
	{ID: "lightcraft", Name: "LightCraft", Repo: "lightcraft", Asset: "lightcraft", Group: "Creative", Category: "Photo library & raw",
		Description: "Photo library and raw development. A clean-room take on Lightroom.",
		Extensions:  []string{"jpg", "jpeg", "png", "tif", "tiff", "webp", "dng", "cr2", "cr3", "nef", "nrw", "arw", "sr2", "srf", "raf", "orf", "rw2", "rwl", "raw", "pef", "srw", "psd", "psb", "jxl", "gif", "bmp", "heic", "heif", "avif", "iiq", "crw", "mrw", "x3f", "kdc", "mos", "erf", "3fr", "fff", "mef"},
		Data:        []dataLoc{{"config", "LightCraft", "LightCraft", "lightcraft"}}},
	{ID: "vectorcraft", Name: "VectorCraft", Repo: "vectorcraft", Asset: "vectorcraft", Group: "Creative", Category: "Vector illustration",
		Description: "Paths, shapes, type and SVG editing. A clean-room take on Illustrator.",
		Extensions:  []string{"vectorcraft", "drawcraft", "svg", "svgz", "pdf", "ai", "ait", "png", "jpg", "jpeg", "gif", "webp", "tif", "tiff", "bmp", "vctemplate", "dxf", "emf", "wmf", "eps", "af", "afdesign", "afpub"},
		Data:        []dataLoc{{"config", "VectorCraft", "VectorCraft", "vectorcraft"}}},
	{ID: "designcraft", Name: "DesignCraft", Repo: "designcraft", Asset: "designcraft", Group: "Creative", Category: "Page layout",
		Description: "Page layout and publishing. A clean-room take on InDesign.",
		Extensions:  []string{"designcraft", "idml", "indd", "indt"},
		Data: []dataLoc{
			{"config", "DesignCraft", "DesignCraft", "designcraft"}, // prefs (+ recovery on macOS/Windows)
			{"data", "", "", "designcraft"},                         // Linux recovery
		}},
	{ID: "figcraft", Name: "FigCraft", Repo: "figcraft", Asset: "figcraft", Group: "Creative", Category: "Interface design",
		Description: "Frames, auto layout, components, variables and prototypes. A clean-room take on Figma.",
		Extensions:  []string{"figcraft", "svg", "png", "jpg", "jpeg"},
		Data:        []dataLoc{{"config", "FigCraft", "FigCraft", "FigCraft"}}},
	{ID: "filmcraft", Name: "FilmCraft", Repo: "filmcraft", Asset: "filmcraft", Group: "Creative", Category: "Video editing",
		Description: "Timeline editing, color and sound. A clean-room take on Premiere Pro.",
		Extensions:  []string{"fcproj", "mp4", "m4v", "mov", "mkv", "webm", "avi", "mxf", "ts", "m2ts", "m2t", "mpg", "mpeg", "3gp"},
		Data: []dataLoc{
			{"data", "FilmCraft", "FilmCraft", "filmcraft"}, // autosave, speech models
			{"temp", "FilmCraft Previews", "FilmCraft Previews", "FilmCraft Previews"},
		}},
	{ID: "effectcraft", Name: "EffectCraft", Repo: "effectcraft", Asset: "effectcraft", Group: "Creative", Category: "Motion & VFX",
		Description: "Compositions, layers, keyframes and effects. A clean-room take on After Effects.",
		Extensions:  []string{"ecproj", "ecprojx"},
		Data: []dataLoc{
			{"config", "EffectCraft", "EffectCraft", "effectcraft"},
			{"cache", "EffectCraft", "EffectCraft", "effectcraft"}, // disk cache
		}},
	{ID: "pdfcraft", Name: "PdfCraft", Repo: "pdfcraft", Asset: "pdfcraft", Group: "Creative", Category: "PDF workbench",
		Description: "Read, organize, combine, split and secure PDFs. A clean-room take on Acrobat.",
		Extensions:  []string{"pdf"},
		Data: []dataLoc{
			{"data", "PdfCraft", "PdfCraft", "pdfcraft"},       // eframe storage_dir + recovery
			{"cache", "PdfCraft", "", ""},                      // %LOCALAPPDATA%\PdfCraft\Recovery
			{"data", "PrintCraft", "PrintCraft", "printcraft"}, // pre-rename folders
			{"cache", "PrintCraft", "", ""},
		}},
	{ID: "wordcraft", Name: "WordCraft", Repo: "wordcraft", Asset: "wordcraft", Group: "Office", Category: "Word processing",
		Description: "Documents, styles and DOCX files. A clean-room take on Word.",
		Extensions:  []string{"docx", "docm", "dotx", "dotm", "doc", "dot", "odt", "rtf", "md", "markdown", "html", "htm", "txt", "tex", "latex", "ltx"},
		Data:        []dataLoc{{"config", "WordCraft", "WordCraft", "wordcraft"}}},
	{ID: "gridcraft", Name: "GridCraft", Repo: "gridcraft", Asset: "gridcraft", Group: "Office", Category: "Spreadsheets",
		Description: "Workbooks, formulas and XLSX files. A clean-room take on Excel.",
		Extensions:  []string{"xlsx", "xlsm", "xlsb", "ods", "csv", "tsv"},
		Data:        []dataLoc{{"config", "GridCraft", "GridCraft", "gridcraft"}}},
	{ID: "deckcraft", Name: "DeckCraft", Repo: "deckcraft", Asset: "deckcraft", Group: "Office", Category: "Presentations",
		Description: "Slides and slide shows. A clean-room take on PowerPoint.",
		Extensions:  []string{"deckcraft", "pptx", "potx", "ppsx"},
		Data: []dataLoc{
			{"config", "DeckCraft", "DeckCraft", "deckcraft"}, // ui.json (+ Recovery on macOS/Windows)
			{"data", "", "", "deckcraft"},                     // Linux recovery
		}},
	{ID: "soundcraft", Name: "SoundCraft", Repo: "soundcraft", Asset: "soundcraft", Group: "Etc.", Category: "Audio production",
		Description: "Recording, editing and mixing. A clean-room take on Pro Tools.",
		Extensions:  []string{"scraft", "wav", "wave", "bwf", "aif", "aiff", "aifc", "flac", "mp3", "mid", "midi", "smf"},
		Data:        []dataLoc{{"config", "SoundCraft", "SoundCraft", "soundcraft"}}},
	{ID: "cadcraft", Name: "CADCraft", Repo: "cadcraft", Asset: "cadcraft", Group: "Etc.", Category: "CAD & drafting",
		Description: "Computer-aided design and drafting with DXF and DWG files. A clean-room take on AutoCAD.",
		Extensions:  []string{"dxf", "dwg", "dwt", "svg", "svgz"}},
}

func findApp(id string) (App, bool) {
	for _, a := range Catalog {
		if a.ID == id {
			return a, true
		}
	}
	return App{}, false
}
