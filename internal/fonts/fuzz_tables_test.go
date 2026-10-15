package fonts

import "testing"

func FuzzTableDirectory(f *testing.F) {
	f.Add([]byte("OTTO"))
	f.Add([]byte{})

	f.Fuzz(func(_ *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}

		_, _ = parseTableDirectory(data)
	})
}
