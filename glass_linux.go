package main

// osGlass reports whether the window shows the desktop through OS blur.
// WebKitGTK has no such material, so the window stays solid.
func osGlass() bool {
	return false
}
