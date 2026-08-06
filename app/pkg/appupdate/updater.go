package appupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// Task status values.
const (
	StatusRunning    = "running"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusRestarting = "restarting"
)

// tagPattern restricts a user-supplied release tag to a safe shape before it
// is interpolated into a download URL or used as a filesystem-adjacent value.
// Allows: v1.2.3, 1.2.3, v1.2.3-rc.1
var tagPattern = regexp.MustCompile(`^v?\d+(\.\d+){0,2}(-[0-9A-Za-z.]+)?$`)

// restartDelay gives the HTTP layer time to flush the final task-status
// response to the browser before the process re-executes itself.
const restartDelay = 2 * time.Second

// downloadTimeout bounds the release asset download.
const downloadTimeout = 15 * time.Minute

// Task tracks a single self-update run.
type Task struct {
	mu sync.RWMutex

	id          string
	status      string
	message     string
	errMsg      string
	progress    int
	fromVersion string
	toVersion   string
}

// TaskSnapshot is an immutable view of a Task, safe to serialize.
type TaskSnapshot struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Error       string `json:"error"`
	Progress    int    `json:"progress"`
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
}

// Snapshot returns a consistent copy of the task's current state.
func (t *Task) Snapshot() TaskSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return TaskSnapshot{
		ID:          t.id,
		Status:      t.status,
		Message:     t.message,
		Error:       t.errMsg,
		Progress:    t.progress,
		FromVersion: t.fromVersion,
		ToVersion:   t.toVersion,
	}
}

// ID returns the task identifier.
func (t *Task) ID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.id
}

func (t *Task) setProgress(progress int, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.progress = progress
	t.message = message
}

func (t *Task) fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = StatusFailed
	t.errMsg = err.Error()
	t.message = "Update failed"
}

func (t *Task) succeed(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = StatusCompleted
	t.progress = 100
	t.message = message
}

func (t *Task) markRestarting() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = StatusRestarting
	t.message = "Restarting sing-box-easy..."
}

func (t *Task) isTerminal() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status == StatusCompleted || t.status == StatusFailed || t.status == StatusRestarting
}

// Updater downloads a sing-box-easy release, swaps it in on disk and restarts
// the process. Only one update may run at a time.
type Updater struct {
	mu      sync.RWMutex
	tasks   map[string]*Task
	running *Task

	// proxy is an optional HTTP proxy used for GitHub metadata + downloads.
	proxy string

	// token resolves the GitHub access token at call time; may be nil.
	token TokenFunc

	// client is built once and reused: the release cache lives on it, so a
	// per-call client would silently defeat caching and hammer GitHub's
	// unauthenticated rate limit.
	clientOnce sync.Once
	client     *ReleaseClient
	clientErr  error

	// restart is swappable for tests; defaults to restartProcess.
	restart func() error
}

// NewUpdater creates an updater. proxy may be empty and token may be nil
// (GitHub is then called anonymously, at 60 requests/hour per IP).
func NewUpdater(proxy string, token TokenFunc) *Updater {
	return &Updater{
		tasks:   make(map[string]*Task),
		proxy:   proxy,
		token:   token,
		restart: restartProcess,
	}
}

// RateLimit exposes the last observed GitHub rate-limit state, so the UI can
// explain a throttled check instead of just failing.
func (u *Updater) RateLimit() RateLimit {
	client, err := u.releaseClient()
	if err != nil {
		return RateLimit{}
	}
	return client.RateLimitState()
}

// Releases returns the published releases newest-first.
func (u *Updater) Releases(force bool) ([]Release, error) {
	client, err := u.releaseClient()
	if err != nil {
		return nil, err
	}
	return client.ListReleases(force)
}

// Status describes the running version relative to the newest release.
type Status struct {
	CurrentVersion string  `json:"current_version"`
	CurrentKnown   bool    `json:"current_known"`
	LatestVersion  string  `json:"latest_version"`
	LatestURL      string  `json:"latest_url"`
	LatestNotes    string  `json:"latest_notes"`
	PublishedAt    string  `json:"published_at"`
	HasUpdate      bool    `json:"has_update"`
	Prerelease     bool    `json:"prerelease"`
	AssetName      string  `json:"asset_name"`
	Updating       bool    `json:"updating"`
	CheckError     string  `json:"check_error"`
	Task           *string `json:"running_task_id"`
}

