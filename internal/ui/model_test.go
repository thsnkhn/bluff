package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/thsnkhn/bluff/internal/api"
	"github.com/thsnkhn/bluff/internal/credentials"
)

type fakeAPI struct {
	bootstrap api.Bootstrap
	users     []api.User
}

func (f fakeAPI) Health(context.Context) error { return nil }
func (f fakeAPI) Login(context.Context, string, string) (api.Session, error) {
	return api.Session{}, errors.New("not implemented")
}
func (f fakeAPI) ValidateInvitation(context.Context, string) error { return nil }
func (f fakeAPI) RedeemInvitation(context.Context, string, string, string) (api.Session, error) {
	return api.Session{}, errors.New("not implemented")
}
func (f fakeAPI) Me(context.Context, string) (api.User, error) {
	return api.User{ID: "u1", Username: "bluff", Role: "admin"}, nil
}
func (f fakeAPI) Bootstrap(context.Context, string) (api.Bootstrap, error) {
	return f.bootstrap, nil
}
func (f fakeAPI) Users(context.Context, string) ([]api.User, error) { return f.users, nil }
func (f fakeAPI) CreateInvitation(context.Context, string) (api.Invitation, error) {
	return api.Invitation{Code: "A1B2C3"}, nil
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
		{name: "saved session opens authenticated menu", store: fakeStore{token: "saved"}, want: sessionRestoredMsg{}},
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
	for _, phrase := range []string{"▒▒▒▒▒▒▒▒▒", "Sign in", "Have an invite code?", "Check connection", "About Bluff", "Quit"} {
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

func TestAdminMenuShowsUsersAndKeepsFutureItemsDisabled(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{Version: "v0.1.0"})
	model.screen = appMenuScreen
	model.loading = false
	model.width = 100
	model.height = 40
	model.user = api.User{Username: "bluff", Role: "admin"}
	model.connected = true

	view := model.View().Content
	for _, phrase := range []string{"@bluff", "ADMIN", "Users", "Tables  soon", "Games  soon", "Game formats  soon", "My info  soon", "Log out"} {
		if !strings.Contains(view, phrase) {
			t.Errorf("authenticated menu does not contain %q", phrase)
		}
	}
	if strings.Contains(view, "Players") {
		t.Fatal("authenticated menu still exposes Players")
	}
	for _, removed := range []string{"Private games. One honest ledger.", "What would you like to do?"} {
		if strings.Contains(view, removed) {
			t.Errorf("authenticated menu still contains %q", removed)
		}
	}
	if !strings.Contains(view, "Choose your next move.") {
		t.Fatal("authenticated menu does not contain the navigation prompt")
	}
}

func TestSharedHelpBarIsPinnedAcrossScreens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		screen screen
		setup  func(*Model)
	}{
		{name: "home", screen: homeScreen},
		{name: "login", screen: loginScreen},
		{name: "invite code", screen: inviteCodeScreen, setup: func(model *Model) { model.resetInviteCodeForm() }},
		{name: "invite account", screen: inviteAccountScreen, setup: func(model *Model) {
			model.invite = &inviteValues{code: "A1B2C3"}
			model.resetInviteAccountForm()
		}},
		{name: "about", screen: aboutScreen},
		{name: "authenticated menu", screen: appMenuScreen},
		{name: "users", screen: usersScreen},
		{name: "dashboard", screen: dashboardScreen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
			model.screen, model.loading = tt.screen, false
			model.width, model.height = 120, 36
			model.connected = true
			model.user = api.User{Username: "bluff", Role: "admin"}
			if tt.setup != nil {
				tt.setup(&model)
			}
			lines := strings.Split(model.View().Content, "\n")
			bottom := lines[len(lines)-1]
			if !strings.Contains(bottom, "Connected") {
				t.Fatalf("bottom help bar = %q, want connection status", bottom)
			}
		})
	}
}

func TestAboutViewUsesProductCopy(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{Version: "v0.1.0"})
	model.screen, model.loading = aboutScreen, false
	model.width, model.height = 110, 36

	view := model.View().Content
	for _, phrase := range []string{
		"Track games. Settle balances. Keep the table honest.",
		"Fast, accessible, and easy for everyone at the table.",
		"Made with love by ",
		"@thsnkhn",
	} {
		if !strings.Contains(view, phrase) {
			t.Errorf("about view does not contain %q", phrase)
		}
	}
	if strings.Contains(view, "keychain") {
		t.Fatal("about view still mentions implementation details")
	}
}

