// Package version holds the build-time version string for octo.
package version

// Version is the current version of octo. It defaults to "dev" for local
// builds and can be overridden at build time, e.g.:
//
//	go build -ldflags "-X github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/version.Version=v1.2.3"
var Version = "dev"
