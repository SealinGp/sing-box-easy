package githubauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient points a Client at a local server standing in for GitHub.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client, err := NewClient("test_client_id", "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.deviceCodeURL = srv.URL + "/device/code"
	client.accessTokenURL = srv.URL + "/access_token"
	client.userAPIURL = srv.URL + "/user"

	return client
}

func TestNewClientRequiresClientID(t *testing.T) {
	if _, err := NewClient("   ", ""); err != ErrNotConfigured {
		t.Errorf("NewClient with a blank ID: err = %v, want ErrNotConfigured", err)
	}
}

func TestRequestDeviceCode(t *testing.T) {
	var gotAccept, gotContentType, gotBody string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotBody = r.Form.Encode()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"device_code":"dc_secret","user_code":"WDJB-MJHT",
			"verification_uri":"https://github.com/login/device",
			"expires_in":900,"interval":5}`))
	})

	code, err := client.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatalf("RequestDeviceCode: %v", err)
	}

	// Without Accept: application/json GitHub replies url-encoded, which would
	// silently fail to unmarshal.
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want form-urlencoded", gotContentType)
	}
	// Device flow must never send a client secret.
	if strings.Contains(gotBody, "client_secret") {
		t.Errorf("request body %q must not contain client_secret", gotBody)
	}
	if !strings.Contains(gotBody, "client_id=test_client_id") {
		t.Errorf("request body = %q, want it to carry the client_id", gotBody)
	}

	if code.UserCode != "WDJB-MJHT" {
		t.Errorf("UserCode = %q, want WDJB-MJHT", code.UserCode)
	}
	if code.DeviceCode != "dc_secret" {
		t.Errorf("DeviceCode = %q, want dc_secret", code.DeviceCode)
	}
	if code.PollInterval() != 5*time.Second {
		t.Errorf("PollInterval = %v, want 5s", code.PollInterval())
	}
}

// The device_flow_disabled error is the single most likely setup mistake, so
// the message must name the checkbox rather than echo GitHub's terse code.
func TestRequestDeviceCodeExplainsDisabledFlow(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"device_flow_disabled","error_description":"nope"}`))
	})

	_, err := client.RequestDeviceCode(context.Background())
	if err == nil {
		t.Fatal("RequestDeviceCode succeeded on device_flow_disabled, want an error")
	}
	if !strings.Contains(err.Error(), "Enable Device Flow") {
		t.Errorf("error = %v, want it to name the 'Enable Device Flow' setting", err)
	}
}

func TestRequestDeviceCodeExplainsBadClientID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"incorrect_client_credentials"}`))
	})

	_, err := client.RequestDeviceCode(context.Background())
	if err == nil {
		t.Fatal("RequestDeviceCode succeeded on incorrect_client_credentials, want an error")
	}
	if !strings.Contains(err.Error(), "oauth_client_id") {
		t.Errorf("error = %v, want it to point at the app.yml key", err)
	}
}

func TestPollToken(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus PollStatus
		wantToken  string
		wantSlow   bool
	}{
		{"authorized", `{"access_token":"gho_abc","token_type":"bearer"}`, StatusAuthorized, "gho_abc", false},
		{"pending", `{"error":"authorization_pending"}`, StatusPending, "", false},
		{"slow down", `{"error":"slow_down"}`, StatusPending, "", true},
		{"denied", `{"error":"access_denied"}`, StatusDenied, "", false},
		{"expired", `{"error":"expired_token"}`, StatusExpired, "", false},
		{"bad device code", `{"error":"incorrect_device_code"}`, StatusFailed, "", false},
		{"empty token", `{"access_token":""}`, StatusFailed, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})

			got := client.PollToken(context.Background(), "dc_secret")

			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.Token != tt.wantToken {
				t.Errorf("Token = %q, want %q", got.Token, tt.wantToken)
			}
			if got.SlowDown != tt.wantSlow {
				t.Errorf("SlowDown = %v, want %v", got.SlowDown, tt.wantSlow)
			}
		})
	}
}

func TestPollTokenSendsDeviceGrantType(t *testing.T) {
	var gotGrant, gotDeviceCode string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotGrant = r.Form.Get("grant_type")
		gotDeviceCode = r.Form.Get("device_code")
		_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
	})

	client.PollToken(context.Background(), "dc_secret")

	if gotGrant != "urn:ietf:params:oauth:grant-type:device_code" {
		t.Errorf("grant_type = %q, want the RFC 8628 device_code grant", gotGrant)
	}
	if gotDeviceCode != "dc_secret" {
		t.Errorf("device_code = %q, want dc_secret", gotDeviceCode)
	}
}

// A network blip must not abort a login the user is still approving.
func TestPollTokenStaysPendingOnTransportError(t *testing.T) {
	client, err := NewClient("test_client_id", "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.accessTokenURL = "http://127.0.0.1:1/access_token" // nothing listening

	got := client.PollToken(context.Background(), "dc_secret")
	if got.Status != StatusPending {
		t.Errorf("Status = %q on a transport error, want %q so polling continues", got.Status, StatusPending)
	}
}

func TestAccountLogin(t *testing.T) {
	var gotAuth string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"login":"octocat"}`))
	})

	login, err := client.AccountLogin(context.Background(), "gho_abc")
	if err != nil {
		t.Fatalf("AccountLogin: %v", err)
	}
	if login != "octocat" {
		t.Errorf("login = %q, want octocat", login)
	}
	if gotAuth != "Bearer gho_abc" {
		t.Errorf("Authorization = %q, want Bearer gho_abc", gotAuth)
	}
}

func TestDeviceCodeDefaults(t *testing.T) {
	// A reply missing expires_in/interval must not expire instantly or busy-loop.
	var code DeviceCode

	if got := code.PollInterval(); got != 5*time.Second {
		t.Errorf("PollInterval with no interval = %v, want 5s", got)
	}

	now := time.Unix(1_700_000_000, 0)
	if got := code.Expiry(now); !got.Equal(now.Add(15 * time.Minute)) {
		t.Errorf("Expiry with no expires_in = %v, want now+15m", got)
	}
}

// GitHub answers a bare 404 (no JSON body) when the client ID does not resolve
// to an app. "HTTP 404" tells an operator nothing; name the setting instead.
func TestRequestDeviceCodeMaps404ToBadClientID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.RequestDeviceCode(context.Background())
	if err == nil {
		t.Fatal("RequestDeviceCode succeeded on HTTP 404, want an error")
	}
	if !strings.Contains(err.Error(), "oauth_client_id") {
		t.Errorf("error = %v, want it to point at the app.yml key", err)
	}
}
