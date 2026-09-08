package ruleset

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"time"

	"github.com/sagernet/bbolt"
	"github.com/sagernet/sing/common/varbin"
)

// DefaultCachePath is what sing-box uses when experimental.cache_file.path is
// left unset.
const DefaultCachePath = "cache.db"

// cacheOpenTimeout bounds the wait for the cache file's lock.
//
// bbolt takes an exclusive flock for writers and a shared one for readers, so a
// read-only open CANNOT proceed while sing-box holds the file. Rather than
// block a request, the wait is short and the caller falls back to reading a
// copy. See OpenCacheFile.
const cacheOpenTimeout = 500 * time.Millisecond

var (
	bucketRuleSet = []byte("rule_set")
	// copySizeLimit caps the fallback copy. A production cache file is ~36MB;
	// anything far past that is not something to duplicate inside a request.
	copySizeLimit int64 = 256 << 20
)

// CacheFile reads sing-box's bbolt cache without disturbing it.
type CacheFile struct {
	db      *bbolt.DB
	cacheID []byte
	// tempPath is set when the file had to be copied, and is removed on Close.
	tempPath string
}

// OpenCacheFile opens the cache read-only.
//
// Two paths, in order:
//
//  1. Open the real file read-only. This succeeds whenever sing-box is stopped
//     — which is the case that matters most, since a stopped sing-box is
//     exactly when its own answer is unavailable and this offline evaluation is
//     the only thing left.
//  2. If the lock is held, copy the file and open the copy. A copy taken while
//     bbolt is mid-write can be torn; that surfaces as a decode error on the
//     affected set and is reported as such, never as a non-match.
//
// The file is never opened for writing. Corrupting a user's rule-set cache to
// answer a diagnostic question would be a poor trade.
func OpenCacheFile(path string, cacheID string) (*CacheFile, error) {
	if path == "" {
		return nil, errors.New("no cache file path")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	options := &bbolt.Options{ReadOnly: true, Timeout: cacheOpenTimeout}

	db, err := bbolt.Open(path, 0o400, options)
	if err == nil {
		return newCacheFile(db, cacheID, ""), nil
	}

	tempPath, copyErr := copyForReading(path)
	if copyErr != nil {
		// Report the original lock failure: it is the one that explains the
		// situation, and the copy error is a consequence of it.
		return nil, err
	}

	db, copyOpenErr := bbolt.Open(tempPath, 0o400, options)
	if copyOpenErr != nil {
		os.Remove(tempPath)
		return nil, copyOpenErr
	}
	return newCacheFile(db, cacheID, tempPath), nil
}

func newCacheFile(db *bbolt.DB, cacheID string, tempPath string) *CacheFile {
	cache := &CacheFile{db: db, tempPath: tempPath}
	if cacheID != "" {
		cache.cacheID = []byte(cacheID)
	}
	return cache
}

// LoadRuleSet returns the stored content for a tag, or nil when absent.
//
// This mirrors sing-box's own reader (experimental/cachefile/cache.go:287),
// including the cache_id nesting: with a cache_id set, every bucket lives
// inside a bucket named by that id, so reading the top-level bucket would
// silently find nothing on a config that uses one.
func (c *CacheFile) LoadRuleSet(tag string) *savedBinary {
	if c == nil || c.db == nil {
		return nil
	}
	var saved savedBinary
	err := c.db.View(func(t *bbolt.Tx) error {
		bucket := c.bucket(t, bucketRuleSet)
		if bucket == nil {
			return os.ErrNotExist
		}
		content := bucket.Get([]byte(tag))
		if len(content) == 0 {
			return os.ErrNotExist
		}
		return saved.unmarshal(content)
	})
	if err != nil {
		return nil
	}
	return &saved
}

func (c *CacheFile) bucket(t *bbolt.Tx, key []byte) *bbolt.Bucket {
	if c.cacheID == nil {
		return t.Bucket(key)
	}
	bucket := t.Bucket(c.cacheID)
	if bucket == nil {
		return nil
	}
	return bucket.Bucket(key)
}

// Close releases the handle and removes the copy, if one was made.
func (c *CacheFile) Close() {
	if c == nil {
		return
	}
	if c.db != nil {
		c.db.Close()
		c.db = nil
	}
	if c.tempPath != "" {
		os.Remove(c.tempPath)
		c.tempPath = ""
	}
}

func copyForReading(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > copySizeLimit {
		return "", errors.New("cache file too large to copy")
	}

	source, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()

	target, err := os.CreateTemp("", "sing-box-cache-*.db")
	if err != nil {
		return "", err
	}
	tempPath := target.Name()

	if _, err = io.Copy(target, io.LimitReader(source, copySizeLimit)); err != nil {
		target.Close()
		os.Remove(tempPath)
		return "", err
	}
	if err = target.Close(); err != nil {
		os.Remove(tempPath)
		return "", err
	}
	return tempPath, nil
}

// savedBinary mirrors adapter.SavedBinary's wire format
// (adapter/experimental.go:57-107).
//
// Redeclared rather than imported for the same reason buildRules avoids
// route/rule: package adapter reaches common/process, which imports sing-tun,
// and this panel is not otherwise linked against the sing-box runtime. The
// format is three fields behind a version byte, and a mismatch would surface
// immediately as a decode failure on every set rather than as bad data.
type savedBinary struct {
	Content     []byte
	LastUpdated time.Time
	LastEtag    string
}

func (s *savedBinary) unmarshal(data []byte) error {
	reader := bytes.NewReader(data)

	var version uint8
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return err
	}
	if err := varbin.Read(reader, binary.BigEndian, &s.Content); err != nil {
		return err
	}
	var lastUpdated int64
	if err := binary.Read(reader, binary.BigEndian, &lastUpdated); err != nil {
		return err
	}
	s.LastUpdated = time.Unix(lastUpdated, 0)
	// The etag is not read: it is the last field, it is only used to make
	// conditional requests, and a future field appended after it must not
	// break the fields above.
	return nil
}