// CheckStatus resolves the current version and the newest available release.
// A GitHub failure is reported inside Status.CheckError rather than as an
// error, so the UI can still render the current version.
func (u *Updater) CheckStatus(force bool) *Status {
	status := &Status{
		CurrentVersion: Current(),
		CurrentKnown:   IsKnown(),
		AssetName:      AssetName(),
	}

	if task := u.RunningTask(); task != nil {
		id := task.ID()
		status.Updating = true
		status.Task = &id
	}

	client, err := u.releaseClient()
	if err != nil {
		status.CheckError = err.Error()
		return status
	}

	latest, err := client.LatestRelease(force)
	if err != nil {
		status.CheckError = err.Error()
		return status
	}

	status.LatestVersion = latest.TagName
	status.LatestURL = latest.HTMLURL
	status.LatestNotes = latest.Body
	status.Prerelease = latest.Prerelease
	if !latest.PublishedAt.IsZero() {
		status.PublishedAt = latest.PublishedAt.UTC().Format(time.RFC3339)
	}
	status.HasUpdate = IsNewer(latest.TagName, status.CurrentVersion)

	return status
}

// GetTask returns a task by ID.
func (u *Updater) GetTask(id string) (*Task, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	task, ok := u.tasks[id]
	if !ok {
		return nil, fmt.Errorf("update task %q not found", id)
	}
	return task, nil
}

// RunningTask returns the in-flight task, or nil when idle.
func (u *Updater) RunningTask() *Task {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if u.running == nil || u.running.isTerminal() {
		return nil
	}
	return u.running
}

// StartUpdate begins an update to the given tag. An empty tag means "latest".
// Returns an error when another update is already running, when the tag is
// malformed, or when the target release does not exist.
func (u *Updater) StartUpdate(tag string) (*Task, error) {
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
		id:          fmt.Sprintf("update_%d", time.Now().UnixNano()),
		status:      StatusRunning,
		message:     "Preparing update...",
		fromVersion: Current(),
		toVersion:   target.TagName,
	}
	u.tasks[task.id] = task
	u.running = task
	u.mu.Unlock()

	logger.Info("Starting self-update",
		zap.String("task_id", task.id),
		zap.String("from", task.fromVersion),
		zap.String("to", task.toVersion),
	)

	go u.run(task, target.TagName)

	return task, nil
}

func (u *Updater) run(task *Task, tag string) {
	if err := u.performUpdate(task, tag); err != nil {
		logger.Error("Self-update failed",
			zap.String("task_id", task.ID()),
			zap.String("tag", tag),
			zap.Error(err),
		)
		task.fail(err)
		return
	}

	task.succeed(fmt.Sprintf("Updated to %s. Restarting...", tag))
	logger.Info("Self-update installed, scheduling restart",
		zap.String("task_id", task.ID()),
		zap.String("tag", tag),
	)

	// Restart out-of-band so the final task-status poll still gets a response.
	go func() {
		time.Sleep(restartDelay)
		task.markRestarting()
		if err := u.restart(); err != nil {
			logger.Error("Failed to restart after update; a manual restart is required", zap.Error(err))
		}
	}()
}

