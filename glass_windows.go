package main

import "golang.org/x/sys/windows"

// osGlass reports whether the window can show the desktop through OS blur.
// Windows 10 has no Acrylic backdrop in WebView2, and Wails' legacy blur-behind
// is not used, so those windows stay solid.
func osGlass() bool {
	return !serverMode && acrylicBuild(windows.RtlGetVersion().BuildNumber)
}
