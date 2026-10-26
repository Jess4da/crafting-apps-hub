//go:build (!windows && !linux && !darwin) || (darwin && !cgo)

package hub

import "errors"

func setDefaultApp(app App, exts []string, in *Installed) error {
	return errors.New("setting the default app is not supported on this platform")
}

func currentDefaults(app App, exts []string) map[string]string { return nil }

func restoreDefaults(app App, exts []string, prev map[string]string) error { return nil }
