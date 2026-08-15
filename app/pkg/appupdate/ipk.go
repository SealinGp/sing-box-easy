package appupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// SelfUpdateMethod names the upgrade path that applies to this install.
type SelfUpdateMethod string

const (
	// SelfUpdateTarball: the panel swaps the binary itself and re-execs.
	SelfUpdateTarball SelfUpdateMethod = "tarball"
	// SelfUpdateOpkg: opkg owns the files. The panel prepares the package but
	// the operator runs the install — see PrepareIpk for why.
	SelfUpdateOpkg SelfUpdateMethod = "opkg"
)

// SelfUpdateInfo tells the UI which upgrade path applies, so it can offer the
// right action instead of a button that cannot work.
type SelfUpdateInfo struct {
	Method SelfUpdateMethod `json:"method"`
	// Automatic reports whether the panel can complete the update on its own.
	// False for opkg installs: opkg would stop this very service mid-install.
	Automatic bool `json:"automatic"`
	// Architecture is the opkg arch of the installed package, naming the ipk
	// variant to fetch. Empty unless Method is opkg.
	Architecture string `json:"architecture"`
	// FeedProvides reports whether a configured opkg feed offers this package,
	// i.e. whether `opkg upgrade` can work. FeedKnown is false when the feed
	// cache is empty (tmpfs, wiped each boot until `opkg update` runs), in
	// which case FeedProvides carries no information.
	FeedProvides bool `json:"feed_provides"`
	FeedKnown    bool `json:"feed_known"`
}

// DetectSelfUpdate resolves how this install can be upgraded.
func DetectSelfUpdate() SelfUpdateInfo {
	install := InspectOpkg()
	if !install.Managed {
		return SelfUpdateInfo{Method: SelfUpdateTarball, Automatic: true}
	}

	provides, known := FeedProvidesSelf()
	return SelfUpdateInfo{
		Method:       SelfUpdateOpkg,
		Automatic:    false,
		Architecture: install.Architecture,
		FeedProvides: provides,
		FeedKnown:    known,
	}
}

// IpkPlan is a downloaded, checksum-verified package plus the exact commands
// that install it. The panel does the tedious parts (resolving the arch,
// fetching the right asset, verifying it) and leaves the privileged,
// irreversible step to the operator.
type IpkPlan struct {
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	Path         string `json:"path"`
	SHA256       string `json:"sha256"`
	Verified     bool   `json:"verified"`
	SizeBytes    int64  `json:"size_bytes"`
	// Command installs the downloaded file. Always usable.
	Command string `json:"command"`
	// FeedCommand upgrades from a configured feed. Empty unless a feed is
	// known to provide the package.
	FeedCommand string `json:"feed_command"`
}

// ipkDownloadDir is where prepared packages land. /tmp is tmpfs on OpenWrt, so
// an abandoned download costs RAM until reboot rather than flash wear.
const ipkDownloadDir = "/tmp"

// ipkInstallCommand builds the opkg command line for a prepared package.
// A downgrade needs an explicit flag, otherwise opkg refuses.
func ipkInstallCommand(path string, downgrade bool) string {
	if downgrade {
		return "opkg install --force-downgrade " + path
	}
	return "opkg install " + path
}

// feedUpgradeCommand is what to run when a configured feed carries the package.
func feedUpgradeCommand() string {
	return "opkg update && opkg upgrade " + opkgPackageName
}

