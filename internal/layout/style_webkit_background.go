package layout

// normalizeWebkitBackgroundClip remaps -webkit-background-clip onto
// background-clip. The existing background painter reads that name.
func normalizeWebkitBackgroundClip(prop string) (string, bool) {
	switch prop {
	case "-webkit-background-clip":
		return "background-clip", true
	default:
		return "", false
	}
}

// normalizeWebkitBackgroundOrigin remaps the prefixed property onto
// background-origin. Image placement stays on the unprefixed painter.
func normalizeWebkitBackgroundOrigin(prop string) (string, bool) {
	switch prop {
	case "-webkit-background-origin":
		return "background-origin", true
	default:
		return "", false
	}
}

// normalizeWebkitBackgroundSize remaps -webkit-background-size onto background-size.
func normalizeWebkitBackgroundSize(prop string) (string, bool) {
	switch prop {
	case "-webkit-background-size":
		return "background-size", true
	default:
		return "", false
	}
}
