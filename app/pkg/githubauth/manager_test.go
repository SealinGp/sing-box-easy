package githubauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeStore is an in-memory TokenStore.
type fakeStore struct {
	mu     sync.Mutex
	token  string
	login  string
	setErr error
}

func (f *fakeStore) SetGitHubToken(token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.token = token
	return nil
}

func (f *fakeStore) GetGitHubToken() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.token
}

func (f *fakeStore) SetGitHubLogin(login string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.login = login
	return nil
}

func (f *fakeStore) GetGitHubLogin() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.login
}

// newTestManager wires a Manager to a stub GitHub. pollBodies are returned from
// the token endpoint in order; the last one repeats.
func newTestManager(t *testing.T, store *fakeStore, pollBodies []string) *Manager {
	t.Helper()

	var mu sync.Mutex
	var polls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/device/code"):
			_, _ = w.Write([]byte(`{
				"device_code":"dc_secret","user_code":"WDJB-MJHT",
				"verification_uri":"https://github.com/login/device",
				"expires_in":900,"interval":0}`))
		case strings.HasSuffix(r.URL.Path, "/access_token"):
			mu.Lock()
			i := polls
			polls++
			mu.Unlock()
			if i >= len(pollBodies) {
				i = len(pollBodies) - 1
			}
			_, _ = w.Write([]byte(pollBodies[i]))
		case strings.HasSuffix(r.URL.Path, "/user"):
			_, _ = w.Write([]byte(`{"login":"octocat"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	m := NewManager("test_client_id", "", store)
	// Redirect the client factory at the stub by overriding the endpoints on
	// every client the manager builds.
	m.newClient = func() (*Client, error) {
		c, err := NewClient("test_client_id", "")
		if err != nil {
			return nil, err
		}
		c.deviceCodeURL = srv.URL + "/device/code"
		c.accessTokenURL = srv.URL + "/access_token"
		c.userAPIURL = srv.URL + "/user"
		return c, nil
	}
	// Poll fast so tests do not sleep through GitHub's 5-second floor.
	m.pollInterval = func(*DeviceCode) time.Duration { return 5 * time.Millisecond }
	return m
}

// waitForStatus polls the session until it leaves pending, or fails the test.
func waitForStatus(t *testing.T, s *Session) SessionView {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if view := s.View(); view.Status != string(StatusPending) {
			return view
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session stayed pending; last view = %+v", s.View())
	return SessionView{}
}

func TestStartLoginUnconfigured(t *testing.T) {
	m := NewManager("", "", &fakeStore{})

	if m.Configured() {
		t.Error("Configured() = true with a blank client ID, want false")
	}
	if _, err := m.StartLogin(context.Background()); err != ErrNotConfigured {
		t.Errorf("StartLogin err = %v, want ErrNotConfigured", err)
	}
	if status := m.Status(); status.Configured {
		t.Error("Status().Configured = true with no client ID, want false")
	}
}

func TestLoginStoresTokenAndLogin(t *testing.T) {
	store := &fakeStore{}
	m := newTestManager(t, store, []string{
		`{"error":"authorization_pending"}`,
		`{"access_token":"gho_abc","token_type":"bearer"}`,
	})

	session, err := m.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}

	if view := session.View(); view.UserCode != "WDJB-MJHT" {
		t.Errorf("UserCode = %q, want WDJB-MJHT", view.UserCode)
	}

	view := waitForStatus(t, session)
	if view.Status != string(StatusAuthorized) {
		t.Fatalf("Status = %q (%s), want authorized", view.Status, view.Error)
	}
	if got := store.GetGitHubToken(); got != "gho_abc" {
		t.Errorf("stored token = %q, want gho_abc", got)
	}
	if got := store.GetGitHubLogin(); got != "octocat" {
		t.Errorf("stored login = %q, want octocat", got)
	}

	status := m.Status()
	if !status.Connected || status.Login != "octocat" {
		t.Errorf("Status = %+v, want connected as octocat", status)
	}
}

// The device code authorizes the token exchange; leaking it to the browser
// would let anyone holding it complete the login.
func TestSessionViewNeverExposesDeviceCode(t *testing.T) {
	m := newTestManager(t, &fakeStore{}, []string{`{"error":"authorization_pending"}`})

	session, err := m.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	t.Cleanup(func() { _ = m.CancelLogin(session.View().ID) })

	if rendered := fmt.Sprintf("%+v", session.View()); strings.Contains(rendered, "dc_secret") {
		t.Errorf("SessionView leaked the device code: %s", rendered)
	}
}

func TestLoginDenied(t *testing.T) {
	store := &fakeStore{}
	m := newTestManager(t, store, []string{`{"error":"access_denied"}`})

	session, err := m.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}

	view := waitForStatus(t, session)
	if view.Status != string(StatusDenied) {
		t.Errorf("Status = %q, want denied", view.Status)
	}
	if store.GetGitHubToken() != "" {
		t.Error("a denied login must not store a token")
	}
}

// A failure to persist must surface, not report a sign-in that did not stick.
func TestLoginFailsWhenStoreRejectsToken(t *testing.T) {
	store := &fakeStore{setErr: fmt.Errorf("disk full")}
	m := newTestManager(t, store, []string{`{"access_token":"gho_abc"}`})

	session, err := m.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}

	view := waitForStatus(t, session)
	if view.Status != string(StatusFailed) {
		t.Errorf("Status = %q, want failed", view.Status)
	}
	if !strings.Contains(view.Error, "disk full") {
		t.Errorf("Error = %q, want it to carry the store failure", view.Error)
	}
}

// Two clicks on "Sign in" must not mint two codes — only one can succeed and
// the user would not know which to type.
func TestStartLoginReusesPendingSession(t *testing.T) {
	m := newTestManager(t, &fakeStore{}, []string{`{"error":"authorization_pending"}`})

	first, err := m.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("first StartLogin: %v", err)
	}
	second, err := m.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("second StartLogin: %v", err)
	}
	t.Cleanup(func() { _ = m.CancelLogin(first.View().ID) })

	if first.View().ID != second.View().ID {
		t.Errorf("StartLogin minted a second session (%s vs %s), want the pending one reused",
			first.View().ID, second.View().ID)
	}
	if pending := m.Status().PendingSession; pending != first.View().ID {
		t.Errorf("Status().PendingSession = %q, want %q", pending, first.View().ID)
	}
}

func TestCancelLogin(t *testing.T) {
	m := newTestManager(t, &fakeStore{}, []string{`{"error":"authorization_pending"}`})

	session, err := m.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	if err := m.CancelLogin(session.View().ID); err != nil {
		t.Fatalf("CancelLogin: %v", err)
	}

	if view := waitForStatus(t, session); view.Status != string(StatusFailed) {
		t.Errorf("Status = %q after cancel, want failed", view.Status)
	}
}

func TestGetSessionUnknownID(t *testing.T) {
	m := NewManager("test_client_id", "", &fakeStore{})

	if _, err := m.GetSession("nope"); err != ErrNoSession {
		t.Errorf("GetSession err = %v, want ErrNoSession", err)
	}
	if err := m.CancelLogin("nope"); err != ErrNoSession {
		t.Errorf("CancelLogin err = %v, want ErrNoSession", err)
	}
}

func TestSignOutClearsCredential(t *testing.T) {
	store := &fakeStore{token: "gho_abc", login: "octocat"}
	m := NewManager("test_client_id", "", store)

	if err := m.SignOut(); err != nil {
		t.Fatalf("SignOut: %v", err)
	}
	if store.GetGitHubToken() != "" || store.GetGitHubLogin() != "" {
		t.Errorf("SignOut left token=%q login=%q, want both cleared",
			store.GetGitHubToken(), store.GetGitHubLogin())
	}
	if status := m.Status(); status.Connected {
		t.Error("Status().Connected = true after SignOut, want false")
	}
}
