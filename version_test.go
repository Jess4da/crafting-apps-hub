package main

import (
	"regexp"
	"testing"
)

// Official SemVer 2.0.0 regex (semver.org), without a leading "v".
var semver = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func TestVersionIsSemVer(t *testing.T) {
	if !semver.MatchString(version) {
		t.Fatalf("build/config.yml info.version %q is not SemVer (MAJOR.MINOR.PATCH)", version)
	}
}
