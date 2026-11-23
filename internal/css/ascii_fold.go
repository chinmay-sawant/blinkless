package css

// IndexASCIIFold returns the byte index of the first occurrence of needle in
// s, comparing ASCII letters case-insensitively, or -1. needle must be ASCII
// lowercase. Unlike indexing strings.ToLower(s), the returned offset is valid
// in s itself: ToLower changes the byte length of some non-ASCII runes (for
// example U+0130), which shifts every later offset.
func IndexASCIIFold(s, needle string) int {
	n := len(needle)
	if n == 0 {
		return 0
	}

	for i := 0; i+n <= len(s); i++ {
		j := 0
		for ; j < n; j++ {
			c := s[i+j]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}

			if c != needle[j] {
				break
			}
		}

		if j == n {
			return i
		}
	}

	return -1
}
