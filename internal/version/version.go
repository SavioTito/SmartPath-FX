package version

// Version is the current binary version.
// Can be overridden at build time via:
//
//	-ldflags "-X github.com/saviotito/currency-router/internal/version.Version=$(git rev-parse --short HEAD)"
var Version = "v0.2.0"
