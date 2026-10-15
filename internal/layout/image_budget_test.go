package layout

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func pngHeader(width, height uint32) []byte {
	payload := make([]byte, 13)
	binary.BigEndian.PutUint32(payload[0:4], width)
	binary.BigEndian.PutUint32(payload[4:8], height)
	payload[8] = 8
	payload[9] = 2

	chunk := make([]byte, 4+4+len(payload)+4)
	binary.BigEndian.PutUint32(chunk[0:4], uint32(len(payload))) //nolint:gosec // IHDR payload is 13 bytes
	copy(chunk[4:8], "IHDR")
	copy(chunk[8:21], payload)
	sum := crc32.ChecksumIEEE(chunk[4:21])
	binary.BigEndian.PutUint32(chunk[21:], sum)

	return append([]byte("\x89PNG\r\n\x1a\n"), chunk...)
}

func gifHeader(width, height uint16) []byte {
	header := []byte("GIF89a")
	header = append(header,
		byte(width), byte(width>>8),
		byte(height), byte(height>>8),
		0, 0, 0,
	)

	return header
}

func TestDecodeImageBytesRejectsExtremeHeaders(t *testing.T) {
	t.Parallel()

	cases := [][]byte{
		pngHeader(100000, 100000),
		gifHeader(20000, 20000),
	}

	for _, data := range cases {
		if _, err := decodeImageBytes(data); err == nil {
			t.Fatalf("decoded an over-budget header (%d bytes)", len(data))
		}

		got := applyImageFilterToImageWithInterp(data, []parsedFilter{{kind: filterBlur, val: 1}}, "srgb")
		if !bytes.Equal(got, data) {
			t.Fatal("filter replaced an image whose header is over the pixel budget")
		}
	}
}
