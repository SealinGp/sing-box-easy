package appupdate

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxEntrySize bounds a single extracted file (guards against zip bombs).
const maxEntrySize = 512 << 20 // 512 MiB

// extractTarGz extracts a gzipped tarball into targetDir.
//
// It rejects any entry whose resolved path escapes targetDir (path traversal
// via "../" or absolute names) and skips symlinks entirely — the release
// package contains only regular files and directories.
func extractTarGz(archivePath, targetDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to read gzip stream: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	cleanRoot := filepath.Clean(targetDir)
	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read archive entry: %w", err)
		}

		targetPath, err := safeJoin(cleanRoot, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", header.Name, err)
			}

		case tar.TypeReg:
			if err := writeArchiveFile(tr, targetPath, os.FileMode(header.Mode).Perm()); err != nil {
				return fmt.Errorf("failed to extract %s: %w", header.Name, err)
			}

		default:
			// Symlinks, devices, fifos: not part of the release package.
			continue
		}
	}
}

// safeJoin resolves name against root and guarantees the result stays inside
// root. Returns root itself for the "." entry.
func safeJoin(root, name string) (string, error) {
	cleaned := filepath.Clean(strings.TrimPrefix(filepath.ToSlash(name), "./"))
	if cleaned == "." || cleaned == "" {
		return root, nil
	}
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("illegal path in archive: %q", name)
	}

	joined := filepath.Join(root, cleaned)
	if joined != root && !strings.HasPrefix(joined, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("illegal path in archive: %q", name)
	}
	return joined, nil
}

func writeArchiveFile(src io.Reader, targetPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}

	out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, io.LimitReader(src, maxEntrySize+1))
	if err != nil {
		return err
	}
	if written > maxEntrySize {
		return fmt.Errorf("entry exceeds the %d byte limit", int64(maxEntrySize))
	}
	return out.Sync()
}