func TestLoginOmitsTaglineAndPinsFormHelp(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = loginScreen, false
	model.width, model.height = 100, 36
	model.connected = true

	view := model.View().Content
	for _, removed := range []string{"Private games. One honest ledger.", "Sign in to take your seat"} {
		if strings.Contains(view, removed) {
			t.Errorf("login still contains %q", removed)
		}
	}
	for _, description := range []string{"Your table username", "Your password is never stored"} {
		if strings.Contains(view, description) {
			t.Errorf("login still contains field description %q", description)
		}
	}
	bottom := strings.Split(view, "\n")[model.height-1]
	if !strings.Contains(bottom, "esc back") {
		t.Fatalf("bottom help bar = %q, want login help", bottom)
	}
}

func TestMemberMenuDoesNotShowUsers(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = appMenuScreen, false
	model.width, model.height = 100, 40
	model.user = api.User{Username: "guest", Role: "member"}

	if strings.Contains(model.View().Content, "Users") {
		t.Fatal("member menu exposes admin-only Users")
	}
}

func TestUsersViewPinsHelpToTerminalBottom(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = usersScreen, false
	model.width, model.height = 100, 32
	model.user = api.User{Username: "bluff", Role: "admin"}
	model.connected = true
	model.users = []api.User{{Username: "bluff", Role: "admin"}, {Username: "dealer", Role: "member"}}

	lines := strings.Split(model.View().Content, "\n")
	plainView := ansi.Strip(model.View().Content)
	if len(lines) != model.height {
		t.Fatalf("rendered lines = %d, want %d", len(lines), model.height)
	}
	if !strings.Contains(lines[len(lines)-1], "mouse click") {
		t.Fatalf("bottom line = %q, want pinned help", lines[len(lines)-1])
	}
	for _, phrase := range []string{"~bluff", "/ users", "@bluff", "c  Create invite code", "@dealer", "MEMBER"} {
		if !strings.Contains(plainView, phrase) {
			t.Errorf("users screen does not contain %q", phrase)
		}
	}
	if strings.Contains(model.View().Content, "▒▒▒") {
		t.Fatal("users screen still renders the large Bluff logo")
	}
	if !strings.Contains(lines[1], "~bluff") {
		t.Fatalf("top page header = %q, want ~bluff breadcrumb", lines[1])
	}
	plainHeader := ansi.Strip(lines[1])
	if strings.Contains(plainHeader, "@bluff") || strings.Contains(plainHeader, "ADMIN") {
		t.Fatalf("page header still contains signed-in identity: %q", plainHeader)
	}
	plainFooter := ansi.Strip(lines[len(lines)-1])
	for _, phrase := range []string{"Connected", "@bluff", "ADMIN"} {
		if !strings.Contains(plainFooter, phrase) {
			t.Fatalf("bottom status %q does not contain %q", plainFooter, phrase)
		}
	}
	if strings.Contains(plainFooter, "api.bluff.thsnkhn.com") {
		t.Fatalf("bottom status exposes API hostname: %q", plainFooter)
	}
}

func TestPageHeaderFillsAvailableWidthWithRule(t *testing.T) {
	t.Parallel()
	const width = 120
	header := pageHeader(width, "users", "dealer")
	if got := ansi.StringWidth(header); got != width {
		t.Fatalf("page header width = %d, want %d", got, width)
	}
	plain := ansi.Strip(header)
	if !strings.Contains(plain, "~bluff / users / dealer ") || !strings.HasSuffix(plain, "////") {
		t.Fatalf("unexpected page header %q", plain)
	}
}

func TestActionBarUsesOnlyItsIntrinsicWidth(t *testing.T) {
	t.Parallel()
	items := usersActionBarItems()
	bar := actionBar(items, "")

	contentWidth := 0
	for _, item := range items {
		contentWidth += lipgloss.Width(item.key + "  " + item.label)
	}
	wantWidth := contentWidth + actionBarGap*(len(items)-1) + 4 // padding and border
	if got := lipgloss.Width(bar); got != wantWidth {
		t.Fatalf("action bar width = %d, want intrinsic width %d", got, wantWidth)
	}
	if got := lipgloss.Width(bar); got >= 96 {
		t.Fatalf("action bar stretches to page width: %d", got)
	}
}

