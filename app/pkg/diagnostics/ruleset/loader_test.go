package ruleset

import (
	"bytes"
	"encoding/binary"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sagernet/bbolt"
	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
	"github.com/sagernet/sing/common/varbin"
)

func optionsWithSets(sets []option.RuleSet, cacheFile *option.CacheFileOptions) *option.Options {
	options := &option.Options{
		Route: &option.RouteOptions{RuleSet: sets},
	}
	if cacheFile != nil {
		options.Experimental = &option.ExperimentalOptions{CacheFile: cacheFile}
	}
	return options
}

func TestInlineRuleSet(t *testing.T) {
	loader := NewLoader(nil, optionsWithSets([]option.RuleSet{{
		Type: C.RuleSetTypeInline,
		Tag:  "inline-block",
		InlineOptions: option.PlainRuleSet{
			Rules: []option.HeadlessRule{{
				Type: "default",
				DefaultOptions: option.DefaultHeadlessRule{
					DomainSuffix: badoption.Listable[string]{"ads.example"},
				},
			}},
		},
	}}, nil))
	defer loader.Close()

	set := loader.Get("inline-block")
	if !set.Available {
		t.Fatalf("inline set unavailable: %s %s", set.Reason, set.Detail)
	}
	if set.RuleCount != 1 {
		t.Errorf("RuleCount = %d, want 1", set.RuleCount)
	}
	assertVerdict(t, set.Match(Target{Domain: "x.ads.example"}), VerdictYes, "inline hit")
	assertVerdict(t, set.Match(Target{Domain: "example.com"}), VerdictNo, "inline miss")
}

// Every unavailable path must answer Unknown, never No. This is the property
// the whole package exists to preserve: "I could not read geoip-cn" and "the
// address is not in geoip-cn" send a connection to different outbounds.
func TestUnavailableSetsAnswerUnknown(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "absent.srs")

	cases := []struct {
		name    string
		options *option.Options
		tag     string
		reason  Reason
	}{
		{
			name:    "unknown tag",
			options: optionsWithSets(nil, nil),
			tag:     "nope",
			reason:  ReasonUnknownTag,
		},
		{
			name: "remote set with no cache file configured",
			options: optionsWithSets([]option.RuleSet{{
				Type: C.RuleSetTypeRemote, Tag: "remote", Format: C.RuleSetFormatBinary,
			}}, nil),
			tag:    "remote",
			reason: ReasonCacheDisabled,
		},
		{
			name: "remote set with cache_file disabled",
			options: optionsWithSets([]option.RuleSet{{
				Type: C.RuleSetTypeRemote, Tag: "remote", Format: C.RuleSetFormatBinary,
			}}, &option.CacheFileOptions{Enabled: false, Path: "cache.db"}),
			tag:    "remote",
			reason: ReasonCacheDisabled,
		},
		{
			name: "local set whose file is gone",
			options: optionsWithSets([]option.RuleSet{{
				Type: C.RuleSetTypeLocal, Tag: "local", Format: C.RuleSetFormatBinary,
				LocalOptions: option.LocalRuleSet{Path: missingPath},
			}}, nil),
			tag:    "local",
			reason: ReasonFileMissing,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loader := NewLoader(nil, tc.options)
			defer loader.Close()

			set := loader.Get(tc.tag)
			if set.Available {
				t.Fatal("set reported available")
			}
			if set.Reason != tc.reason {
				t.Errorf("Reason = %q, want %q (detail: %s)", set.Reason, tc.reason, set.Detail)
			}
			assertVerdict(t, set.Match(Target{Domain: "example.com"}), VerdictUnknown, "unavailable set")
		})
	}
}

func TestRemoteSetMissingFromCache(t *testing.T) {
	cachePath := writeCache(t, nil)
	loader := NewLoader(nil, optionsWithSets([]option.RuleSet{{
		Type: C.RuleSetTypeRemote, Tag: "never-downloaded", Format: C.RuleSetFormatBinary,
	}}, &option.CacheFileOptions{Enabled: true, Path: cachePath}))
	defer loader.Close()

	set := loader.Get("never-downloaded")
	if set.Reason != ReasonNotCached {
		t.Errorf("Reason = %q, want %q", set.Reason, ReasonNotCached)
	}
}

