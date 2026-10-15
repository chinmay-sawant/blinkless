package load

import "testing"

func FuzzContentRange(f *testing.F) {
	f.Add("bytes 0-1/2", int64(2))
	f.Add("bytes 0-0/*", int64(-1))
	f.Add("bytes 5-1/9", int64(2))
	f.Add("", int64(0))

	f.Fuzz(func(_ *testing.T, header string, length int64) {
		if len(header) > 256 {
			return
		}

		_ = checkRangeConsistency(header, length)
	})
}
