package appupdate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// defaultBinaryName is the executable name shipped inside the release tarball.
const defaultBinaryName = "sing-box-easy"

// installRelease swaps the freshly extracted binary and frontend bundle into
// place.
//
// The running binary cannot be overwritten in place (ETXTBSY on Linux), so the
// current one is renamed aside to "<name>.old" first — that also leaves a
// manual rollback artifact behind. Both swaps are same-filesystem renames, and
// a failure after the binary swap restores the previous binary before
// returning, so the install directory is never left without an executable.
func installRelease(binaryDir, extractDir string) error {
	binaryName := currentBinaryName()

	newBinary, err := locateNewBinary(extractDir, binaryName)
	if err != nil {
		return err
	}

	// The frontend is resolved from the process working directory ("./dist"),
	// which is the install directory under both systemd and the nohup fallback.
	distDir, err := os.Getwd()
	if err != nil {
		logger.Warn("Failed to resolve the working directory; installing dist next to the binary", zap.Error(err))
		distDir = binaryDir
	}

	newDist := filepath.Join(extractDir, "dist")
	if err := requireDir(newDist); err != nil {
		return fmt.Errorf("release package is missing the frontend bundle: %w", err)
	}
	if _, err := os.Stat(filepath.Join(newDist, "index.html")); err != nil {
		return fmt.Errorf("release package frontend is incomplete (dist/index.html missing): %w", err)
	}

	// ── Swap the binary ────────────────────────────────────────────────────
	livePath := filepath.Join(binaryDir, binaryName)
	backupPath := livePath + ".old"

	_ = os.Remove(backupPath)
	binaryExisted := true
	if err := os.Rename(livePath, backupPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to move the current binary aside: %w", err)
		}
		binaryExisted = false
	}

	restoreBinary := func() {
		if !binaryExisted {
			return
		}
		if err := os.Rename(backupPath, livePath); err != nil {
			logger.Error("Failed to restore the previous binary after a failed update",
				zap.String("backup", backupPath), zap.Error(err))
		}
	}

	if err := moveFile(newBinary, livePath); err != nil {
		restoreBinary()
		return fmt.Errorf("failed to install the new binary: %w", err)
	}
	if err := os.Chmod(livePath, 0o755); err != nil {
		restoreBinary()
		return fmt.Errorf("failed to mark the new binary executable: %w", err)
	}

	// ── Swap the frontend bundle ───────────────────────────────────────────
	if err := swapDir(newDist, filepath.Join(distDir, "dist")); err != nil {
		restoreBinary()
		return fmt.Errorf("failed to install the frontend bundle: %w", err)
	}

	// Best-effort extras: never fail the update over these.
	copyIfPresent(filepath.Join(extractDir, "app.yml.example"), filepath.Join(binaryDir, "app.yml.example"))

	logger.Info("Release installed",
		zap.String("binary", livePath),
		zap.String("dist", filepath.Join(distDir, "dist")),
		zap.String("backup", backupPath),
	)
	return nil
}

// currentBinaryName returns the file name the binary is installed under.
//
// It resolves from the startup-captured executable path with any ".old" backup
// suffixes stripped — NOT from a live os.Executable() call. After the swap
// below, os.Executable() reports the displaced backup, so deriving the name
// from it made each successive update target "<name>.old", "<name>.old.old", …
// while the real install path was never written again.
func currentBinaryName() string {
	return BinaryName()
}

// locateNewBinary finds the executable inside the extracted package. The
// tarball is flat, so it is either named after the running binary or carries
// the packaged default name.
func locateNewBinary(extractDir, binaryName string) (string, error) {
	candidates := []string{binaryName}
	if binaryName != defaultBinaryName {
		candidates = append(candidates, defaultBinaryName)
	}

	for _, name := range candidates {
		path := filepath.Join(extractDir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if info.Size() == 0 {
			return "", fmt.Errorf("release package contains an empty binary %q", name)
		}
		return path, nil
	}

	return "", fmt.Errorf("release package does not contain a %q binary", binaryName)
}

func requireDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

// swapDir replaces dst with src, keeping the previous contents until the new
// directory is in place so a mid-swap failure can be rolled back.
func swapDir(src, dst string) error {
	backup := dst + ".old"
	_ = os.RemoveAll(backup)

	hadExisting := true
	if err := os.Rename(dst, backup); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to move %s aside: %w", dst, err)
		}
		hadExisting = false
	}

	if err := os.Rename(src, dst); err != nil {
		if hadExisting {
			if restoreErr := os.Rename(backup, dst); restoreErr != nil {
				logger.Error("Failed to restore the previous directory",
					zap.String("backup", backup), zap.Error(restoreErr))
			}
		}
		return fmt.Errorf("failed to move the new bundle into place: %w", err)
	}

	if hadExisting {
		if err := os.RemoveAll(backup); err != nil {
			logger.Warn("Failed to remove the previous bundle backup",
				zap.String("backup", backup), zap.Error(err))
		}
	}
	return nil
}

// moveFile renames src to dst, falling back to a copy when the two live on
// different filesystems.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// copyIfPresent copies src to dst when src exists, logging (not failing) on error.
func copyIfPresent(src, dst string) {
	if _, err := os.Stat(src); err != nil {
		return
	}
	if err := moveFile(src, dst); err != nil {
		logger.Warn("Failed to copy a packaged file",
			zap.String("src", src), zap.String("dst", dst), zap.Error(err))
	}
}
