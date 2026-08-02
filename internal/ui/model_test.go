package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/thsnkhn/bluff/internal/api"
	"github.com/thsnkhn/bluff/internal/credentials"
)

type fakeAPI struct {
	bootstrap api.Bootstrap
}

func (f fakeAPI) Health(context.Context) error { return nil }
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
		{name: "no session opens home", store: fakeStore{err: credentials.ErrNotFound}, want: loginRequiredMsg{}},
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

func TestHomeNavigationWraps(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen = homeScreen
	model.loading = false

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	got := updated.(Model)
	if got.homeIndex != homeQuit {
		t.Fatalf("selected item = %d, want quit", got.homeIndex)
	}
}

func TestHomeMouseHoverAndClick(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen = homeScreen
	model.loading = false
	model.width, model.height = 100, 40
	region := model.homeHitRegions()[homeAbout]

	updated, _ := model.Update(menuMouseMsg{index: homeAbout})
	hovered := updated.(Model)
	if hovered.homeIndex != homeAbout {
		t.Fatalf("hovered item = %d, want about", hovered.homeIndex)
	}
	if !inRegion(region.x0, region.y0, region) {
		t.Fatal("expected generated mouse region to contain its origin")
	}

	updated, _ = hovered.Update(menuMouseMsg{index: homeAbout, activate: true})
	clicked := updated.(Model)
	if clicked.screen != aboutScreen {
		t.Fatalf("screen = %v, want about", clicked.screen)
	}
}

func TestHomeViewUsesCompactCenteredMenu(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen = homeScreen
	model.loading = false
	model.width, model.height = 100, 36

	view := model.View().Content
	for _, phrase := range []string{"▒▒▒▒▒▒▒▒▒", "Sign in", "Check connection", "About Bluff", "Quit"} {
		if !strings.Contains(view, phrase) {
			t.Errorf("home does not contain %q", phrase)
		}
	}
	for _, oldMark := range []string{"♦", "♣", "♠", "♥"} {
		if strings.Contains(view, oldMark) {
			t.Errorf("home still contains old suit mark %q", oldMark)
		}
	}
	for _, description := range []string{"Open your private table", "Verify the Bluff service", "Version and product details", "Leave the table"} {
		if strings.Contains(view, description) {
			t.Errorf("home still contains menu description %q", description)
		}
	}
}

func TestLoginFormKeepsTypedValueAcrossModelCopies(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	copied := model
	copied.form.Init()

	for _, character := range "bluff" {
		updated, _ := copied.form.Update(tea.KeyPressMsg(tea.Key{
			Code: character,
			Text: string(character),
		}))
		form, ok := updated.(*huh.Form)
		if !ok {
			t.Fatalf("form update returned %T", updated)
		}
		copied.form = form
	}

	if copied.login.username != "bluff" {
		t.Fatalf("submitted username = %q, want %q", copied.login.username, "bluff")
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
	for _, phrase := range []string{"Standings", "No players yet", "The table is clear", "No hands in the book", "v0.1.0", "////"} {
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
