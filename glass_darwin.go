package main

// osGlass reports whether the window shows the desktop through OS vibrancy.
// Server mode runs in a browser, which has no window to blur.
func osGlass() bool {
	return !serverMode
}
