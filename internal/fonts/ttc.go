package fonts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	ttcHeaderSize = 12
	ttcTagSize    = 4
	// maxTTCMembers bounds how many faces one collection may declare. Real
	// system collections hold two to ten.
	maxTTCMembers = 32
	// maxTTCTables bounds the table directory of one member.
	maxTTCTables = 256
)

var (
	errTTCNotCollection = errors.New("font: not a TrueType collection")
	errTTCBadHeader     = errors.New("font: malformed TrueType collection header")
	errTTCBadMember     = errors.New("font: malformed TrueType collection member")
)

// IsTTC reports whether data starts with the 'ttcf' collection tag.
func IsTTC(data []byte) bool {
	return len(data) >= ttcHeaderSize && bytes.Equal(data[:ttcTagSize], []byte("ttcf"))
}

// SplitTTC extracts every member of a TrueType collection as a standalone
// SFNT file. A consumer such as text.NewGoTextFaceSource reads one face per
// byte slice, so each member is rebuilt with its own table directory instead
// of sharing the collection's. Members with CFF outlines come out unchanged
// here and are rejected later by ParseTTF.
func SplitTTC(data []byte) ([][]byte, error) {
	if !IsTTC(data) {
		return nil, errTTCNotCollection
	}

	count := int(binary.BigEndian.Uint32(data[8:12]))
	if count < 1 || count > maxTTCMembers || ttcHeaderSize+4*count > len(data) {
		return nil, errTTCBadHeader
	}

	members := make([][]byte, 0, count)

	for i := range count {
		offset := int(binary.BigEndian.Uint32(data[ttcHeaderSize+4*i:]))

		face, err := extractTTCMember(data, offset)
		if err != nil {
			return nil, fmt.Errorf("member %d: %w", i, err)
		}

		members = append(members, face)
	}

	return members, nil
}

// extractTTCMember rebuilds the face whose offset table starts at offset.
func extractTTCMember(data []byte, offset int) ([]byte, error) {
	if offset < 0 || offset+sfntHeaderSize > len(data) {
		return nil, errTTCBadMember
	}

	numTables := int(binary.BigEndian.Uint16(data[offset+4:]))
	if numTables < 1 || numTables > maxTTCTables {
		return nil, errTTCBadMember
	}

	dirEnd := offset + sfntHeaderSize + sfntRecordSize*numTables
	if dirEnd > len(data) {
		return nil, errTTCBadMember
	}

	headerSize := sfntHeaderSize + sfntRecordSize*numTables
	total := headerSize

	type span struct{ start, length int }

	spans := make([]span, numTables)

	for t := range numTables {
		rec := data[offset+sfntHeaderSize+sfntRecordSize*t:]
		start := int(binary.BigEndian.Uint32(rec[8:12]))
		length := int(binary.BigEndian.Uint32(rec[12:16]))

		if start < 0 || length < 0 || start > len(data) || length > len(data)-start {
			return nil, errTTCBadMember
		}

		spans[t] = span{start: start, length: length}

		total += paddedLen(length)
		if total > maxSFNTSize {
			return nil, errTTCBadMember
		}
	}

	out := make([]byte, total)
	copy(out[:sfntHeaderSize], data[offset:offset+sfntHeaderSize])

	next := headerSize

	for t, sp := range spans {
		rec := data[offset+sfntHeaderSize+sfntRecordSize*t:]
		dst := out[sfntHeaderSize+sfntRecordSize*t:]

		copy(dst[:8], rec[:8]) // tag and checksum
		binary.BigEndian.PutUint32(dst[8:12], uint32(next))
		binary.BigEndian.PutUint32(dst[12:16], uint32(sp.length)) //nolint:gosec // length is bounded by len(data)

		copy(out[next:], data[sp.start:sp.start+sp.length])

		next += paddedLen(sp.length)
	}

	return out, nil
}
