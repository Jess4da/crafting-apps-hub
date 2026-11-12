package main

import "testing"

func TestAcrylicBuild(t *testing.T) {
	trueCases := []uint32{22621, 26100} // Windows 11 22H2 and later support Acrylic backdrops
	falseCases := []uint32{22000, 19045, 17763}
	for _, build := range trueCases {
		if !acrylicBuild(build) {
			t.Errorf("acrylicBuild(%d) = false, want true", build)
		}
	}
	for _, build := range falseCases {
		if acrylicBuild(build) {
			t.Errorf("acrylicBuild(%d) = true, want false", build)
		}
	}
}
