package main

// acrylicBuild reports whether the Windows build supports the WebView2 Acrylic
// backdrop: Windows 11 22H2 (22621) and later. It is untagged so the test runs
// on every platform; only glass_windows.go calls it.
func acrylicBuild(build uint32) bool {
	return build >= 22621
}
