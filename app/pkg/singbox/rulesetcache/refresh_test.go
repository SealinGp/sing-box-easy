package rulesetcache

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"

	"github.com/sagernet/bbolt"
)

func record(body string) []byte {
	var b bytes.Buffer
	b.WriteByte(2)
	var n [10]byte
	size := binary.PutUvarint(n[:], uint64(len(body)))
	b.Write(n[:size])
	b.WriteString(body)
	_ = binary.Write(&b, binary.BigEndian, int64(100))
	b.WriteString("\x00\x00")
	return b.Bytes()
}

func TestEvictRestorePreservesNamespaceAndOpaqueBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	original := record(`{"version":3,"rules":[{"domain_suffix":"example.com"}]}`)
	err = db.Update(func(tx *bbolt.Tx) error {
		root, _ := tx.CreateBucket([]byte("rule_set"))
		_ = root.Put([]byte("ai"), []byte("root stays"))
		parent, _ := tx.CreateBucket([]byte("\x00tenant"))
		b, _ := parent.CreateBucket([]byte("rule_set"))
		_ = b.Put([]byte("other"), []byte("other stays"))
		return b.Put([]byte("ai"), original)
	})
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	backup, err := Evict(path, "tenant", "ai")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatal("backup changed opaque record")
	}
	err = edit(path, "tenant", func(b *bbolt.Bucket) error {
		if b.Get([]byte("ai")) != nil || string(b.Get([]byte("other"))) != "other stays" {
			t.Fatal("wrong entry removed")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = Restore(path, "tenant", "ai", backup); err != nil {
		t.Fatal(err)
	}
	db, err = bbolt.Open(path, 0600, &bbolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_ = db.View(func(tx *bbolt.Tx) error {
		if string(tx.Bucket([]byte("rule_set")).Get([]byte("ai"))) != "root stays" {
			t.Fatal("root changed")
		}
		if !bytes.Equal(tx.Bucket([]byte("\x00tenant")).Bucket([]byte("rule_set")).Get([]byte("ai")), original) {
			t.Fatal("rollback changed bytes")
		}
		return nil
	})
}

func TestCorruptEntryIsNotEvicted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	db, _ := bbolt.Open(path, 0600, nil)
	_ = db.Update(func(tx *bbolt.Tx) error {
		b, _ := tx.CreateBucket([]byte("rule_set"))
		return b.Put([]byte("ai"), []byte{9, 1})
	})
	_ = db.Close()
	if _, err := Evict(path, "", "ai"); err == nil {
		t.Fatal("unsupported wire version accepted")
	}
	_ = edit(path, "", func(b *bbolt.Bucket) error {
		if !bytes.Equal(b.Get([]byte("ai")), []byte{9, 1}) {
			t.Fatal("corrupt entry was deleted")
		}
		return nil
	})
}

func TestContentHashIgnoresMetadata(t *testing.T) {
	a := record("same")
	b := append([]byte(nil), a...)
	b[len(b)-3] = 55
	if ContentHash(a) != ContentHash(b) {
		t.Fatal("metadata counted as content change")
	}
	if ContentHash(a) == ContentHash(record("different")) {
		t.Fatal("missed content change")
	}
}
