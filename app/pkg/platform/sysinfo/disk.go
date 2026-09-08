package sysinfo

import (
	"os"
	"strconv"
	"strings"
)

// DiskUsage describes the filesystem backing one path the panel writes to.
//
// It exists because a full filesystem surfaces to the operator as an opaque
// driver error — SQLite reports SQLITE_CANTOPEN(14) when it cannot create its
// journal, which reads like a permissions bug and is not one. Showing free
// space next to the paths we write makes the real cause self-evident.
type DiskUsage struct {
	// Path is the queried path (e.g. the sing-box config directory).
	Path string `json:"path"`
	// MountPoint is the filesystem's mount point ("/overlay"), empty when
	// /proc/self/mounts is unavailable.
	MountPoint string `json:"mount_point"`
	// Device is the backing device ("/dev/sdd1", "tmpfs"), empty when unknown.
	Device string `json:"device"`
	// TotalBytes is the filesystem size.
	TotalBytes uint64 `json:"total_bytes"`
	// UsedBytes counts blocks in use, including the root-reserved ones.
	UsedBytes uint64 `json:"used_bytes"`
	// FreeBytes is what an unprivileged writer can still consume, which is
	// what actually matters for "will the next write fail".
	FreeBytes uint64 `json:"free_bytes"`
	// UsedPercent is used/(used+free)*100, matching what `df` prints.
	UsedPercent float64 `json:"used_percent"`
}

const mountsPath = "/proc/self/mounts"

// fsStat is the platform-independent result of a statfs call, in bytes.
type fsStat struct {
	total uint64
	used  uint64
	free  uint64
}

// CollectDisks reports usage for the filesystems backing the given paths,
// collapsing paths that share a filesystem so the same device is never listed
// twice. Paths that cannot be stat'ed are dropped rather than failing the call.
func CollectDisks(paths ...string) []DiskUsage {
	if len(paths) == 0 {
		return nil
	}

	table := loadMounts()

	var disks []DiskUsage
	seen := make(map[string]bool, len(paths))

	for _, path := range paths {
		if path == "" {
			continue
		}
		stat, err := statfs(path)
		if err != nil {
			continue
		}

		mountPoint, device := table.lookup(path)

		// Prefer the mount point as the identity so /etc/sing-box and
		// /etc/sing-box/db collapse; fall back to the raw byte counts when
		// /proc/self/mounts is not readable (non-Linux, restricted container).
		key := mountPoint
		if key == "" {
			key = strconv.FormatUint(stat.total, 10) + ":" + strconv.FormatUint(stat.free, 10)
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		disks = append(disks, DiskUsage{
			Path:        path,
			MountPoint:  mountPoint,
			Device:      device,
			TotalBytes:  stat.total,
			UsedBytes:   stat.used,
			FreeBytes:   stat.free,
			UsedPercent: usedPercent(stat.used, stat.free),
		})
	}

	return disks
}

// usedPercent mirrors df: the denominator is the space visible to an
// unprivileged writer, not the raw filesystem size, so a filesystem whose only
// remaining space is root-reserved correctly reads as 100%.
func usedPercent(used, free uint64) float64 {
	denominator := used + free
	if denominator == 0 {
		return 0
	}
	return float64(used) / float64(denominator) * 100
}

// mountTable maps mount points to their backing device, longest mount point
// first so lookup can take the first prefix match.
type mountTable []mountEntry

type mountEntry struct {
	mountPoint string
	device     string
}

func loadMounts() mountTable {
	data, err := os.ReadFile(mountsPath)
	if err != nil {
		return nil
	}
	return parseMounts(string(data))
}

// parseMounts reads the /proc/self/mounts format: "device mountpoint fstype
// options dump pass", with whitespace octal-escaped in the first two fields.
func parseMounts(content string) mountTable {
	var table mountTable

	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		table = append(table, mountEntry{
			mountPoint: unescapeMount(fields[1]),
			device:     unescapeMount(fields[0]),
		})
	}

	// Longest mount point wins: "/overlay" must beat "/" for /overlay/upper.
	// A stable insertion sort keeps later duplicate mounts (the effective one)
	// ahead of earlier ones at equal length.
	for i := 1; i < len(table); i++ {
		for j := i; j > 0 && len(table[j].mountPoint) > len(table[j-1].mountPoint); j-- {
			table[j], table[j-1] = table[j-1], table[j]
		}
	}

	return table
}

// lookup returns the mount point and device for the filesystem containing path.
func (t mountTable) lookup(path string) (mountPoint, device string) {
	for _, entry := range t {
		if underMount(path, entry.mountPoint) {
			return entry.mountPoint, entry.device
		}
	}
	return "", ""
}

// underMount reports whether path sits inside mount, matching on path segments
// so "/overlay2" is never treated as living under "/overlay".
func underMount(path, mount string) bool {
	if mount == "" {
		return false
	}
	if mount == "/" || path == mount {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(mount, "/")+"/")
}

// unescapeMount decodes the octal escapes the kernel writes for characters that
// would otherwise break the space-separated format (\040 space, \011 tab,
// \012 newline, \134 backslash).
func unescapeMount(field string) string {
	if !strings.Contains(field, `\`) {
		return field
	}

	var out strings.Builder
	out.Grow(len(field))

	for i := 0; i < len(field); i++ {
		if field[i] == '\\' && i+3 < len(field) {
			if value, err := strconv.ParseUint(field[i+1:i+4], 8, 8); err == nil {
				out.WriteByte(byte(value))
				i += 3
				continue
			}
		}
		out.WriteByte(field[i])
	}

	return out.String()
}
