package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/thsnkhn/bluff/internal/api"
	"github.com/thsnkhn/bluff/internal/credentials"
)

type fakeAPI struct {
	bootstrap api.Bootstrap
}

func (f fakeAPI) Login(context.Context, string, string) (api.Session, error) {
	return api.Session{}, errors.New("not implemented")
}
func (f fakeAPI) Me(context.Context, string) (api.User, error) {
	return api.User{ID: "u1", Username: "bluff", Role: "admin"}, nil
}
func (f fakeAPI) Bootstrap(context.Context, string) (api.Bootstrap, error) {
	return f.bootstrap, nil
}
func (f fakeAPI) Logout(context.Context, string) error { return nil }

type fakeStore struct {
	token string
	err   error
}

func (f fakeStore) Load(context.Context) (string, error) { return f.token, f.err }
func (f fakeStore) Save(context.Context, string) error   { return nil }
func (f fakeStore) Delete(context.Context) error         { return nil }

func TestRestoreSessionCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		store fakeStore
		want  any
	}{
		{name: "no session opens login", store: fakeStore{err: credentials.ErrNotFound}, want: loginRequiredMsg{}},
		{name: "saved session opens dashboard", store: fakeStore{token: "saved"}, want: sessionRestoredMsg{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := New(fakeAPI{}, tt.store, BuildInfo{})
			msg := model.restoreSessionCmd()()
			switch tt.want.(type) {
			case loginRequiredMsg:
				if _, ok := msg.(loginRequiredMsg); !ok {
					t.Fatalf("message = %T, want loginRequiredMsg", msg)
				}
			case sessionRestoredMsg:
				restored, ok := msg.(sessionRestoredMsg)
				if !ok || restored.token != "saved" {
					t.Fatalf("message = %#v, want restored saved token", msg)
				}
			}
		})
	}
}

func TestDashboardViewRendersUsefulEmptyState(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{Version: "v0.1.0"})
	model.screen = dashboardScreen
	model.loading = false
	model.width = 120
	model.height = 40
	model.user = api.User{Username: "bluff", Role: "admin"}

	view := model.View().Content
	for _, phrase := range []string{"STANDINGS", "No players yet", "The table is clear", "No hands in the book", "v0.1.0"} {
		if !strings.Contains(view, phrase) {
			t.Errorf("dashboard does not contain %q", phrase)
		}
	}
}

func TestSortedPlayersUsesStandingThenName(t *testing.T) {
	t.Parallel()
	players := sortedPlayers([]api.Player{
		{Name: "Zara", Standing: 10},
		{Name: "Ali", Standing: 10},
		{Name: "Hamza", Standing: -20},
	})
	got := []string{players[0].Name, players[1].Name, players[2].Name}
	want := []string{"Ali", "Zara", "Hamza"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}
