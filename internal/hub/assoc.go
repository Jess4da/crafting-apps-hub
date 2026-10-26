package hub

import (
	"fmt"
	"maps"
	"slices"
)

// SetDefaultApp makes the installed app the OS default handler for exactly
// exts (a subset of its catalog Extensions). Types no longer listed go back to
// the handler recorded when they were claimed; an empty exts restores all.
// The choice is persisted, so Install re-applies it after an update moves Exe.
func (h *Hub) SetDefaultApp(id string, exts []string) (AppState, error) {
	app, ok := findApp(id)
	if !ok {
		return AppState{}, fmt.Errorf("unknown app %q", id)
	}
	for _, e := range exts {
		if !slices.Contains(app.Extensions, e) {
			return AppState{}, fmt.Errorf("%s does not open .%s files", app.Name, e)
		}
	}
	h.mu.Lock()
	in, ok := h.st.Installed[id]
	var rec Installed
	if ok {
		rec = *in
	}
	h.mu.Unlock()
	if !ok {
		return AppState{}, fmt.Errorf("%s is not installed", app.Name)
	}
	var added, removed []string
	for _, e := range exts {
		if !slices.Contains(rec.DefaultExts, e) {
			added = append(added, e)
		}
	}
	for _, e := range rec.DefaultExts {
		if !slices.Contains(exts, e) {
			removed = append(removed, e)
		}
	}
	// Remember who handled newly claimed files, keeping any older record so a
	// re-ticked type still goes back to its pre-Craft handler.
	prev := maps.Clone(rec.PrevDefaults)
	if prev == nil {
		prev = map[string]string{}
	}
	for k, v := range currentDefaults(app, added) {
		if _, seen := prev[k]; !seen {
			prev[k] = v
		}
	}
	// Restore first: on Windows/Linux it drops our registration, which the
	// set below rewrites for the kept types (a shared MIME type stays ours).
	if len(removed) > 0 {
		if err := restoreDefaults(app, removed, prev); err != nil {
			return AppState{}, fmt.Errorf("could not restore the previous default apps: %w", err)
		}
	}
	if len(exts) > 0 {
		if err := setDefaultApp(app, exts, &rec); err != nil {
			return AppState{}, fmt.Errorf("could not make %s the default app: %w", app.Name, err)
		}
	} else {
		prev = nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	// The app may have been uninstalled or updated while the OS calls ran.
	if cur, ok := h.st.Installed[id]; ok && cur.Exe == rec.Exe {
		cur.DefaultExts, cur.PrevDefaults = slices.Clone(exts), prev
		if err := h.saveLocked(); err != nil {
			return AppState{}, err
		}
	}
	return h.stateLocked(app), nil
}
