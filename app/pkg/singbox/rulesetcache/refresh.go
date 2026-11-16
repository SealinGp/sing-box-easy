// Package rulesetcache invalidates one remote rule set while the core is stopped.
package rulesetcache

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/sagernet/bbolt"
)

// Evict preserves the exact wire bytes for rollback, including fields introduced
// by newer cores. Callers must stop sing-box first; a held lock fails closed.
func Evict(path, id, tag string) ([]byte, error) {
	previous, err := EvictMany(path, id, []string{tag})
	return previous[tag], err
}

// EvictMany validates every selected entry before atomically deleting any.
func EvictMany(path, id string, tags []string) (map[string][]byte, error) {
	previous := make(map[string][]byte, len(tags))
	err := edit(path, id, func(b *bbolt.Bucket) error {
		if b == nil {
			return fmt.Errorf("rule-set cache is empty")
		}
		for _, tag := range tags {
			previous[tag] = append([]byte(nil), b.Get([]byte(tag))...)
			if len(previous[tag]) == 0 {
				return fmt.Errorf("rule set %q has no cached content to refresh", tag)
			}
			if _, err := content(previous[tag]); err != nil {
				return err
			}
		}
		for _, tag := range tags {
			if err := b.Delete([]byte(tag)); err != nil {
				return err
			}
		}
		return nil
	})
	return previous, err
}

// ContentHash ignores update timestamps and ETags when reporting a change.
// Evict validates the record first, so this accepts only its returned bytes.
func ContentHash(raw []byte) [32]byte { body, _ := content(raw); return sha256.Sum256(body) }

func content(raw []byte) ([]byte, error) {
	if len(raw) < 2 || raw[0] > 2 {
		return nil, fmt.Errorf("unsupported cached rule-set record")
	}
	r := bytes.NewReader(raw[1:])
	n, err := binary.ReadUvarint(r)
	if err != nil || n == 0 || n > uint64(r.Len()) || uint64(r.Len())-n < 8 {
		return nil, fmt.Errorf("invalid cached rule-set record")
	}
	offset := len(raw) - r.Len()
	return raw[offset : offset+int(n)], nil
}

// Restore replaces only this entry; every other cache bucket stays untouched.
func Restore(path, id, tag string, previous []byte) error {
	return RestoreMany(path, id, map[string][]byte{tag: previous})
}

func RestoreMany(path, id string, previous map[string][]byte) error {
	return edit(path, id, func(b *bbolt.Bucket) error {
		if b == nil {
			return fmt.Errorf("rule-set cache bucket disappeared")
		}
		for tag, raw := range previous {
			if err := b.Put([]byte(tag), raw); err != nil {
				return err
			}
		}
		return nil
	})
}

func edit(path, id string, fn func(*bbolt.Bucket) error) error {
	// Do not create a new database when the configured path is wrong.
	if _, err := os.Stat(path); err != nil {
		return err
	}
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bbolt.Tx) error {
		var bucket *bbolt.Bucket
		if id == "" {
			bucket = tx.Bucket([]byte("rule_set"))
		} else if parent := tx.Bucket(append([]byte{0}, []byte(id)...)); parent != nil {
			bucket = parent.Bucket([]byte("rule_set"))
		}
		return fn(bucket)
	})
}
