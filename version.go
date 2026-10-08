package main

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed build/config.yml
var buildConfig []byte

// version is info.version from build/config.yml, the single SemVer source
// (it also feeds the Info.plist / info.json build assets).
var version = func() string {
	var c struct{ Info struct{ Version string } }
	if err := yaml.Unmarshal(buildConfig, &c); err != nil {
		panic(err)
	}
	return c.Info.Version
}()
