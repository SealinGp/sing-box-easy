package githubauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// ErrNotConfigured is returned when no OAuth client ID has been configured, so
// sign-in cannot be offered at all.
var ErrNotConfigured = errors.New("github sign-in is not configured: set github.oauth_client_id in app.yml")

// ErrNoSession is returned when a login session ID is unknown or has been
// evicted.
var ErrNoSession = errors.New("login session not found or expired")

// sessionRetention keeps a finished session queryable long enough for the
// frontend's final poll to observe the outcome.
const sessionRetention = 2 * time.Minute

// TokenStore persists the credential obtained from GitHub. The settings
// manager satisfies it; keeping it an interface stops this package from
// depending on the database layer.
type TokenStore interface {
	SetGitHubToken(token string) error
	GetGitHubToken() string
	SetGitHubLogin(login string) error
	GetGitHubLogin() string
}

// Session is one in-flight device-flow login.
type Session struct {
	mu sync.RWMutex

	id              string
	userCode        string
	verificationURI string
	expiresAt       time.Time

	status PollStatus
	login  string
	errMsg string

	// finishedAt marks when the session reached a terminal state, so it can
	// be evicted after sessionRetention.
	finishedAt time.Time

	cancel context.CancelFunc
}

// SessionView is an immutable snapshot safe to serialize to the browser.
//
// It deliberately omits the device code: that value is the bearer credential
// for the pending authorization, and it never needs to leave the server.
type SessionView struct {
	ID              string `json:"id"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Status          string `json:"status"`
	Login           string `json:"login"`
	Error           string `json:"error"`
	ExpiresAt       string `json:"expires_at"`
	ExpiresInSecs   int    `json:"expires_in"`
}

// View returns a consistent snapshot of the session.
func (s *Session) View() SessionView {
	s.mu.RLock()
	defer s.mu.RUnlock()

	view := SessionView{
		ID:              s.id,
		UserCode:        s.userCode,
		VerificationURI: s.verificationURI,
		Status:          string(s.status),
		Login:           s.login,
		Error:           s.errMsg,
	}
	if !s.expiresAt.IsZero() {
		view.ExpiresAt = s.expiresAt.UTC().Format(time.RFC3339)
		if remaining := int(time.Until(s.expiresAt).Seconds()); remaining > 0 {
			view.ExpiresInSecs = remaining
		}
	}
	return view
}

func (s *Session) finish(status PollStatus, login string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = status
	s.login = login
	s.finishedAt = time.Now()
	if err != nil {
		s.errMsg = err.Error()
	}
}

func (s *Session) isFinished() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status != StatusPending
}

// Manager owns device-flow login sessions and the stored credential.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	active   *Session
	seq      int

	clientID ClientIDFunc
	proxy    string
	store    TokenStore

	// now, newClient and pollInterval are swappable in tests. pollInterval
	// exists so tests need not sleep through GitHub's 5-second floor.
	now          func() time.Time
	newClient    func() (*Client, error)
	pollInterval func(*DeviceCode) time.Duration
}

// ClientIDFunc resolves the OAuth client ID at call time.
//
// It is a function rather than a plain string so an ID saved from the
// dashboard takes effect immediately: the manager is built once at startup,
// long before the operator pastes a client ID into Settings, and rebuilding it
// would drop any in-flight login session.
type ClientIDFunc func() string

// NewManager creates a sign-in manager. clientID may be nil or resolve to "",
// in which case every operation reports ErrNotConfigured and the UI hides
// sign-in until one is configured.
func NewManager(clientID ClientIDFunc, proxy string, store TokenStore) *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
		clientID: clientID,
		proxy:    proxy,
		store:    store,
		now:      time.Now,
	}
	m.newClient = func() (*Client, error) { return NewClient(m.resolveClientID(), m.proxy) }
	m.pollInterval = func(c *DeviceCode) time.Duration { return c.PollInterval() }
	return m
}

// resolveClientID reads the current client ID, tolerating a nil resolver.
func (m *Manager) resolveClientID() string {
	if m.clientID == nil {
		return ""
	}
	return strings.TrimSpace(m.clientID())
}

// Configured reports whether an OAuth client ID is available.
func (m *Manager) Configured() bool {
	return m.resolveClientID() != ""
}

// Status describes the current GitHub connection.
type Status struct {
	// Configured is false when the deployment has no OAuth client ID, so the
	// UI can explain that instead of offering a button that cannot work.
	Configured bool `json:"configured"`

	Connected bool   `json:"connected"`
	Login     string `json:"login"`

	// PendingSession is the ID of a login already in flight, letting a
	// reloaded page rejoin it instead of starting a duplicate.
	PendingSession string `json:"pending_session"`
}

// Status returns the current connection state.
func (m *Manager) Status() Status {
	status := Status{
		Configured: m.Configured(),
		Connected:  m.store.GetGitHubToken() != "",
		Login:      m.store.GetGitHubLogin(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && !m.active.isFinished() {
		status.PendingSession = m.active.id
	}
	return status
}

// StartLogin begins a device-flow login and returns the session the user must
// approve. An already-running login is returned as-is rather than duplicated —
// two concurrent codes would be confusing and only one can succeed usefully.
func (m *Manager) StartLogin(ctx context.Context) (*Session, error) {
	if !m.Configured() {
		return nil, ErrNotConfigured
	}

	m.mu.Lock()
	if m.active != nil && !m.active.isFinished() {
		existing := m.active
		m.mu.Unlock()
		return existing, nil
	}
	m.mu.Unlock()

	client, err := m.newClient()
	if err != nil {
		return nil, err
	}

	code, err := client.RequestDeviceCode(ctx)
	if err != nil {
		return nil, err
	}

	// The poll loop outlives this HTTP request, so it gets its own context
	// rather than inheriting one that is cancelled when the response is sent.
	pollCtx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.seq++
	session := &Session{
		id:              fmt.Sprintf("ghlogin_%d_%d", m.now().UnixNano(), m.seq),
		userCode:        code.UserCode,
		verificationURI: code.VerificationURI,
		expiresAt:       code.Expiry(m.now()),
		status:          StatusPending,
		cancel:          cancel,
	}
	m.sessions[session.id] = session
	m.active = session
	m.evictExpiredLocked()
	m.mu.Unlock()

	logger.Info("GitHub device login started",
		zap.String("session_id", session.id),
		zap.String("verification_uri", code.VerificationURI),
	)

	go m.poll(pollCtx, client, session, code)

	return session, nil
}

// poll drives the token exchange until the session resolves or expires.
func (m *Manager) poll(ctx context.Context, client *Client, session *Session, code *DeviceCode) {
	defer session.cancel()

	interval := m.pollInterval(code)
	deadline := code.Expiry(m.now())

	for {
		select {
		case <-ctx.Done():
			session.finish(StatusFailed, "", fmt.Errorf("login was cancelled"))
			return
		case <-time.After(interval):
		}

		if m.now().After(deadline) {
			session.finish(StatusExpired, "", fmt.Errorf("the login code expired before it was approved"))
			return
		}

		result := client.PollToken(ctx, code.DeviceCode)

		if result.SlowDown {
			// GitHub mandates widening the interval, not just retrying.
			interval += SlowDownPenalty()
		}

		switch result.Status {
		case StatusPending:
			continue
		case StatusAuthorized:
			m.completeLogin(ctx, client, session, result.Token)
			return
		default:
			logger.Warn("GitHub device login did not complete",
				zap.String("session_id", session.id),
				zap.String("status", string(result.Status)),
				zap.Error(result.Err),
			)
			session.finish(result.Status, "", result.Err)
			return
		}
	}
}

// completeLogin persists the token and records the account name.
func (m *Manager) completeLogin(ctx context.Context, client *Client, session *Session, token string) {
	if err := m.store.SetGitHubToken(token); err != nil {
		session.finish(StatusFailed, "", fmt.Errorf("failed to save the GitHub token: %w", err))
		return
	}

	// The account name is cosmetic. Failing to read it must not undo a
	// successful sign-in, so the error is logged and swallowed here.
	login, err := client.AccountLogin(ctx, token)
	if err != nil {
		logger.Warn("Signed in to GitHub but could not read the account name", zap.Error(err))
	}
	if login != "" {
		if err := m.store.SetGitHubLogin(login); err != nil {
			logger.Warn("Failed to persist the GitHub account name", zap.Error(err))
		}
	}

	logger.Info("GitHub sign-in complete",
		zap.String("session_id", session.id),
		zap.String("login", login),
	)
	session.finish(StatusAuthorized, login, nil)
}

// GetSession returns a session by ID.
func (m *Manager) GetSession(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, ErrNoSession
	}
	return session, nil
}

// CancelLogin aborts an in-flight login.
func (m *Manager) CancelLogin(id string) error {
	session, err := m.GetSession(id)
	if err != nil {
		return err
	}
	session.cancel()
	return nil
}

// SignOut clears the stored credential, dropping back to anonymous access.
func (m *Manager) SignOut() error {
	if err := m.store.SetGitHubToken(""); err != nil {
		return fmt.Errorf("failed to clear the GitHub token: %w", err)
	}
	if err := m.store.SetGitHubLogin(""); err != nil {
		return fmt.Errorf("failed to clear the GitHub account name: %w", err)
	}
	logger.Info("Signed out of GitHub")
	return nil
}

// evictExpiredLocked drops sessions that finished long enough ago that no
// client could still be polling them. Callers must hold m.mu.
func (m *Manager) evictExpiredLocked() {
	cutoff := m.now().Add(-sessionRetention)
	for id, s := range m.sessions {
		s.mu.RLock()
		finished := s.status != StatusPending && !s.finishedAt.IsZero() && s.finishedAt.Before(cutoff)
		s.mu.RUnlock()

		if finished {
			delete(m.sessions, id)
		}
	}
}
