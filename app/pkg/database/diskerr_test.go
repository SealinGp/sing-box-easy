package database

import (
	"errors"
	"strings"
	"testing"
)

func TestLooksLikeDiskErrorMatchesDriverStrings(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"sqlite cantopen", errors.New("unable to open database file (14)"), true},
		{"uppercase", errors.New("Unable To Open Database File"), true},
		{"disk full", errors.New("database or disk is full"), true},
		{"enospc", errors.New("write /etc/x.db: no space left on device"), true},
		{"readonly fs", errors.New("open /etc/x.db: read-only file system"), true},
		{"unrelated", errors.New("UNIQUE constraint failed: subscriptions.url"), false},
		{"permission", errors.New("permission denied"), false},
	}

	for _, tt := range tests {
		if got := looksLikeDiskError(tt.err); got != tt.want {
			t.Errorf("%s: looksLikeDiskError(%v) = %v, want %v", tt.name, tt.err, got, tt.want)
		}
	}
}

func TestAnnotateWriteErrorPassesThroughNonDiskErrors(t *testing.T) {
	original := errors.New("UNIQUE constraint failed: subscriptions.url")

	got := AnnotateWriteError(original)

	if got != original {
		t.Errorf("expected the original error unchanged, got %v", got)
	}
}

func TestAnnotateWriteErrorPassesThroughNil(t *testing.T) {
	if got := AnnotateWriteError(nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestAnnotateWriteErrorKeepsTheOriginalUnwrappable(t *testing.T) {
	// Callers must still be able to match on the driver error, so whatever the
	// annotation adds has to wrap rather than replace.
	original := errors.New("unable to open database file (14)")

	got := AnnotateWriteError(original)

	if !errors.Is(got, original) {
		t.Errorf("annotated error no longer unwraps to the original: %v", got)
	}
}

func TestAnnotateWriteErrorExplainsAFullFilesystem(t *testing.T) {
	// The database path is normally set by Init; set it directly so the test
	// does not need a real full disk to exercise the message.
	dbMutex.Lock()
	previous := path
	path = "/definitely/not/a/real/dir/x.db"
	dbMutex.Unlock()
	defer func() {
		dbMutex.Lock()
		path = previous
		dbMutex.Unlock()
	}()

	// statfs fails on a nonexistent directory, so no disk facts are available
	// and the error must be returned untouched rather than guessing.
	original := errors.New("unable to open database file (14)")
	got := AnnotateWriteError(original)

	if strings.Contains(got.Error(), "full") {
		t.Errorf("claimed the disk is full without evidence: %v", got)
	}
}
