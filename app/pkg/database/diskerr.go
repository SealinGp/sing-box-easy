package database

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sysinfo"
)

// diskErrorMarkers are the driver strings that a full or read-only filesystem
// produces. SQLite creates a rollback journal beside the database file for
// every write transaction; when that file cannot be created it reports
// SQLITE_CANTOPEN ("unable to open database file (14)"), which names the
// database and never mentions the disk. Reads keep working, so the failure
// looks like a permissions bug on the .db file — operators reliably chmod it
// and get nowhere.
var diskErrorMarkers = []string{
	"unable to open database file",
	"disk i/o error",
	"database or disk is full",
	"attempt to write a readonly database",
	"no space left on device",
	"read-only file system",
}

// AnnotateWriteError explains a database write failure in terms of the disk
// when the disk is what actually failed, and returns err unchanged otherwise.
//
// It only adds the explanation after confirming with statfs, so a genuine
// permissions problem is never mislabelled as a full disk.
func AnnotateWriteError(err error) error {
	if err == nil || !looksLikeDiskError(err) {
		return err
	}

	dir := filepath.Dir(Path())
	if dir == "" || dir == "." {
		return err
	}

	disks := sysinfo.CollectDisks(dir)
	if len(disks) == 0 {
		return err
	}
	disk := disks[0]

	where := disk.MountPoint
	if where == "" {
		where = dir
	}

	if disk.FreeBytes == 0 {
		return fmt.Errorf("%w — the filesystem holding the database (%s) is full: "+
			"%d%% used, 0 bytes free. SQLite cannot create its journal file, so every "+
			"write fails. Free space on %s and retry",
			err, where, int(disk.UsedPercent), where)
	}

	if isReadOnly(dir) {
		return fmt.Errorf("%w — the filesystem holding the database (%s) is mounted "+
			"read-only, so no write can succeed. Remount it read-write and retry",
			err, where)
	}

	return err
}

func looksLikeDiskError(err error) bool {
	message := strings.ToLower(err.Error())
	for _, marker := range diskErrorMarkers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

// isReadOnly is implemented per platform; see readonly_unix.go.
