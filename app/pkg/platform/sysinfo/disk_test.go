package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCollectDisksReportsTheFilesystemBackingAPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("statfs is unix-only")
	}

	disks := CollectDisks(t.TempDir())
	if len(disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(disks))
	}

	d := disks[0]
	if d.TotalBytes == 0 {
		t.Error("total bytes is 0; statfs did not report a size")
	}
	if d.UsedBytes+d.FreeBytes > d.TotalBytes {
		t.Errorf("used(%d)+free(%d) exceeds total(%d)", d.UsedBytes, d.FreeBytes, d.TotalBytes)
	}
	if d.UsedPercent < 0 || d.UsedPercent > 100 {
		t.Errorf("used percent %v out of range", d.UsedPercent)
	}
}

func TestCollectDisksDeduplicatesPathsOnTheSameFilesystem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("statfs is unix-only")
	}

	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	// Both paths live on the same filesystem, so the card must not show the
	// same device twice.
	disks := CollectDisks(dir, nested)
	if len(disks) != 1 {
		t.Fatalf("expected paths on one filesystem to collapse to 1 entry, got %d", len(disks))
	}
}

func TestCollectDisksSkipsUnreadablePaths(t *testing.T) {
	// A missing path is a cosmetic problem — the About card must still render
	// the filesystems that could be read.
	disks := CollectDisks("/definitely/not/a/real/path", t.TempDir())
	if runtime.GOOS == "windows" {
		return
	}
	if len(disks) != 1 {
		t.Fatalf("expected the unreadable path to be dropped, got %d entries", len(disks))
	}
}

func TestCollectDisksWithNoPathsReturnsNil(t *testing.T) {
	if disks := CollectDisks(); disks != nil {
		t.Errorf("expected nil for no paths, got %v", disks)
	}
}

func TestParseMountsFindsTheLongestMatchingMountPoint(t *testing.T) {
	const mounts = `/dev/sdc2 /rom squashfs ro,relatime 0 0
/dev/sdd1 /overlay ext4 rw,relatime 0 0
overlayfs:/overlay / overlay rw,noatime 0 0
tmpfs /tmp tmpfs rw,nosuid 0 0
`
	table := parseMounts(mounts)

	tests := []struct {
		path       string
		wantMount  string
		wantDevice string
	}{
		{"/etc/sing-box", "/", "overlayfs:/overlay"},
		{"/overlay/upper/etc", "/overlay", "/dev/sdd1"},
		{"/tmp/cache.db", "/tmp", "tmpfs"},
		{"/rom", "/rom", "/dev/sdc2"},
	}

	for _, tt := range tests {
		mount, device := table.lookup(tt.path)
		if mount != tt.wantMount || device != tt.wantDevice {
			t.Errorf("lookup(%q) = (%q, %q), want (%q, %q)",
				tt.path, mount, device, tt.wantMount, tt.wantDevice)
		}
	}
}

func TestParseMountsUnescapesOctalSequences(t *testing.T) {
	// /proc/self/mounts escapes spaces as \040.
	table := parseMounts(`/dev/sda1 /mnt/my\040disk ext4 rw 0 0` + "\n")

	mount, device := table.lookup("/mnt/my disk/file")
	if mount != "/mnt/my disk" || device != "/dev/sda1" {
		t.Errorf("got (%q, %q), want (\"/mnt/my disk\", \"/dev/sda1\")", mount, device)
	}
}
