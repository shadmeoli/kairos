package version

import "runtime/debug"

// Tag may be overridden at build time with -ldflags.
var Tag = "dev"

func Current() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return Tag
}