// StartIpkPrepare downloads and verifies the ipk for the given tag, without
// installing it. An empty tag means "latest".
//
// The panel deliberately stops short of installing. Our ipk's prerm runs
// `/etc/init.d/sing-box-easy stop`, so an opkg install driven from inside this
// process would kill the process group mid-transaction and could leave the
// router with no panel binary — recoverable only over SSH. Handing back a
// verified file and an exact command keeps that step under the operator's
// control.
//
// Progress is reported through the same Task machinery as a tarball update, so
// the frontend reuses its existing polling. The task finishes as `completed`
// with a Plan attached and never transitions to `restarting`.
func (u *Updater) StartIpkPrepare(tag string) (*Task, error) {
	info := DetectSelfUpdate()
	if info.Method != SelfUpdateOpkg {
		return nil, fmt.Errorf("this instance was not installed via opkg; use the standard update instead")
	}
	if info.Architecture == "" {
		return nil, fmt.Errorf("could not determine the opkg architecture of the installed package; " +
			"check `opkg status " + opkgPackageName + "`")
	}

	tag = strings.TrimSpace(tag)
	if tag != "" && !tagPattern.MatchString(tag) {
		return nil, fmt.Errorf("invalid version tag: %q (expected something like v1.2.3)", tag)
	}

	client, err := u.releaseClient()
	if err != nil {
		return nil, err
	}

	var target *Release
	if tag == "" {
		target, err = client.LatestRelease(true)
	} else {
		target, err = client.FindRelease(tag)
	}
	if err != nil {
		return nil, err
	}

	u.mu.Lock()
	if u.running != nil && !u.running.isTerminal() {
		id := u.running.id
		u.mu.Unlock()
		return nil, fmt.Errorf("an update is already in progress (task %s)", id)
	}

	task := &Task{
		id:          fmt.Sprintf("ipk_%d", time.Now().UnixNano()),
		status:      StatusRunning,
		message:     "Preparing package...",
		fromVersion: Current(),
		toVersion:   target.TagName,
	}
	u.tasks[task.id] = task
	u.running = task
	u.mu.Unlock()

	logger.Info("Preparing ipk for manual install",
		zap.String("task_id", task.id),
		zap.String("tag", target.TagName),
		zap.String("arch", info.Architecture),
	)

	go func() {
		plan, err := u.prepareIpk(task, target.TagName, info)
		if err != nil {
			logger.Error("ipk prepare failed",
				zap.String("task_id", task.ID()),
				zap.String("tag", target.TagName),
				zap.Error(err))
			task.fail(err)
			return
		}
		task.succeedWithPlan(plan)
		logger.Info("ipk ready for manual install",
			zap.String("task_id", task.ID()),
			zap.String("path", plan.Path),
			zap.Bool("verified", plan.Verified))
	}()

	return task, nil
}

// prepareIpk downloads the architecture-matched ipk and verifies its checksum.
func (u *Updater) prepareIpk(task *Task, tag string, info SelfUpdateInfo) (*IpkPlan, error) {
	assetName := IpkAssetName(tag, info.Architecture)
	destPath := filepath.Join(ipkDownloadDir, assetName)

	// A partial file from an interrupted run would otherwise be appended to or
	// silently reused.
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clear the previous download: %w", err)
	}

	task.setProgress(5, "Downloading "+assetName+"...")
	sum, err := u.download(task, IpkAssetURL(tag, info.Architecture), destPath)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat the downloaded package: %w", err)
	}

	task.setProgress(80, "Verifying checksum...")
	verified, err := u.verifyIpkChecksum(tag, info.Architecture, sum)
	if err != nil {
		// A mismatch means the file on disk is untrustworthy — do not leave it
		// lying around for someone to install by hand.
		if removeErr := os.Remove(destPath); removeErr != nil {
			logger.Warn("Failed to remove the mismatched download",
				zap.String("path", destPath), zap.Error(removeErr))
		}
		return nil, err
	}

	plan := &IpkPlan{
		Version:      tag,
		Architecture: info.Architecture,
		Path:         destPath,
		SHA256:       sum,
		Verified:     verified,
		SizeBytes:    stat.Size(),
		Command:      ipkInstallCommand(destPath, CompareVersions(tag, Current()) < 0),
	}
	if info.FeedProvides {
		plan.FeedCommand = feedUpgradeCommand()
	}

	task.setProgress(100, "Package ready to install")
	return plan, nil
}

// verifyIpkChecksum compares the downloaded sum against the published sidecar.
// Reports whether verification actually happened: a missing sidecar is
// tolerated (older releases may not publish one) but must not be presented to
// the operator as a verified download.
//
// The sidecar records the build-time path ("ipk-out/<name>.ipk"), so only the
// first field — the digest — is compared.
func (u *Updater) verifyIpkChecksum(tag, arch, actualSum string) (bool, error) {
	body, err := u.fetchText(IpkChecksumURL(tag, arch))
	if err != nil {
		logger.Warn("ipk checksum sidecar unreachable; skipping verification",
			zap.String("tag", tag), zap.Error(err))
		return false, nil
	}

	fields := strings.Fields(body)
	if len(fields) == 0 {
		logger.Warn("ipk checksum sidecar is empty; skipping verification",
			zap.String("tag", tag))
		return false, nil
	}

	if expected := strings.ToLower(fields[0]); expected != actualSum {
		return false, fmt.Errorf("checksum mismatch for %s: expected %s, got %s",
			IpkAssetName(tag, arch), expected, actualSum)
	}
	return true, nil
}

// succeedWithPlan marks a prepare task complete and attaches its result.
func (t *Task) succeedWithPlan(plan *IpkPlan) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = StatusCompleted
	t.progress = 100
	t.plan = plan
	t.message = "Package downloaded and ready to install"
}
