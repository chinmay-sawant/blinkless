package svg

import (
	"bytes"
	"strings"
)

// Link parameters (CSS Link Parameters) pass named values into a linked
// resource. The resource reads one with env(--name, fallback). The pinned
// canvas rasterizer does not evaluate env(), so the values are written into
// the SVG text before the raster call.
//
// A parameter list is a canonical string: one "name<TAB>value" pair per line.
// An empty list changes nothing, so a document with no link-parameters
// rasterizes as stored.

// ApplyLinkParams replaces each env(--name[, fallback]) in data whose name is
// in params with that parameter's value. An env() whose name is not listed
// is left as written. Data without "env(" is returned unchanged.
func ApplyLinkParams(data []byte, params string) []byte {
	if params == "" || !bytes.Contains(data, []byte("<svg")) || !bytes.Contains(data, []byte("env(")) {
		return data
	}

	values := parseLinkParamList(params)
	if len(values) == 0 {
		return data
	}

	src := string(data)

	var out strings.Builder

	out.Grow(len(src))

	for {
		start := strings.Index(src, "env(")
		if start < 0 {
			out.WriteString(src)

			break
		}

		end := matchingParen(src, start+len("env"))
		if end < 0 {
			out.WriteString(src)

			break
		}

		name := envName(src[start+len("env(") : end])
		if value, ok := values[name]; ok {
			out.WriteString(src[:start])
			out.WriteString(value)
		} else {
			out.WriteString(src[:end+1])
		}

		src = src[end+1:]
	}

	return []byte(out.String())
}

func parseLinkParamList(params string) map[string]string {
	values := map[string]string{}

	for _, line := range strings.Split(params, "\n") {
		name, value, ok := strings.Cut(line, "\t")
		if ok && name != "" {
			values[name] = value
		}
	}

	return values
}

// envName returns the --name at the start of an env() argument list.
func envName(args string) string {
	args = strings.TrimSpace(args)
	if i := strings.IndexByte(args, ','); i >= 0 {
		args = args[:i]
	}

	return strings.TrimSpace(args)
}

// matchingParen returns the index of the ")" that closes the "(" at open, or
// -1 when it is unbalanced.
func matchingParen(s string, open int) int {
	depth := 0

	for pos := open; pos < len(s); pos++ {
		switch s[pos] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return pos
			}
		}
	}

	return -1
}
