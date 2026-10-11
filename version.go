package main

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed build/config.yml
var buildConfig []byte

// version is the hub's SemVer. Release builds set it from the git tag with
// -ldflags "-X main.version=X.Y.Z" (VERSION env, see the platform Taskfiles);
// local builds fall back to info.version in build/config.yml.
// Must stay uninitialized: -X is ignored for vars with a non-constant initializer.
var version string

func init() {
	if version != "" {
		return
	}
	var c struct{ Info struct{ Version string } }
	if err := yaml.Unmarshal(buildConfig, &c); err != nil {
		panic(err)
	}
	version = c.Info.Version
}