// performUpdate downloads, verifies and installs the release identified by tag.
func (u *Updater) performUpdate(task *Task, tag string) error {
	installDir, err := InstallDir()
	if err != nil {
		return fmt.Errorf("failed to locate the install directory: %w", err)
	}

	// Staging lives inside the install dir so the final moves are same-filesystem
	// renames (atomic, and they cannot fail with EXDEV).
	stagingDir := filepath.Join(installDir, ".update-staging")
	if err := os.RemoveAll(stagingDir); err != nil {
		return fmt.Errorf("failed to clear the staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	task.setProgress(5, "Downloading "+AssetName()+"...")
	archivePath := filepath.Join(stagingDir, AssetName())
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return fmt.Errorf("failed to create the staging directory: %w", err)
	}

	sum, err := u.download(task, AssetURL(tag), archivePath)
	if err != nil {
		return err
	}

	task.setProgress(60, "Verifying checksum...")
	if err := u.verifyChecksum(tag, sum); err != nil {
		return err
	}

	task.setProgress(70, "Extracting package...")
	extractDir := filepath.Join(stagingDir, "extract")
	if err := extractTarGz(archivePath, extractDir); err != nil {
		return err
	}

	task.setProgress(85, "Installing files...")
	if err := installRelease(installDir, extractDir); err != nil {
		return err
	}

	if err := writeVersionFile(installDir, tag); err != nil {
		// Non-fatal: the binary may still carry a build-time stamp.
		logger.Warn("Failed to write the version file", zap.Error(err))
	}

	task.setProgress(95, "Installed. Preparing to restart...")
	return nil
}

// download fetches url into destPath and returns the sha256 of the content.
func (u *Updater) download(task *Task, downloadURL, destPath string) (string, error) {
	client := &http.Client{Timeout: downloadTimeout}
	if u.proxy != "" {
		proxyURL, err := url.Parse(u.proxy)
		if err != nil {
			return "", fmt.Errorf("invalid proxy URL: %w", err)
		}
		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build the download request: %w", err)
	}
	req.Header.Set("User-Agent", "sing-box-easy")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download the release asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with HTTP %d for %s", resp.StatusCode, downloadURL)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create the download file: %w", err)
	}
	defer out.Close()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(out, hasher), &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
		onTick: func(read, total int64) {
			// Map the download onto the 5–60% band of the overall task.
			pct := 5
			label := fmt.Sprintf("Downloading... (%.1f MB)", float64(read)/(1024*1024))
			if total > 0 {
				pct = 5 + int(float64(read)/float64(total)*55)
				label = fmt.Sprintf("Downloading... %.1f/%.1f MB",
					float64(read)/(1024*1024), float64(total)/(1024*1024))
			}
			task.setProgress(pct, label)
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to write the release asset: %w", err)
	}
	if written == 0 {
		return "", fmt.Errorf("downloaded an empty release asset from %s", downloadURL)
	}
	if err := out.Sync(); err != nil {
		return "", fmt.Errorf("failed to flush the release asset: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// verifyChecksum compares the downloaded sum against the published .sha256
// sidecar. A missing sidecar is tolerated (older releases may not have one);
// a mismatching one is fatal.
func (u *Updater) verifyChecksum(tag, actualSum string) error {
	client := &http.Client{Timeout: apiTimeout}
	if u.proxy != "" {
		proxyURL, err := url.Parse(u.proxy)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	resp, err := client.Get(ChecksumURL(tag))
	if err != nil {
		logger.Warn("Checksum sidecar unreachable; skipping verification",
			zap.String("tag", tag), zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("Checksum sidecar not published; skipping verification",
			zap.String("tag", tag), zap.Int("status", resp.StatusCode))
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("failed to read the checksum file: %w", err)
	}

	// Format: "<sha256>  <filename>"
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return fmt.Errorf("checksum file for %s is empty", tag)
	}
	expected := strings.ToLower(fields[0])

	if expected != actualSum {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", tag, expected, actualSum)
	}
	return nil
}

// releaseClient returns the shared, cache-bearing GitHub client.
func (u *Updater) releaseClient() (*ReleaseClient, error) {
	u.clientOnce.Do(func() {
		u.client, u.clientErr = NewReleaseClient(u.proxy, u.token)
	})
	return u.client, u.clientErr
}

// progressReader wraps a reader and reports throughput at most ~4x/second.
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	lastTick time.Time
	onTick   func(read, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	p.read += int64(n)

	if p.onTick != nil && (time.Since(p.lastTick) > 250*time.Millisecond || err == io.EOF) {
		p.lastTick = time.Now()
		p.onTick(p.read, p.total)
	}
	return n, err
}