// The full production path: a binary .srs, stored the way sing-box stores a
// downloaded rule set, read back out of a bbolt cache file.
func TestRemoteBinarySetFromCache(t *testing.T) {
	content := writeSRS(t, option.PlainRuleSet{
		Rules: []option.HeadlessRule{{
			Type: "default",
			DefaultOptions: option.DefaultHeadlessRule{
				DomainSuffix: badoption.Listable[string]{"netflix.com"},
				IPCIDR:       badoption.Listable[string]{"198.51.100.0/24"},
			},
		}},
	})

	updated := time.Now().Add(-36 * time.Hour).Truncate(time.Second)
	cachePath := writeCache(t, map[string]savedBinary{
		"geosite-netflix": {Content: content, LastUpdated: updated, LastEtag: "W/\"abc\""},
	})

	loader := NewLoader(nil, optionsWithSets([]option.RuleSet{{
		Type: C.RuleSetTypeRemote, Tag: "geosite-netflix", Format: C.RuleSetFormatBinary,
	}}, &option.CacheFileOptions{Enabled: true, Path: cachePath}))
	defer loader.Close()

	set := loader.Get("geosite-netflix")
	if !set.Available {
		t.Fatalf("set unavailable: %s %s", set.Reason, set.Detail)
	}
	if !set.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt = %s, want %s", set.UpdatedAt, updated)
	}
	assertVerdict(t, set.Match(Target{Domain: "www.netflix.com"}), VerdictYes, "suffix hit")
	assertVerdict(t, set.Match(Target{
		Domain: "example.com", IP: netip.MustParseAddr("203.0.113.9"),
	}), VerdictNo, "both halves miss")

	// The rule matches on domain OR address, so a target with no address is
	// undecidable even though its domain plainly misses: the address it
	// resolves to could still be in the set. Answering "no" here is how a
	// diagnostic quietly starts lying.
	assertVerdict(t, set.Match(Target{Domain: "example.com"}), VerdictUnknown, "domain misses, address unknown")
}

// cache_id nests every bucket inside a bucket of that name
// (experimental/cachefile/cache.go:209). Reading the top level on such a config
// finds nothing, which would look exactly like "never downloaded".
func TestCacheIDNesting(t *testing.T) {
	content := writeSRS(t, option.PlainRuleSet{
		Rules: []option.HeadlessRule{{
			Type:           "default",
			DefaultOptions: option.DefaultHeadlessRule{Domain: badoption.Listable[string]{"example.com"}},
		}},
	})
	cachePath := writeCacheWithID(t, "profile-2", map[string]savedBinary{
		"set": {Content: content, LastUpdated: time.Now()},
	})

	sets := []option.RuleSet{{Type: C.RuleSetTypeRemote, Tag: "set", Format: C.RuleSetFormatBinary}}

	withID := NewLoader(nil, optionsWithSets(sets, &option.CacheFileOptions{
		Enabled: true, Path: cachePath, CacheID: "profile-2",
	}))
	defer withID.Close()
	if set := withID.Get("set"); !set.Available {
		t.Fatalf("with cache_id: unavailable (%s %s)", set.Reason, set.Detail)
	}

	withoutID := NewLoader(nil, optionsWithSets(sets, &option.CacheFileOptions{
		Enabled: true, Path: cachePath,
	}))
	defer withoutID.Close()
	if set := withoutID.Get("set"); set.Available {
		t.Error("without cache_id: found a set that lives under one")
	}
}

