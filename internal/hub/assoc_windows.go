//go:build windows

package hub

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const progIDPrefix = "CraftingApps."

func setDefaultApp(app App, exts []string, in *Installed) error {
	progID := progIDPrefix + app.ID
	base := `Software\Classes\` + progID
	if err := setString(registry.CURRENT_USER, base, "", app.Name); err != nil {
		return err
	}
	if err := setString(registry.CURRENT_USER, base+`\DefaultIcon`, "", `"`+in.Exe+`",0`); err != nil {
		return err
	}
	if err := setString(registry.CURRENT_USER, base+`\shell\open\command`, "", `"`+in.Exe+`" "%1"`); err != nil {
		return err
	}
	for _, ext := range exts {
		key := `Software\Classes\.` + ext
		if err := setString(registry.CURRENT_USER, key, "", progID); err != nil {
			return fmt.Errorf("%s: %w", ext, err)
		}
		ow, _, err := registry.CreateKey(registry.CURRENT_USER, key+`\OpenWithProgids`, registry.SET_VALUE)
		if err != nil {
			return err
		}
		err = ow.SetBinaryValue(progID, nil)
		ow.Close()
		if err != nil {
			return err
		}
	}
	// ponytail: an existing Explorer UserChoice (HKCU\...\FileExts\.ext\UserChoice)
	// still wins over HKCU Classes and cannot be written by design; the user has
	// to pick "Always" in Explorer's Open With once.
	notifyShell()
	return nil
}

// currentDefaults returns each extension's HKCU\Software\Classes\.ext default
// value (the ProgID it pointed at before us; absent means none was set).
func currentDefaults(app App, exts []string) map[string]string {
	prev := map[string]string{}
	progID := progIDPrefix + app.ID
	for _, ext := range exts {
		k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\.`+ext, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		if v, _, err := k.GetStringValue(""); err == nil && v != "" && !strings.EqualFold(v, progID) {
			prev[ext] = v
		}
		k.Close()
	}
	return prev
}

// restoreDefaults puts back the recorded ProgIDs (or clears ours where none
// was recorded) for exts that still point at us, then deletes our ProgID
// (SetDefaultApp rewrites it for any types kept). Another app's later choice
// is left alone.
func restoreDefaults(app App, exts []string, prev map[string]string) error {
	progID := progIDPrefix + app.ID
	for _, ext := range exts {
		key := `Software\Classes\.` + ext
		if k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.QUERY_VALUE|registry.SET_VALUE); err == nil {
			if v, _, err := k.GetStringValue(""); err == nil && strings.EqualFold(v, progID) {
				if p := prev[ext]; p != "" {
					k.SetStringValue("", p)
				} else {
					k.DeleteValue("")
				}
			}
			k.Close()
		}
		if k, err := registry.OpenKey(registry.CURRENT_USER, key+`\OpenWithProgids`, registry.SET_VALUE); err == nil {
			k.DeleteValue(progID)
			k.Close()
		}
	}
	base := `Software\Classes\` + progID
	for _, k := range []string{base + `\DefaultIcon`, base + `\shell\open\command`, base + `\shell\open`, base + `\shell`, base} {
		registry.DeleteKey(registry.CURRENT_USER, k)
	}
	notifyShell()
	return nil
}

func setString(root registry.Key, path, name, value string) error {
	k, _, err := registry.CreateKey(root, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(name, value)
}

var shChangeNotify = windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")

// notifyShell tells Explorer that file associations changed (SHCNE_ASSOCCHANGED).
func notifyShell() {
	const shcneAssocChanged, shcnfIDList = 0x08000000, 0x0000
	shChangeNotify.Call(shcneAssocChanged, shcnfIDList, 0, 0)
}