func TestActionBarHitRegionsFollowIntrinsicLayout(t *testing.T) {
	t.Parallel()
	items := usersActionBarItems()
	const x, y = 2, 3
	regions := actionBarHitRegions(x, y, items)
	if len(regions) != len(items) {
		t.Fatalf("hit regions = %d, want %d", len(regions), len(items))
	}
	for index, region := range regions {
		if region.value != items[index].action {
			t.Fatalf("region %d action = %q, want %q", index, region.value, items[index].action)
		}
		if index > 0 && region.x0 <= regions[index-1].x1 {
			t.Fatalf("region %d overlaps previous region", index)
		}
	}
	wantRightEdge := x + lipgloss.Width(actionBar(items, "")) - 1
	if got := regions[len(regions)-1].x1; got != wantRightEdge {
		t.Fatalf("last region edge = %d, want bar edge %d", got, wantRightEdge)
	}
}

func TestActionBarHoverAddsVisualStateWithoutChangingWidth(t *testing.T) {
	t.Parallel()
	items := usersActionBarItems()
	plain := actionBar(items, "")
	hovered := actionBar(items, "refresh")
	if plain == hovered {
		t.Fatal("hovered action bar has no visual change")
	}
	if lipgloss.Width(hovered) != lipgloss.Width(plain) {
		t.Fatalf("hover changed action bar width from %d to %d", lipgloss.Width(plain), lipgloss.Width(hovered))
	}
}

func TestUsersActionBarTracksMouseHover(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = usersScreen, false

	updated, _ := model.Update(usersMouseMsg{action: "refresh", userIndex: -1})
	hovered := updated.(Model)
	if hovered.usersActionHover != "refresh" {
		t.Fatalf("hovered action = %q, want refresh", hovered.usersActionHover)
	}

	updated, _ = hovered.Update(usersMouseMsg{userIndex: -1})
	cleared := updated.(Model)
	if cleared.usersActionHover != "" {
		t.Fatalf("hovered action = %q after leaving action bar, want empty", cleared.usersActionHover)
	}
}

func TestInvitePasswordAcceptsEightCharacters(t *testing.T) {
	t.Parallel()
	if err := validPassword("12345678"); err != nil {
		t.Fatalf("8-character password rejected: %v", err)
	}
	if err := validPassword("1234567"); err == nil {
		t.Fatal("7-character password accepted")
	}
}

func TestInviteCodeInputUppercasesTypedCharacters(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = inviteCodeScreen, false
	model.resetInviteCodeForm()
	model.form.Init()

	for _, character := range "a1b2c3" {
		updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: character, Text: string(character)}))
		model = updated.(Model)
	}

	if model.invite.code != "A1B2C3" {
		t.Fatalf("invite code = %q, want A1B2C3", model.invite.code)
	}
}

func TestCenteredInputOmitsPromptAndCentersPlaceholderAndText(t *testing.T) {
	t.Parallel()
	value := ""
	input := newCenteredInput("Invite code", "Enter your code", "A1B2C3", &value, 6, false, inviteCode)
	input.WithTheme(huh.ThemeFunc(centeredFormTheme))
	input.WithKeyMap(huh.NewDefaultKeyMap())
	input.WithWidth(40)
	input.Focus()

	assertCenteredLine(t, ansi.Strip(input.View()), "A1B2C3", 40)
	if strings.Contains(ansi.Strip(input.View()), ">") {
		t.Fatal("centered input still renders the prompt arrow")
	}

	for _, character := range "abc123" {
		_, _ = input.Update(tea.KeyPressMsg(tea.Key{Code: character, Text: string(character)}))
	}
	assertCenteredLine(t, ansi.Strip(input.View()), "abc123", 40)
}

func TestCenteredInputRendersValidationErrorBelowField(t *testing.T) {
	t.Parallel()
	value := ""
	input := newCenteredInput("Invite code", "", "A1B2C3", &value, 6, false, inviteCode)
	input.WithTheme(huh.ThemeFunc(centeredFormTheme))
	input.WithKeyMap(huh.NewDefaultKeyMap())
	input.WithWidth(40)
	input.WithPosition(huh.FieldPosition{})
	input.Focus()
	_, _ = input.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	view := ansi.Strip(input.View())
	errorIndex := strings.Index(view, "enter all 6 characters")
	borderIndex := strings.LastIndex(view, "━")
	if errorIndex < 0 || errorIndex < borderIndex {
		t.Fatalf("field error is not below its border: %q", view)
	}
}

func assertCenteredLine(t *testing.T, view, content string, width int) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		index := strings.Index(line, content)
		if index < 0 {
			continue
		}
		left := ansi.StringWidth(line[:index])
		right := width - left - ansi.StringWidth(content)
		if difference := left - right; difference < -1 || difference > 1 {
			t.Fatalf("content %q is not centered: left=%d right=%d", content, left, right)
		}
		return
	}
	t.Fatalf("content %q not found in view %q", content, view)
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
