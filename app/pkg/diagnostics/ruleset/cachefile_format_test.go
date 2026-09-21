package ruleset

// Cross-version compatibility of sing-box's rule-set cache entry.
//
// This is the format the panel reads to evaluate a REMOTE rule set, and it is
// written by whatever sing-box the operator is running — not by the library
// this panel links against. sing-box 1.14 bumped the entry version from 1 to 2
// and appended a URLHash field, so "does the pinned decoder still read a cache
// written by 1.14?" is a question with a yes/no answer and real consequences:
// a no means every geosite-* rule silently reports `parse_error` on a router,
// which is exactly where this feature is supposed to work.
//
// The payloads below are built by hand rather than by importing the newer
// adapter, because Go cannot link two versions of one module — the same
// constraint documented for the generated option inventories.

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/sagernet/sing/common/varbin"
)

// encodeV1 reproduces sing-box 1.12's adapter.SavedBinary.MarshalBinary:
// version byte 1, then varbin-encoded content, unix seconds, varbin etag.
func encodeV1(t *testing.T, content []byte, updated time.Time, etag string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.BigEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if err := varbin.Write(&buffer, binary.BigEndian, content); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buffer, binary.BigEndian, updated.Unix()); err != nil {
		t.Fatal(err)
	}
	if err := varbin.Write(&buffer, binary.BigEndian, etag); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

// encodeV2 reproduces sing-box 1.14's version: the byte is 2, content is
// written as an explicit uvarint length plus raw bytes, and a URLHash field is
// appended after the etag.
func encodeV2(t *testing.T, content []byte, updated time.Time, etag string, urlHash []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.BigEndian, uint8(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := varbin.WriteUvarint(&buffer, uint64(len(content))); err != nil {
		t.Fatal(err)
	}
	buffer.Write(content)
	if err := binary.Write(&buffer, binary.BigEndian, updated.Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := varbin.WriteUvarint(&buffer, uint64(len(etag))); err != nil {
		t.Fatal(err)
	}
	buffer.WriteString(etag)
	if _, err := varbin.WriteUvarint(&buffer, uint64(len(urlHash))); err != nil {
		t.Fatal(err)
	}
	buffer.Write(urlHash)
	return buffer.Bytes()
}

func TestSavedBinaryReadsBothCacheVersions(t *testing.T) {
	content := []byte("SRS\x03pretend-this-is-a-compiled-rule-set")
	updated := time.Unix(1_770_000_000, 0)

	cases := []struct {
		name    string
		encoded []byte
	}{
		{"v1 (sing-box 1.12)", encodeV1(t, content, updated, `W/"abc123"`)},
		{"v2 (sing-box 1.14)", encodeV2(t, content, updated, `W/"abc123"`, []byte{0xde, 0xad, 0xbe, 0xef})},
		// A future version that appends further fields must still yield the
		// content and timestamp, because the decoder stops after them.
		{"v3 with extra trailing data", append(
			encodeV2(t, content, updated, `W/"abc123"`, []byte{0x01}),
			[]byte{0x99, 0x98, 0x97}...)},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var saved savedBinary
			if err := saved.unmarshal(test.encoded); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if !bytes.Equal(saved.Content, content) {
				t.Fatalf("content = %q, want %q", saved.Content, content)
			}
			if !saved.LastUpdated.Equal(updated) {
				t.Fatalf("last updated = %s, want %s", saved.LastUpdated, updated)
			}
		})
	}
}

func TestSavedBinaryRejectsTruncatedEntry(t *testing.T) {
	// A torn entry — the cache was copied mid-write — must surface as a decode
	// error so the set reports parse_error, never as empty content that would
	// read as "this rule set matches nothing".
	full := encodeV2(t, []byte("SRS\x03content"), time.Unix(1_770_000_000, 0), "", nil)
	for _, cut := range []int{1, 3, len(full) - 4} {
		var saved savedBinary
		if err := saved.unmarshal(full[:cut]); err == nil {
			t.Fatalf("truncated at %d decoded without error: %+v", cut, saved)
		}
	}
}