func TestDecodeSourceFormat(t *testing.T) {
	rules, err := Decode(C.RuleSetFormatSource, []byte(`{
		"version": 1,
		"rules": [{"domain_suffix": ["example.com"]}]
	}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
}

// The format field is a hint; the magic bytes decide. A config assembled in
// code never runs option.RuleSet.UnmarshalJSON, so `format` can be empty even
// for binary content.
func TestDecodeSniffsBinaryWithoutFormat(t *testing.T) {
	content := writeSRS(t, option.PlainRuleSet{
		Rules: []option.HeadlessRule{{
			Type:           "default",
			DefaultOptions: option.DefaultHeadlessRule{Domain: badoption.Listable[string]{"example.com"}},
		}},
	})
	if _, err := Decode("", content); err != nil {
		t.Fatalf("Decode with empty format: %v", err)
	}
}

// --- fixtures ---------------------------------------------------------------

func writeSRS(t *testing.T, plain option.PlainRuleSet) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := srs.Write(&buffer, plain, C.RuleSetVersion3); err != nil {
		t.Fatalf("srs.Write: %v", err)
	}
	return buffer.Bytes()
}

func writeCache(t *testing.T, entries map[string]savedBinary) string {
	t.Helper()
	return writeCacheWithID(t, "", entries)
}

func writeCacheWithID(t *testing.T, cacheID string, entries map[string]savedBinary) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cache.db")

	db, err := bbolt.Open(path, 0o644, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("bbolt.Open: %v", err)
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		var parent interface {
			CreateBucketIfNotExists([]byte) (*bbolt.Bucket, error)
		} = tx
		if cacheID != "" {
			idBucket, bucketErr := tx.CreateBucketIfNotExists([]byte(cacheID))
			if bucketErr != nil {
				return bucketErr
			}
			parent = idBucket
		}
		bucket, bucketErr := parent.CreateBucketIfNotExists(bucketRuleSet)
		if bucketErr != nil {
			return bucketErr
		}
		for tag, saved := range entries {
			encoded, encodeErr := marshalSavedBinary(saved)
			if encodeErr != nil {
				return encodeErr
			}
			if putErr := bucket.Put([]byte(tag), encoded); putErr != nil {
				return putErr
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	if err = db.Close(); err != nil {
		t.Fatalf("close cache: %v", err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatalf("stat cache: %v", err)
	}
	return path
}

// marshalSavedBinary mirrors adapter.SavedBinary.MarshalBinary
// (adapter/experimental.go:63-81). Written out here rather than imported: the
// adapter package reaches sing-tun, which this binary does not otherwise link.
func marshalSavedBinary(saved savedBinary) ([]byte, error) {
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.BigEndian, uint8(1)); err != nil {
		return nil, err
	}
	if err := varbin.Write(&buffer, binary.BigEndian, saved.Content); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.BigEndian, saved.LastUpdated.Unix()); err != nil {
		return nil, err
	}
	if err := varbin.Write(&buffer, binary.BigEndian, saved.LastEtag); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// The case that matters in production: sing-box is running and holds the cache
// file's lock. bbolt gives writers an exclusive flock, so even a read-only open
// blocks — the loader has to fall back to reading a copy, or a diagnostic would
// be unusable exactly while the thing being diagnosed is up.
func TestCacheReadWhileLocked(t *testing.T) {
	content := writeSRS(t, option.PlainRuleSet{
		Rules: []option.HeadlessRule{{
			Type:           "default",
			DefaultOptions: option.DefaultHeadlessRule{Domain: badoption.Listable[string]{"example.com"}},
		}},
	})
	cachePath := writeCache(t, map[string]savedBinary{
		"set": {Content: content, LastUpdated: time.Now()},
	})

	// Stand in for a running sing-box: hold the write lock for the duration.
	holder, err := bbolt.Open(cachePath, 0o644, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	defer holder.Close()

	loader := NewLoader(nil, optionsWithSets([]option.RuleSet{{
		Type: C.RuleSetTypeRemote, Tag: "set", Format: C.RuleSetFormatBinary,
	}}, &option.CacheFileOptions{Enabled: true, Path: cachePath}))
	defer loader.Close()

	set := loader.Get("set")
	if !set.Available {
		t.Fatalf("locked cache unreadable: %s %s", set.Reason, set.Detail)
	}
	assertVerdict(t, set.Match(Target{Domain: "example.com"}), VerdictYes, "read through the lock")
}

func TestCompileRejectsUnknownRuleType(t *testing.T) {
	if _, err := compile(option.HeadlessRule{Type: "nonsense"}); err == nil {
		t.Error("expected an error for an unknown rule type")
	}
}

func TestTagsListsEveryConfiguredSet(t *testing.T) {
	loader := NewLoader(nil, optionsWithSets([]option.RuleSet{
		{Type: C.RuleSetTypeRemote, Tag: "a", Format: C.RuleSetFormatBinary},
		{Type: C.RuleSetTypeRemote, Tag: "b", Format: C.RuleSetFormatBinary},
	}, nil))
	defer loader.Close()

	if got := len(loader.Tags()); got != 2 {
		t.Errorf("Tags() returned %d entries, want 2", got)
	}
}
