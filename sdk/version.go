package sdk

import "runtime/debug"

func commit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "<none>"
	}
	var rev string = "<none>"
	var dirty string = ""
	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			rev = v.Value
		}
		if v.Key == "vcs.modified" {
			if v.Value == "true" {
				dirty = "-dirty"
			} else {
				dirty = ""
			}
		}
	}
	return rev + dirty
}

func Version() string {
	return commit()
}
