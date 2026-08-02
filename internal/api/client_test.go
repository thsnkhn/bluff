package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientRequiresSecureRemoteURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "production HTTPS", url: "https://api.bluff.example"},
		{name: "localhost HTTP", url: "http://localhost:8787"},
		{name: "loopback HTTP", url: "http://127.0.0.1:8787"},
		{name: "remote HTTP", url: "http://api.bluff.example", wantErr: true},
		{name: "missing host", url: "/v1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewClient(tt.url, http.DefaultClient)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginSendsCredentialsAndDecodesSession(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/auth/login" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["username"] != "bluff" || body["password"] != "long-password" {
			t.Errorf("unexpected credentials: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"user":{"id":"u1","username":"bluff","role":"admin"},"token":"secret","expiresAt":"2026-09-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	session, err := client.Login(context.Background(), "bluff", "long-password")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if session.Token != "secret" || session.User.Role != "admin" {
		t.Fatalf("unexpected session: %#v", session)
	}
}

func TestBootstrapAuthenticatesAndReturnsProblem(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer expired" {
			t.Errorf("Authorization = %q", got)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"unauthorized","message":"Login is required"},"requestId":"req-1"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.Bootstrap(context.Background(), "expired")
	if !IsUnauthorized(err) {
		t.Fatalf("Bootstrap error = %v, want unauthorized", err)
	}
	if !strings.Contains(err.Error(), "req-1") {
		t.Fatalf("error %q does not include request ID", err)
	}
}

func TestUsersListsAccounts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/auth/users" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer admin-token" {
			t.Errorf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"users":[{"id":"u1","username":"bluff","role":"admin"}]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	users, err := client.Users(context.Background(), "admin-token")
	if err != nil {
		t.Fatalf("Users: %v", err)
	}
	if len(users) != 1 || users[0].Username != "bluff" {
		t.Fatalf("users = %#v", users)
	}
}

func TestCreateInvitationUsesAuthenticatedEndpoint(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/auth/invitations" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer admin-token" {
			t.Errorf("Authorization = %q", got)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true,"data":{"invitation":{"code":"A1B2C3","createdAt":"2026-08-02T00:00:00Z"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	invitation, err := client.CreateInvitation(context.Background(), "admin-token")
	if err != nil {
		t.Fatalf("CreateInvitation: %v", err)
	}
	if invitation.Code != "A1B2C3" {
		t.Fatalf("invitation = %#v", invitation)
	}
}

func TestInvitationOnboardingEndpoints(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.URL.Path != "/v1/auth/invitations/validate" {
				t.Errorf("validate path = %q", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"ok":true,"data":{"valid":true}}`))
		case 2:
			if r.URL.Path != "/v1/auth/invitations/redeem" {
				t.Errorf("redeem path = %q", r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ok":true,"data":{"user":{"id":"u2","username":"dealer","role":"member"},"token":"secret","expiresAt":"2026-09-01T00:00:00Z"}}`))
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.ValidateInvitation(context.Background(), "A1B2C3"); err != nil {
		t.Fatalf("ValidateInvitation: %v", err)
	}
	session, err := client.RedeemInvitation(context.Background(), "A1B2C3", "dealer", "dealpass")
	if err != nil {
		t.Fatalf("RedeemInvitation: %v", err)
	}
	if session.User.Username != "dealer" || session.Token != "secret" {
		t.Fatalf("session = %#v", session)
	}
}
