package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
	health    api.HealthStatus
}

func (f fakeAPI) Health(context.Context) error { return nil }
func (f fakeAPI) HealthStatus(context.Context) (api.HealthStatus, error) {
	if f.health.Status == "" {
		return api.HealthStatus{Status: "ok"}, nil
	}
	return f.health, nil
}
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
func (f fakeAPI) Tables(context.Context, string) ([]api.TableSummary, error) {
	return f.bootstrap.Tables, nil
}
func (f fakeAPI) Table(context.Context, string, string) (api.TableDetail, error) {
	return api.TableDetail{}, errors.New("not implemented")
}
func (f fakeAPI) CreateTable(context.Context, string, string) (api.TableSummary, error) {
	return api.TableSummary{}, errors.New("not implemented")
}
func (f fakeAPI) CreateTablePlayer(context.Context, string, string, string) (api.TablePlayer, error) {
	return api.TablePlayer{}, errors.New("not implemented")
}
func (f fakeAPI) UpdateTablePlayer(context.Context, string, string, string, string) (api.TablePlayer, error) {
	return api.TablePlayer{}, errors.New("not implemented")
}
func (f fakeAPI) DeleteTablePlayer(context.Context, string, string, string) error  { return nil }
func (f fakeAPI) DisableTablePlayer(context.Context, string, string, string) error { return nil }
func (f fakeAPI) CreateGameFormat(context.Context, string, string, string, int, []api.ChipDenomination) (api.GameFormat, error) {
	return api.GameFormat{}, errors.New("not implemented")
}
func (f fakeAPI) UpdateGameFormat(context.Context, string, string, string, string, int, []api.ChipDenomination) (api.GameFormat, error) {
	return api.GameFormat{}, errors.New("not implemented")
}
func (f fakeAPI) PreviewTableGame(context.Context, string, string, string, string, string, []api.GameParticipantInput) (api.TableGame, error) {
	return api.TableGame{}, errors.New("not implemented")
}
func (f fakeAPI) RecordTableGame(context.Context, string, string, string, string, string, []api.GameParticipantInput) (api.TableDetail, error) {
	return api.TableDetail{}, errors.New("not implemented")
}
func (f fakeAPI) PreviewTableGameEdit(context.Context, string, string, string, string, string, string, []api.GameParticipantInput) (api.TableGame, error) {
	return api.TableGame{}, errors.New("not implemented")
}
func (f fakeAPI) UpdateTableGame(context.Context, string, string, string, string, string, string, int, []api.GameParticipantInput) (api.TableDetail, error) {
	return api.TableDetail{}, errors.New("not implemented")
}
func (f fakeAPI) Logout(context.Context, string) error { return nil }

type fakeStore struct {
	token string
	err   error
}

type fakeInstaller struct {
	release api.ClientRelease
	err     error
}

func (f *fakeInstaller) Install(_ context.Context, release api.ClientRelease) error {
	f.release = release
	return f.err
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

func TestRestoreSessionRestartsAfterNewerRelease(t *testing.T) {
	t.Parallel()
	installer := &fakeInstaller{}
	model := New(fakeAPI{health: api.HealthStatus{Status: "ok", ClientVersion: "v0.1.4"}}, fakeStore{err: credentials.ErrNotFound}, BuildInfo{Version: "v0.1.3"}, installer)
	msg := model.restoreSessionCmd()()
	if _, ok := msg.(updateRestartedMsg); !ok {
		t.Fatalf("message = %T, want updateRestartedMsg", msg)
	}
	if installer.release.Version != "v0.1.4" {
		t.Fatalf("installer version = %q, want v0.1.4", installer.release.Version)
	}
}

func TestBootShowsUpdateProgressBeforeInstalling(t *testing.T) {
	t.Parallel()
	model := New(
		fakeAPI{health: api.HealthStatus{Status: "ok", ClientVersion: "v0.1.6"}},
		fakeStore{err: credentials.ErrNotFound},
		BuildInfo{Version: "v0.1.5"},
		&fakeInstaller{},
	)

	updated, cmd := model.Update(model.bootHealthCmd()())
	got := updated.(Model)
	if cmd == nil {
		t.Fatal("boot update command is nil")
	}
	if got.status != "Downloading update v0.1.6" {
		t.Fatalf("boot status = %q, want update progress", got.status)
	}
}

func TestBootFallsBackToSessionRestoreWithoutUpdate(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{err: credentials.ErrNotFound}, BuildInfo{Version: "v0.1.5"})

	updated, cmd := model.Update(model.bootHealthCmd()())
	got := updated.(Model)
	if cmd == nil {
		t.Fatal("session restore command is nil")
	}
	if got.status != "Restoring session" {
		t.Fatalf("boot status = %q, want session restore", got.status)
	}
}

func TestUpdateFailureDoesNotPoisonSessionContext(t *testing.T) {
	t.Parallel()
	installer := &fakeInstaller{err: context.DeadlineExceeded}
	model := New(
		fakeAPI{health: api.HealthStatus{Status: "ok", ClientVersion: "v0.1.6"}},
		fakeStore{err: credentials.ErrNotFound},
		BuildInfo{Version: "v0.1.5"},
		installer,
	)

	msg := model.restoreSessionCmd()()
	required, ok := msg.(loginRequiredMsg)
	if !ok {
		t.Fatalf("message = %T, want loginRequiredMsg", msg)
	}
	if required.err != nil {
		t.Fatalf("login error = %v, want no keychain error after update failure", required.err)
	}
	if required.update == nil || required.update.Version != "v0.1.6" {
		t.Fatalf("update = %#v, want v0.1.6", required.update)
	}
}

func TestConnectionLineUsesConnectingWhileHealthCheckRuns(t *testing.T) {
	t.Parallel()
	for _, render := range []func(bool, bool, string) string{connectionLine, compactConnectionLine} {
		line := render(false, true, "·")
		if !strings.Contains(line, "Connecting") {
			t.Fatalf("line = %q, want Connecting", line)
		}
		if strings.Contains(line, "Offline") {
			t.Fatalf("line = %q, must not say Offline while connecting", line)
		}
	}
}

func TestHelpBarUsesFullWidthSegmentedStatusLayout(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{Version: "v0.2.0"})
	model.width = 120
	model.connected = true
	model.user = api.User{Username: "bluff", Role: "admin"}

	bar := model.helpBar("↑↓ move   enter open   esc back")
	plain := ansi.Strip(bar)
	if ansi.StringWidth(bar) != model.width {
		t.Fatalf("help bar width = %d, want %d", ansi.StringWidth(bar), model.width)
	}
	for _, want := range []string{"connected", "@bluff ADMIN", "enter open"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("help bar missing %q: %q", want, plain)
		}
	}
	if strings.Contains(plain, "BLUFF") || strings.Contains(plain, "v0.2.0") {
		t.Fatalf("help bar still contains build branding: %q", plain)
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
	for _, phrase := range []string{"@bluff", "ADMIN", "Users", "Tables", "My info  soon", "Log out"} {
		if !strings.Contains(view, phrase) {
			t.Errorf("authenticated menu does not contain %q", phrase)
		}
	}
	for _, removed := range []string{"Games", "Game formats"} {
		if strings.Contains(view, removed) {
			t.Errorf("authenticated menu still contains standalone %q", removed)
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
	if strings.Contains(view, "Choose your next move.") {
		t.Fatal("authenticated menu still contains the removed navigation prompt")
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
			if !strings.Contains(bottom, "connected") {
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
	if strings.Contains(bottom, "esc") || strings.Contains(bottom, "refresh") || strings.Contains(bottom, "search") {
		t.Fatalf("bottom help bar exposes hidden shortcuts: %q", bottom)
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
	bottom := lines[len(lines)-1]
	if strings.Contains(bottom, "refresh") || strings.Contains(bottom, "search") || strings.Contains(bottom, "esc") || strings.Contains(bottom, "mouse click") {
		t.Fatalf("bottom line exposes hidden shortcuts: %q", bottom)
	}
	for _, phrase := range []string{"~bluff", "/ users", "@bluff", "c  Create invite code", "@dealer", "ADMIN"} {
		if !strings.Contains(plainView, phrase) {
			t.Errorf("users screen does not contain %q", phrase)
		}
	}
	if strings.Contains(plainView, "@dealer MEMBER") {
		t.Fatal("member row still renders a trailing MEMBER role")
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
	for _, phrase := range []string{"connected", "@bluff", "ADMIN"} {
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
	visible := visibleActionBarItems(items)

	contentWidth := 0
	for _, item := range visible {
		contentWidth += lipgloss.Width(item.key + "  " + item.label)
	}
	wantWidth := contentWidth + actionBarGap*(len(visible)-1) + 4 // padding and border
	if got := lipgloss.Width(bar); got != wantWidth {
		t.Fatalf("action bar width = %d, want intrinsic width %d", got, wantWidth)
	}
	if got := lipgloss.Width(bar); got >= 96 {
		t.Fatalf("action bar stretches to page width: %d", got)
	}
	if strings.Contains(ansi.Strip(bar), "esc  Back") {
		t.Fatalf("action bar contains global back help: %q", ansi.Strip(bar))
	}
}

func TestActionBarHitRegionsFollowIntrinsicLayout(t *testing.T) {
	t.Parallel()
	items := usersActionBarItems()
	visible := visibleActionBarItems(items)
	const x, y = 2, 3
	regions := actionBarHitRegions(x, y, items)
	if len(regions) != len(visible) {
		t.Fatalf("hit regions = %d, want %d", len(regions), len(visible))
	}
	for index, region := range regions {
		if region.value != visible[index].action {
			t.Fatalf("region %d action = %q, want %q", index, region.value, visible[index].action)
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

func TestTableRecordFlowOffersQuickAddAndGameHistory(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 36
	model.table = &api.TableDetail{
		Table:     api.TableSummary{ID: "table-1", Name: "#Saturday table", HostUsername: "bluff"},
		CanManage: true,
		Players:   []api.TablePlayer{{ID: "p1", Name: "Alice"}, {ID: "p2", Name: "Bob"}},
		Formats:   []api.GameFormat{{ID: "f1", Name: "Saturday 100", RequiredEntry: 100, Chips: []api.ChipDenomination{{ID: "c1", Label: "white", Color: "white", Value: 50}}}},
		Games:     []api.TableGame{{ID: "g1", Date: "2026-08-06", Format: api.GameFormat{Name: "Saturday 100", RequiredEntry: 100}, Status: "settled"}},
	}
	model.screen, model.loading = recordGameScreen, false
	model.recordPhase = recordPlayersPhase
	updated, _, handled := model.updateTableKey("c")
	if !handled || updated.(Model).screen != playerCreateScreen {
		t.Fatalf("quick add transition = %#v, handled=%v", updated, handled)
	}
	quickAdd := updated.(Model)
	if !quickAdd.recordQuickAdd {
		t.Fatal("quick add mode was not retained while creating the player")
	}

	quickAdd.screen = gamesScreen
	view := quickAdd.View().Content
	if !strings.Contains(view, "Saturday 100") || !strings.Contains(view, "2026-08-06") {
		t.Fatalf("games view is missing the recorded game: %q", view)
	}
	quickAdd.screen = tableDetailScreen
	if !strings.Contains(quickAdd.View().Content, "♛") {
		t.Fatal("table host is missing the crown marker")
	}
}

func TestGameInspectionMatchesReviewAndOnlyLatestGameIsEditable(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 120, 40
	model.screen, model.loading = gameDetailScreen, false
	model.table = &api.TableDetail{
		Table:     api.TableSummary{ID: "table-1", Name: "#saturday", HostUsername: "host"},
		CanManage: true,
		Players:   []api.TablePlayer{{ID: "p1", Name: "winner"}, {ID: "p2", Name: "loser"}},
		Formats:   []api.GameFormat{{ID: "f1", Name: "rookie", RequiredEntry: 100, Chips: []api.ChipDenomination{{ID: "c1", Value: 10}}}},
		Games: []api.TableGame{
			{ID: "g1", Date: "2026-08-08", Format: api.GameFormat{ID: "f1", Name: "rookie", RequiredEntry: 100}, Version: 1},
			{ID: "g2", Date: "2026-08-09", Format: api.GameFormat{ID: "f1", Name: "rookie", RequiredEntry: 100}, Version: 2,
				ExpectedTableValue: 200, ActualTableValue: 200, Participants: []api.TableGameParticipant{
					{PlayerID: "p2", PlayerName: "loser", FinalValue: 50, ProfitLoss: -50, StartingStanding: 0, ChipCounts: []api.ChipCount{{DenominationID: "c1", Color: "white", Value: 10, Count: 5}}},
					{PlayerID: "p1", PlayerName: "winner", FinalValue: 150, ProfitLoss: 50, StartingStanding: 0, ChipCounts: []api.ChipCount{{DenominationID: "c1", Color: "white", Value: 10, Count: 15}}},
				}},
		},
	}

	model.gameIndex = 0
	older := ansi.Strip(model.gameDetailView())
	if !strings.Contains(older, "Inspecting") || strings.Contains(older, "e  Edit") {
		t.Fatalf("older game inspection has the wrong actions:\n%s", older)
	}

	model.gameIndex = 1
	latest := ansi.Strip(model.gameDetailView())
	if !strings.Contains(latest, "Inspecting") || !strings.Contains(latest, "e  Edit") ||
		strings.Index(latest, "winner") > strings.Index(latest, "loser") || !strings.Contains(latest, "× 15") {
		t.Fatalf("latest game inspection is not review-style or P/L sorted:\n%s", latest)
	}
	if !lineContainsAll(latest, "Total 150 cr", "P/L +50 cr", "× 15") || strings.Contains(latest, "10 ×") {
		t.Fatalf("inspection totals, P/L, and denominations are not on the same line:\n%s", latest)
	}
	moved, _, handled := model.updateTableKey("down")
	model = moved.(Model)
	if !handled || model.resultIndex != 1 {
		t.Fatalf("inspection selector did not move: handled=%v index=%d", handled, model.resultIndex)
	}
	updated, _, handled := model.updateTableKey("e")
	edit := updated.(Model)
	if !handled || edit.screen != recordGameScreen || edit.recordEditGameID != "g2" || edit.recordEditVersion != 2 ||
		!edit.recordEntered["p1"] || !edit.recordEntered["p2"] {
		t.Fatalf("latest game did not open as a populated edit: handled=%v screen=%v id=%q version=%d", handled, edit.screen, edit.recordEditGameID, edit.recordEditVersion)
	}
	if actions := recordActionItemsText(edit.recordActionItems()); strings.Contains(actions, "Review game") {
		t.Fatalf("unchanged game edit shows review action: %s", actions)
	}
	edit.recordCounts["p1"]["c1"]++
	if actions := recordActionItemsText(edit.recordActionItems()); !strings.Contains(actions, "Review game") {
		t.Fatalf("changed game edit does not show review action: %s", actions)
	}
}

func TestGameDateLabelNumbersMultipleGamesOnTheSameDate(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{Games: []api.TableGame{
		{Date: "2026-08-08"},
		{Date: "2026-08-09"},
		{Date: "2026-08-09"},
		{Date: "2026-08-10"},
	}}

	want := []string{"2026-08-08", "2026-08-09 [1]", "2026-08-09 [2]", "2026-08-10"}
	for index, expected := range want {
		if got := model.gameDateLabel(index); got != expected {
			t.Fatalf("gameDateLabel(%d) = %q, want %q", index, got, expected)
		}
	}
}

func TestGameNoteViewIsOptionalAndFullWidth(t *testing.T) {
	t.Parallel()
	if got := gameNoteView("   ", 72); got != "" {
		t.Fatalf("empty note rendered as %q", got)
	}
	view := gameNoteView("Bring the marked deck for the next round.", 72)
	if lipgloss.Width(view) != 72 {
		t.Fatalf("note width = %d, want 72", lipgloss.Width(view))
	}
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "Note") || !strings.Contains(plain, "Bring the marked deck") {
		t.Fatalf("note content missing:\n%s", plain)
	}
}

func TestRecordPlayerEntryOpensEarningsDirectly(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{
		Table:     api.TableSummary{HostUsername: "bluff"},
		CanManage: true,
		Players:   []api.TablePlayer{{ID: "player-1", Name: "alice"}, {ID: "player-2", Name: "bob"}},
		Formats:   []api.GameFormat{{ID: "format-1", Name: "rookie", RequiredEntry: 200, Chips: []api.ChipDenomination{{ID: "chip-1", Value: 10}}}},
	}
	model.screen, model.loading = recordGameScreen, false
	model.recordPhase, model.recordFormatIndex, model.playerIndex = recordPlayersPhase, 0, 0
	model.recordCounts = map[string]map[string]int{}
	model.recordEntered = map[string]bool{}

	updated, _, handled := model.updateTableKey("enter")
	if !handled {
		t.Fatal("enter was not handled")
	}
	got := updated.(Model)
	if got.recordPhase != recordChipCountsPhase || got.recordPlayerIndex != 0 || got.form == nil {
		t.Fatalf("direct earnings transition = phase %v, player %d, form %v", got.recordPhase, got.recordPlayerIndex, got.form != nil)
	}
}

func TestRecordStartsAtFormatAndKeepsMetadataOnRecorder(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 36
	model.table = &api.TableDetail{
		Table:     api.TableSummary{Name: "#saturday", HostUsername: "bluff"},
		CanManage: true,
		Formats:   []api.GameFormat{{ID: "format-1", Name: "rookie", RequiredEntry: 200}},
		Players:   []api.TablePlayer{{ID: "player-1", Name: "alice"}},
	}
	model.loading = false
	model.startRecordGame()
	if model.recordPhase != recordFormatPhase || model.form != nil {
		t.Fatalf("record start = phase %v form=%v; want format selection without a form", model.recordPhase, model.form != nil)
	}
	if model.recordDetails == nil || model.recordDetails.date != time.Now().Format("2006-01-02") {
		t.Fatalf("record date = %#v; want today's date", model.recordDetails)
	}
	if !strings.Contains(ansi.Strip(model.recordGameView()), "/ record / "+model.recordDetails.date) {
		t.Fatalf("record header does not include date:\n%s", ansi.Strip(model.recordGameView()))
	}

	updated, _, handled := model.updateTableKey("t")
	metadataPopup := updated.(Model)
	if !handled || metadataPopup.recordPopup != recordMetadataPopup || metadataPopup.form == nil {
		t.Fatalf("metadata popup = handled=%v popup=%v form=%v; want an in-place date/note editor", handled, metadataPopup.recordPopup, metadataPopup.form != nil)
	}
	metadataPopup.recordMetadataDraft.date = "2026-08-01"
	metadataPopup.recordMetadataDraft.remarks = "discard me"
	updated, _, handled = metadataPopup.updateTableKey("esc")
	closed := updated.(Model)
	if !handled || closed.screen != recordGameScreen || closed.recordPhase != recordFormatPhase || closed.recordPopup != recordPopupNone {
		t.Fatalf("metadata popup close = handled=%v screen=%v phase=%v popup=%v; want recorder format screen", handled, closed.screen, closed.recordPhase, closed.recordPopup)
	}
	if closed.recordDetails.date == "2026-08-01" || closed.recordDetails.remarks != "" {
		t.Fatalf("closing metadata popup retained draft values: %#v", closed.recordDetails)
	}

	updated, _ = closed.updateTableMouse(tableMouseMsg{action: "metadata", activate: true})
	metadataPopup = updated.(Model)
	if metadataPopup.recordPopup != recordMetadataPopup || metadataPopup.form == nil {
		t.Fatalf("metadata popup = popup=%v form=%v; want an in-place date/note editor", metadataPopup.recordPopup, metadataPopup.form != nil)
	}
}

func TestRecordMetadataPopupsStayCompact(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 120, 50
	model.screen, model.recordPhase = recordGameScreen, recordPlayersPhase
	model.recordDetails = &recordDetailsValues{date: "2026-08-09"}

	model.openRecordMetadataPopup()
	if height := lipgloss.Height(model.form.View()); height > 14 {
		t.Fatalf("metadata form height = %d, want compact content-sized form", height)
	}

	model.recordDetails.remarks = strings.Repeat("long note ", 12)
	model.openRecordMetadataPopup()
	model.form.WithWidth(30)
	if width := lipgloss.Width(model.form.View()); width > 30 {
		t.Fatalf("note form width = %d, want textarea content wrapped within 30 columns", width)
	}
}

func TestPopupShellPinsHeaderAndActionsWithTwoRowGutters(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 30
	view := ansi.Strip(model.formPopupSizedWithActions("background", "Popup title", "Popup content", "esc close", 58, []actionBarItem{
		{key: "enter", label: "Save", action: "submit", accent: true},
		{key: "esc", label: "Close", action: "close"},
	}))
	lines := strings.Split(view, "\n")
	titleLine, contentLine, actionsLine := -1, -1, -1
	for index, line := range lines {
		switch {
		case strings.Contains(line, "Popup title"):
			titleLine = index
		case strings.Contains(line, "Popup content"):
			contentLine = index
		case strings.Contains(line, "enter  Save"):
			actionsLine = index
		}
	}
	if titleLine < 1 || contentLine-titleLine != 3 || actionsLine-contentLine != 3 {
		t.Fatalf("popup rows = title:%d content:%d actions:%d; want two-row gutters\n%s", titleLine, contentLine, actionsLine, view)
	}
	if strings.TrimSpace(lines[titleLine-1]) == "" || strings.TrimSpace(lines[actionsLine+1]) == "" {
		t.Fatalf("popup header or actions are not adjacent to the popup border:\n%s", view)
	}
}

func TestRecordMetadataPopupRendersSafelyOverEarningsPhase(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 120, 50
	model.screen, model.recordPhase, model.loading = recordGameScreen, recordChipCountsPhase, false
	model.recordFormatIndex, model.recordPlayerIndex, model.playerIndex = 0, 0, 0
	model.recordCounts, model.recordEntered = map[string]map[string]int{}, map[string]bool{}
	model.recordDetails = &recordDetailsValues{date: "2026-08-09"}
	model.table = &api.TableDetail{
		Table:   api.TableSummary{ID: "table-1", Name: "#saturday"},
		Players: []api.TablePlayer{{ID: "player-1", Name: "alice"}},
		Formats: []api.GameFormat{{
			ID:            "format-1",
			Name:          "rookie",
			RequiredEntry: 200,
			Chips:         []api.ChipDenomination{{ID: "chip-1", Label: "white", Color: "white", Value: 10}},
		}},
	}
	model.resetRecordChipForm()
	model.openRecordMetadataPopup()

	view := ansi.Strip(model.recordGameView())
	if !strings.Contains(view, "Edit date/note") || !strings.Contains(view, "Game date") || !strings.Contains(view, "Note") {
		t.Fatalf("metadata popup did not render over earnings phase:\n%s", view)
	}
}

func TestAllInEarningsSkipsDenominations(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 36
	model.screen, model.loading = recordGameScreen, false
	model.recordPhase, model.recordFormatIndex, model.recordPlayerIndex = recordChipCountsPhase, 0, 0
	model.recordCounts, model.recordEntered = map[string]map[string]int{}, map[string]bool{}
	model.table = &api.TableDetail{
		Table:   api.TableSummary{ID: "table-1"},
		Players: []api.TablePlayer{{ID: "player-1", Name: "alice"}},
		Formats: []api.GameFormat{{
			ID:            "format-1",
			Name:          "rookie",
			RequiredEntry: 200,
			Chips:         []api.ChipDenomination{{ID: "chip-1", Label: "white", Color: "white", Value: 10}},
		}},
	}
	model.resetRecordChipForm()
	model.form.Init()
	normalView := ansi.Strip(model.recordChipCountView())
	if !strings.Contains(normalView, "white 10") {
		t.Fatal("normal earnings form does not show denomination counters")
	}
	if strings.Contains(normalView, "rookie") {
		t.Fatalf("earnings popup still shows the game format:\n%s", normalView)
	}
	lines := strings.Split(normalView, "\n")
	allInLine, optionsLine := -1, -1
	for index, line := range lines {
		if strings.Contains(line, "All in?") {
			allInLine = index
		}
		if strings.Contains(line, "Yes") && strings.Contains(line, "No") {
			optionsLine = index
		}
	}
	if allInLine < 0 || optionsLine != allInLine+1 {
		t.Fatalf("all-in confirmation options are not directly below the label:\n%s", normalView)
	}

	updatedModel, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	model = updatedModel.(Model)
	view := ansi.Strip(model.recordChipCountView())
	if !model.recordAllIn || strings.Contains(view, "white 10") || !strings.Contains(view, "All in?") {
		t.Fatalf("all-in selection = %v; want denomination counters hidden immediately:\n%s", model.recordAllIn, view)
	}
	updated, _ := model.handleTableFormCompleted()
	got := updated.(Model)
	if !got.recordEntered["player-1"] || len(got.recordCounts["player-1"]) != 0 {
		t.Fatalf("all-in save = entered=%v counts=%#v; want an entered player with zero chip counts", got.recordEntered["player-1"], got.recordCounts["player-1"])
	}
}

func TestEscapingTablePopupsKeepsTheirSection(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{Table: api.TableSummary{ID: "table-1"}}

	model.screen = formatCreateScreen
	updated, _, handled := model.updateTableKey("esc")
	if !handled || updated.(Model).screen != formatsScreen {
		t.Fatalf("format popup escape = screen %v, handled=%v; want formats screen", updated.(Model).screen, handled)
	}

	model.screen = playerCreateScreen
	updated, _, handled = model.updateTableKey("esc")
	if !handled || updated.(Model).screen != playersScreen {
		t.Fatalf("player popup escape = screen %v, handled=%v; want players screen", updated.(Model).screen, handled)
	}
}

func TestEscapingRecordChipPopupKeepsRecorder(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 36
	model.screen, model.loading = recordGameScreen, false
	model.recordPhase = recordChipCountsPhase
	model.recordFormatIndex, model.recordPlayerIndex = 0, 0
	model.recordCounts, model.recordEntered = map[string]map[string]int{}, map[string]bool{}
	model.table = &api.TableDetail{
		Table:     api.TableSummary{ID: "table-1"},
		CanManage: true,
		Players:   []api.TablePlayer{{ID: "player-1", Name: "alice"}, {ID: "player-2", Name: "bob"}},
		Formats: []api.GameFormat{{
			ID:            "format-1",
			Name:          "rookie 2k",
			RequiredEntry: 2000,
			Chips: []api.ChipDenomination{
				{ID: "chip-1", Label: "white", Color: "white", Value: 200},
				{ID: "chip-2", Label: "black", Color: "black", Value: 100},
			},
		}},
	}
	model.resetRecordChipForm()

	updated, _, handled := model.updateTableKey("esc")
	got := updated.(Model)
	if !handled || got.screen != recordGameScreen || got.recordPhase != recordPlayersPhase {
		t.Fatalf("chip popup escape = handled=%v screen=%v phase=%v; want recorder player list", handled, got.screen, got.recordPhase)
	}
	if got.form != nil {
		t.Fatal("chip popup form remains open after escape")
	}
}

func TestClearEarningIsOnlyAvailableInsideEditPopup(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 36
	model.screen, model.recordPhase, model.loading = recordGameScreen, recordPlayersPhase, false
	model.recordFormatIndex, model.playerIndex = 0, 0
	model.recordCounts = map[string]map[string]int{"player-1": {"chip-1": 2}}
	model.recordEntered = map[string]bool{"player-1": true}
	model.table = &api.TableDetail{
		Table:     api.TableSummary{ID: "table-1"},
		CanManage: true,
		Players:   []api.TablePlayer{{ID: "player-1", Name: "alice"}},
		Formats: []api.GameFormat{{
			ID:            "format-1",
			Name:          "rookie 2k",
			RequiredEntry: 2000,
			Chips:         []api.ChipDenomination{{ID: "chip-1", Label: "white", Color: "white", Value: 200}},
		}},
	}

	if view := ansi.Strip(model.recordGameView()); strings.Contains(view, "Clear earning") {
		t.Fatalf("player list action bar contains clear earning:\n%s", view)
	}

	model.beginRecordPlayerEarnings()
	if view := ansi.Strip(model.recordGameView()); !strings.Contains(view, "Clear earning") {
		t.Fatalf("earnings edit popup does not contain clear earning:\n%s", view)
	}

	updated, cmd, handled := model.updateTableKey("alt+d")
	armed := updated.(Model)
	if !handled || cmd != nil || !armed.recordClearConfirm || armed.recordPhase != recordChipCountsPhase {
		t.Fatalf("first clear press = handled=%v cmd=%v confirm=%v phase=%v; want armed popup confirmation", handled, cmd != nil, armed.recordClearConfirm, armed.recordPhase)
	}
	if view := ansi.Strip(armed.recordGameView()); !strings.Contains(view, "Confirm Clear") {
		t.Fatalf("armed earnings popup does not contain confirmation:\n%s", view)
	}

	updated, cmd, handled = armed.updateTableKey("alt+d")
	cleared := updated.(Model)
	if !handled || cmd != nil || cleared.recordEntered["player-1"] || cleared.recordPhase != recordPlayersPhase || cleared.form != nil {
		t.Fatalf("second clear press = handled=%v cmd=%v entered=%v phase=%v form=%v; want cleared earning and closed popup", handled, cmd != nil, cleared.recordEntered["player-1"], cleared.recordPhase, cleared.form != nil)
	}
}

func TestPlayerDeleteRequiresConfirmation(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = playerDetailScreen, false
	model.width, model.height = 100, 36
	model.table = &api.TableDetail{
		Table:     api.TableSummary{ID: "table-1"},
		CanManage: true,
		Players:   []api.TablePlayer{{ID: "player-1", Name: "alice"}},
	}

	updated, cmd, handled := model.updateTableKey("d")
	unchanged := updated.(Model)
	if handled || cmd != nil || unchanged.playerDeleteConfirm {
		t.Fatalf("plain d = handled=%v cmd=%v confirm=%v; want no delete action", handled, cmd != nil, unchanged.playerDeleteConfirm)
	}

	updated, cmd, handled = model.updateTableKey("shift+d")
	unchanged = updated.(Model)
	if handled || cmd != nil || unchanged.playerDeleteConfirm {
		t.Fatalf("shift+d = handled=%v cmd=%v confirm=%v; want no delete action", handled, cmd != nil, unchanged.playerDeleteConfirm)
	}

	updated, cmd, handled = model.updateTableKey("alt+d")
	first := updated.(Model)
	if !handled || cmd != nil || !first.playerDeleteConfirm {
		t.Fatalf("first delete press = handled=%v cmd=%v confirm=%v; want armed confirmation", handled, cmd != nil, first.playerDeleteConfirm)
	}

	updated, cmd, handled = first.updateTableKey("alt+d")
	second := updated.(Model)
	if !handled || cmd == nil || second.playerDeleteConfirm || !second.loading {
		t.Fatalf("second delete press = handled=%v cmd=%v confirm=%v loading=%v; want delete command", handled, cmd != nil, second.playerDeleteConfirm, second.loading)
	}
}

func TestHostPlayerIsLockedByUserLinkAndPlayerFormsHaveNoInvites(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = playerDetailScreen, false
	model.width, model.height = 100, 36
	model.table = &api.TableDetail{
		Table:     api.TableSummary{ID: "table-1", HostUserID: "host-id", HostUsername: "alice"},
		CanManage: true,
		Players: []api.TablePlayer{
			{ID: "host-player", Name: "alice", UserID: "host-id", Username: "alice"},
			{ID: "ordinary-player", Name: "alice"},
		},
	}

	model.playerIndex = 0
	model.resetPlayerEditForm()
	if model.form != nil {
		t.Fatal("host player received an editable form")
	}
	for _, key := range []string{"alt+d", "x"} {
		updated, cmd, handled := model.updateTableKey(key)
		got := updated.(Model)
		if handled || cmd != nil || got.loading {
			t.Fatalf("host %s = handled=%v cmd=%v loading=%v; want no action", key, handled, cmd != nil, got.loading)
		}
	}

	model.playerIndex = 1
	model.resetPlayerEditForm()
	if model.form == nil {
		t.Fatal("ordinary player with the host's name was treated as the host")
	}
	model.resetPlayerCreateForm()
	if strings.Contains(strings.ToLower(ansi.Strip(model.form.View())), "invite") {
		t.Fatal("player creation form still offers an account invite")
	}
	playersView := ansi.Strip(model.playerList(80))
	if strings.Count(playersView, "♛") != 1 {
		t.Fatalf("player list host marker count = %d, want one:\n%s", strings.Count(playersView, "♛"), playersView)
	}
}

func TestFormatRowsOpenEditPopup(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 36
	model.screen, model.loading = formatsScreen, false
	model.table = &api.TableDetail{
		Table:     api.TableSummary{ID: "table-1"},
		CanManage: true,
		Formats: []api.GameFormat{{
			ID:            "format-1",
			Name:          "rookie 2k",
			RequiredEntry: 2000,
			Chips:         []api.ChipDenomination{{ID: "chip-1", Label: "white", Color: "white", Value: 200}},
		}},
	}

	updated, cmd, handled := model.updateTableKey("enter")
	got := updated.(Model)
	if !handled || got.screen != formatCreateScreen {
		t.Fatalf("format enter = handled=%v cmd=%v screen=%v; want edit popup", handled, cmd != nil, got.screen)
	}
	if got.formatEditIndex != 0 || got.form == nil || !strings.Contains(ansi.Strip(got.formatCreateView()), "Edit game format") {
		t.Fatalf("format edit state = index=%d form=%v view=%q", got.formatEditIndex, got.form != nil, ansi.Strip(got.formatCreateView()))
	}
}

func TestFormatListShowsFittingChipDenominations(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.formatIndex = 0
	model.table = &api.TableDetail{
		Table: api.TableSummary{ID: "table-1"},
		Formats: []api.GameFormat{{
			Name:          "rookie 2k",
			RequiredEntry: 2000,
			Chips: []api.ChipDenomination{
				{Color: "white", Value: 200},
				{Color: "black", Value: 100},
				{Color: "green", Value: 50},
				{Color: "blue", Value: 20},
			},
		}},
	}

	view := ansi.Strip(model.formatList(100))
	for _, value := range []string{"200", "100", "50", "20"} {
		if !strings.Contains(view, value) {
			t.Fatalf("format list omitted fitting chip value %q:\n%s", value, view)
		}
	}
}

func TestVerticalChipCountersMoveAndAdjust(t *testing.T) {
	t.Parallel()
	values := []string{"0", "3"}
	row := newVerticalChipCounters([]api.ChipDenomination{
		{Label: "white", Color: "white", Value: 10},
		{Label: "red", Color: "red", Value: 50},
	}, &values)
	row.Focus()

	_, _ = row.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	if values[0] != "1" {
		t.Fatalf("right arrow count = %q, want 1", values[0])
	}
	_, _ = row.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	_, _ = row.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight, Mod: tea.ModShift}))
	if row.active != 1 || values[1] != "13" {
		t.Fatalf("shift-right result = active %d values %#v, want active 1 and second count 13", row.active, values)
	}
	_, _ = row.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft, Mod: tea.ModShift}))
	if values[1] != "3" {
		t.Fatalf("shift-left count = %q, want 3", values[1])
	}
	view := ansi.Strip(row.View())
	for _, want := range []string{"white 10", "red 50", "- 1 +", "- 3 +"} {
		if !strings.Contains(view, want) {
			t.Fatalf("counter view missing %q:\n%s", want, view)
		}
	}
}

func TestRecordPlayerListShowsSavedValueAndPL(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.recordFormatIndex = 0
	model.recordEntered = map[string]bool{"player-1": true, "player-2": true}
	model.recordCounts = map[string]map[string]int{"player-1": {"chip-1": 2}, "player-2": {}}
	model.table = &api.TableDetail{
		Table:   api.TableSummary{HostUsername: "bluff"},
		Players: []api.TablePlayer{{ID: "player-1", Name: "alice"}, {ID: "player-2", Name: "bob"}},
		Formats: []api.GameFormat{{
			RequiredEntry: 200,
			Chips:         []api.ChipDenomination{{ID: "chip-1", Value: 10}},
		}},
	}

	view := ansi.Strip(model.recordPlayerList(100))
	if !strings.Contains(view, "Add/Edit earnings") {
		t.Fatalf("record player list does not use the shared add/edit heading:\n%s", view)
	}
	if !strings.Contains(view, "Total 20 cr") || !strings.Contains(view, "P/L -180 cr") {
		t.Fatalf("recorded player summary missing from list:\n%s", view)
	}
	if strings.Contains(view, "saved") {
		t.Fatalf("recorded player still uses saved marker:\n%s", view)
	}
	if !strings.Contains(view, "bob") || !strings.Contains(view, "ALL IN") || strings.Count(view, "Total") != 1 || strings.Count(view, "P/L") != 1 {
		t.Fatalf("all-in player does not show only the ALL IN badge:\n%s", view)
	}
}

func TestRecorderPlayerSearchFiltersNamesAndUsernames(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.recordPhase, model.recordFormatIndex = recordGameScreen, recordPlayersPhase, 0
	model.table = &api.TableDetail{
		CanManage: true,
		Players: []api.TablePlayer{
			{ID: "player-1", Name: "alice"},
			{ID: "player-2", Name: "robert", Username: "bob"},
		},
		Formats: []api.GameFormat{{Name: "rookie", RequiredEntry: 200}},
	}

	if !model.isSearchableScreen() {
		t.Fatal("editable recorder is not searchable")
	}
	actions := recordActionItemsText(model.recordActionItems())
	if !strings.Contains(actions, "/ Search") {
		t.Fatalf("recorder action bar has no player search: %s", actions)
	}
	model.searchQuery = "bob"
	view := ansi.Strip(model.recordPlayerList(100))
	if !strings.Contains(view, "@bob") || strings.Contains(view, "alice") {
		t.Fatalf("recorder search did not filter by username:\n%s", view)
	}
}

func TestRecordBalanceStripShowsBalancedBadgeOrDeficit(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.recordFormatIndex = 0
	model.recordEntered = map[string]bool{"player-1": true, "player-2": true, "ignored": false}
	model.recordCounts = map[string]map[string]int{
		"player-1": {"chip-1": 2},
		"player-2": {"chip-1": 2},
	}
	model.table = &api.TableDetail{Formats: []api.GameFormat{{
		Name:          "rookie 2k",
		RequiredEntry: 200,
		Chips:         []api.ChipDenomination{{ID: "chip-1", Value: 100}},
	}}}

	balanced := ansi.Strip(model.recordBalanceStrip(100))
	for _, want := range []string{"FORMAT", "rookie 2k", "PLAYERS", "2", "TABLE VALUE", "400 cr", "RECORDED VALUE", "BALANCED"} {
		if !strings.Contains(balanced, want) {
			t.Fatalf("balanced strip missing %q:\n%s", want, balanced)
		}
	}

	model.recordCounts["player-2"]["chip-1"] = 1
	deficit := ansi.Strip(model.recordBalanceStrip(100))
	if !strings.Contains(deficit, "+100 cr") || strings.Contains(deficit, "BALANCED") {
		t.Fatalf("deficit strip does not show the outstanding value:\n%s", deficit)
	}
}

func TestGamePreviewEntersLockedReviewWithRecordedPlayersOnly(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.recordPhase = recordGameScreen, recordPlayersPhase
	model.recordFormatIndex = 0
	model.recordEntered = map[string]bool{"player-1": true, "player-2": true}
	model.recordCounts = map[string]map[string]int{"player-1": {"chip-1": 2}, "player-2": {}}
	model.table = &api.TableDetail{
		CanManage: true,
		Table:     api.TableSummary{ID: "table-1", Name: "#saturday", HostUsername: "bluff"},
		Players: []api.TablePlayer{
			{ID: "player-2", Name: "bob"},
			{ID: "player-1", Name: "alice"},
			{ID: "player-3", Name: "carol"},
		},
		Formats: []api.GameFormat{{ID: "format-1", RequiredEntry: 200, Chips: []api.ChipDenomination{{ID: "chip-1", Value: 100}}}},
	}

	updated, _ := model.Update(tableGamePreviewedMsg{game: api.TableGame{ExpectedTableValue: 400, ActualTableValue: 400}})
	review := updated.(Model)
	if review.recordPhase != recordReviewPhase || review.recordPreview == nil {
		t.Fatalf("preview phase = %v preview=%v; want locked review", review.recordPhase, review.recordPreview != nil)
	}
	view := ansi.Strip(review.recordReviewPlayerList(100))
	for _, want := range []string{"Review", "alice", "bob", "ALL IN"} {
		if !strings.Contains(view, want) {
			t.Fatalf("review list missing %q:\n%s", want, view)
		}
	}
	if !strings.Contains(view, "× 2") {
		t.Fatalf("review list does not show the recorded denomination:\n%s", view)
	}
	if !lineContainsAll(view, "Total 200 cr", "P/L 0 cr", "× 2") || strings.Contains(view, "100 ×") {
		t.Fatalf("review totals, P/L, and denominations are not on the same line:\n%s", view)
	}
	if strings.Contains(view, "carol") || strings.Contains(view, "Add earnings") {
		t.Fatalf("review list includes editable or unrecorded content:\n%s", view)
	}
	if strings.Index(view, "alice") > strings.Index(view, "bob") {
		t.Fatalf("review players are not sorted by P/L descending:\n%s", view)
	}
	moved, _, handled := review.updateTableKey("down")
	review = moved.(Model)
	if !handled || review.resultIndex != 1 {
		t.Fatalf("review selector did not move: handled=%v index=%d", handled, review.resultIndex)
	}
	actions := recordActionItemsText(review.recordActionItems())
	for _, want := range []string{"s Submit", "e Edit", "esc Back"} {
		if !strings.Contains(actions, want) {
			t.Fatalf("review actions missing %q: %s", want, actions)
		}
	}
	for _, blocked := range []string{"Add player", "Edit date/note"} {
		if strings.Contains(actions, blocked) {
			t.Fatalf("review actions still contain %q: %s", blocked, actions)
		}
	}

	edited, _, handled := review.updateTableKey("e")
	if !handled || edited.(Model).recordPhase != recordPlayersPhase || edited.(Model).recordPreview != nil {
		t.Fatalf("edit review did not return to editable earnings")
	}
	for _, key := range []string{"t"} {
		blocked, _, _ := review.updateTableKey(key)
		if blocked.(Model).recordPopup != recordPopupNone {
			t.Fatalf("review key %q opened a metadata popup", key)
		}
	}
	submitted, _ := review.Update(tableGameRecordedMsg{table: *review.table})
	if submitted.(Model).screen != tableDetailScreen || submitted.(Model).recordPreview != nil {
		t.Fatalf("successful submission did not return to table overview")
	}
}

func recordActionItemsText(items []actionBarItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item.key+" "+item.label)
	}
	return strings.Join(parts, " | ")
}

func TestEarningsSummaryRowsAlignLabelsAndAmountsRight(t *testing.T) {
	t.Parallel()
	rows := []string{
		ansi.Strip(earningsSummaryRow("Final value", "600 cr", valueStyle)),
		ansi.Strip(earningsSummaryRow("Buy-in", "2000 cr", mutedStyle)),
		ansi.Strip(earningsSummaryRow("P/L", "-1400 cr", standingStyle(-1400))),
	}
	for _, row := range rows {
		if ansi.StringWidth(row) != 26 {
			t.Fatalf("summary row width = %d, want 26: %q", ansi.StringWidth(row), row)
		}
	}
	if !strings.HasSuffix(rows[0], "      600 cr") || !strings.HasSuffix(rows[1], "     2000 cr") || !strings.HasSuffix(rows[2], "    -1400 cr") {
		t.Fatalf("summary amounts are not right aligned: %#v", rows)
	}
}

func TestTableWorkspaceOverviewUsesChartsAndLocalNavigation(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 120, 40
	model.screen, model.loading = tableDetailScreen, false
	model.table = &api.TableDetail{
		Table:   api.TableSummary{Name: "#saturday-table", HostUsername: "bluff"},
		Players: []api.TablePlayer{{ID: "p1", Name: "Alice", Standing: 120}, {ID: "p2", Name: "Bob", Standing: -40}},
		Games: []api.TableGame{{Date: "2026-08-09", Participants: []api.TableGameParticipant{
			{PlayerID: "p1", EndingStanding: 120}, {PlayerID: "p2", EndingStanding: -40},
		}}},
	}
	view := ansi.Strip(model.View().Content)
	statsWidth := 0
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "HOST") {
			statsWidth = ansi.StringWidth(line)
			break
		}
	}
	if statsWidth != model.width {
		t.Fatalf("overview stats width = %d, want full width %d", statsWidth, model.width)
	}
	for _, want := range []string{"PLAYERS", "FORMATS", "GAMES", "History", "Standings"} {
		if !strings.Contains(view, want) {
			t.Fatalf("overview is missing %q:\n%s", want, view)
		}
	}
	if strings.Index(view, "Standings") > strings.Index(view, "History") {
		t.Fatalf("standings should appear before history:\n%s", view)
	}
	if strings.Contains(view, "Recent activity") {
		t.Fatal("overview still contains recent activity")
	}
	for _, item := range tableDetailActionItems(false) {
		if item.action == "players" || item.action == "formats" || item.action == "games" {
			t.Fatalf("table action bar still contains navigation action %q", item.action)
		}
	}
}

func TestTableWorkspaceOverviewShowsOneEmptyStateBeforeFirstGame(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 120, 40
	model.screen, model.loading = tableDetailScreen, false
	model.table = &api.TableDetail{
		Table:   api.TableSummary{Name: "#saturday", HostUsername: "bluff"},
		Players: []api.TablePlayer{{ID: "p1", Name: "Alice"}},
	}
	view := ansi.Strip(model.View().Content)
	if !strings.Contains(view, "Record a game") || !strings.Contains(view, "Stats will appear here") {
		t.Fatalf("overview empty state is missing:\n%s", view)
	}
	if strings.Contains(view, "History") || strings.Contains(view, "Standings") {
		t.Fatalf("overview renders charts before the first game:\n%s", view)
	}
}

func TestOverviewChartExpansionShortcuts(t *testing.T) {
	t.Parallel()
	base := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	base.width, base.height = 120, 40
	base.screen, base.loading = tableDetailScreen, false
	base.table = &api.TableDetail{
		Table: api.TableSummary{Name: "#saturday", HostUsername: "bluff"},
		Players: []api.TablePlayer{
			{ID: "p1", Name: "bluff", Standing: 100},
			{ID: "p2", Name: "alice", Standing: -100},
		},
		Games: []api.TableGame{{Date: "2026-08-09", Participants: []api.TableGameParticipant{
			{PlayerID: "p1", EndingStanding: 100},
			{PlayerID: "p2", EndingStanding: -100},
		}}},
	}

	normal := ansi.Strip(base.View().Content)
	for _, shortcut := range []string{"alt+h expand", "alt+s expand"} {
		if !strings.Contains(normal, shortcut) {
			t.Fatalf("overview is missing %q:\n%s", shortcut, normal)
		}
	}

	tests := []struct {
		name       string
		key        string
		focus      tableChartFocus
		visible    string
		notVisible string
	}{
		{name: "history", key: "alt+h", focus: tableChartHistory, visible: "History", notVisible: "Standings"},
		{name: "standings", key: "alt+s", focus: tableChartStandings, visible: "Standings", notVisible: "History"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			updated, _, handled := base.updateTableKey(test.key)
			if !handled {
				t.Fatalf("%s was not handled", test.key)
			}
			expanded := updated.(Model)
			if expanded.expandedChart != test.focus {
				t.Fatalf("expanded chart = %d, want %d", expanded.expandedChart, test.focus)
			}
			view := ansi.Strip(expanded.View().Content)
			if !strings.Contains(view, test.visible) || !strings.Contains(view, "esc back") {
				t.Fatalf("expanded chart is incomplete:\n%s", view)
			}
			if strings.Contains(view, test.notVisible) || strings.Contains(view, "HOST") {
				t.Fatalf("expanded chart still renders overview content:\n%s", view)
			}

			closed, _, handled := expanded.updateTableKey("esc")
			if !handled || closed.(Model).expandedChart != tableChartNone || closed.(Model).screen != tableDetailScreen {
				t.Fatalf("escape did not return to the table overview")
			}
		})
	}
}

func TestTableOverviewStatsExcludeBalancedTableValue(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{
		Table:   api.TableSummary{HostUsername: "bluff"},
		Players: []api.TablePlayer{{Standing: 100}, {Standing: -100}},
		Games:   []api.TableGame{{}},
	}

	view := ansi.Strip(model.tableOverviewStats(120))
	for _, want := range []string{"HOST", "PLAYERS", "GAMES"} {
		if !strings.Contains(view, want) {
			t.Fatalf("overview stats are missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "TABLE VALUE") {
		t.Fatalf("overview still shows the balanced table value:\n%s", view)
	}
}

func TestExpandedChipChartAddsRowsBetweenPlayers(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{
		Table: api.TableSummary{HostUsername: "bluff"},
		Players: []api.TablePlayer{
			{ID: "p1", Name: "alice", Standing: 100},
			{ID: "p2", Name: "bob", Standing: -100},
		},
	}

	compact := ansi.Strip(model.tableChipChartWithShortcut(100, "alt+s expand"))
	model.expandedChart = tableChartStandings
	expanded := ansi.Strip(model.tableChipChartWithShortcut(100, "esc back"))
	if strings.Count(expanded, "\n\n") != strings.Count(compact, "\n\n")+1 {
		t.Fatalf("expanded chip chart did not add one row between players:\n%s", expanded)
	}
}

func TestExpandedStandingChartUsesRemainingTerminalHeight(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.height = 60
	if got, want := model.expandedStandingChartHeight(), 51; got != want {
		t.Fatalf("expanded chart height = %d, want %d", got, want)
	}

	model.height = 12
	if got, want := model.expandedStandingChartHeight(), 8; got != want {
		t.Fatalf("small-terminal chart height = %d, want %d", got, want)
	}
}

func TestTableChipChartSortsHighestValueFirst(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{
		Table: api.TableSummary{HostUserID: "host-middle", HostUsername: "middle"},
		Players: []api.TablePlayer{
			{Name: "middle", UserID: "host-middle", Username: "middle", Standing: 100},
			{Name: "lowest", Standing: -900},
			{Name: "highest", Standing: 2500},
		},
	}

	view := ansi.Strip(model.tableChipChart(100))
	highest := strings.Index(view, "highest")
	middle := strings.Index(view, "@middle")
	lowest := strings.Index(view, "lowest")
	if highest < 0 || middle < 0 || lowest < 0 || !(highest < middle && middle < lowest) {
		t.Fatalf("chip chart is not sorted by value descending:\n%s", view)
	}
	lines := strings.Split(view, "\n")
	if !strings.HasPrefix(lines[0], "/// Standings ") || strings.TrimSpace(lines[1]) != "" {
		t.Fatalf("chip chart heading is not followed by one blank row:\n%s", view)
	}
}

func TestStandingsChartShowsLifetimeLedgerColumns(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{
		Table:   api.TableSummary{HostUserID: "host-alice", HostUsername: "alice"},
		Players: []api.TablePlayer{{ID: "p1", Name: "alice", UserID: "host-alice", Username: "alice", Standing: 500}},
		Games: []api.TableGame{
			{Participants: []api.TableGameParticipant{{PlayerID: "p1", RequiredEntry: 2000, FinalValue: 2500}}},
			{Participants: []api.TableGameParticipant{{PlayerID: "p1", RequiredEntry: 2000, FinalValue: 2000}}},
		},
	}

	summary := model.tablePlayerLedgerSummaries()["p1"]
	if summary.withdrawn != 4500 || summary.buyIn != 4000 || summary.games != 2 {
		t.Fatalf("ledger summary = %#v, want withdrawn 4500, buy-in 4000, games 2", summary)
	}
	view := ansi.Strip(model.tableChipChart(120))
	for _, want := range []string{"BALANCE", "CHANGE", "GAMES"} {
		if !strings.Contains(view, want) {
			t.Fatalf("standings chart is missing %q:\n%s", want, view)
		}
	}
}

func TestTableStandingChartPlotsGamesWithPlayerLegend(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{
		Table: api.TableSummary{HostUserID: "host-alice", HostUsername: "alice"},
		Players: []api.TablePlayer{
			{ID: "player-1", Name: "alice", UserID: "host-alice", Username: "alice"},
			{ID: "player-2", Name: "bob"},
		},
		Games: []api.TableGame{
			{Date: "2026-08-08", Participants: []api.TableGameParticipant{{PlayerID: "player-1", EndingStanding: 100}, {PlayerID: "player-2", EndingStanding: -100}}},
			{Date: "2026-08-09", Participants: []api.TableGameParticipant{{PlayerID: "player-1", EndingStanding: 40}, {PlayerID: "player-2", EndingStanding: -40}}},
		},
	}

	view := ansi.Strip(model.tableStandingChart(100))
	for _, want := range []string{"History", "100 cr", "-100 cr", "0", "◆ @alice", "◆ bob", "•"} {
		if !strings.Contains(view, want) {
			t.Fatalf("standing chart missing %q:\n%s", want, view)
		}
	}
	if strings.Count(view, "\n") < 16 {
		t.Fatalf("standing chart is not using the taller plot area:\n%s", view)
	}
	if strings.Contains(view, "2026-08") || strings.Contains(view, "date-wise") {
		t.Fatalf("standing chart still exposes dates:\n%s", view)
	}
	lines := strings.Split(view, "\n")
	if !strings.HasPrefix(lines[0], "/// History ") || strings.TrimSpace(lines[1]) != "" {
		t.Fatalf("standing chart heading is not followed by one blank row:\n%s", view)
	}
}

func TestTableSubpageTablesStayInsideTerminalWidth(t *testing.T) {
	t.Parallel()
	base := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	base.width, base.height, base.loading = 120, 36, false
	base.table = &api.TableDetail{
		Table: api.TableSummary{Name: "#saturday", HostUsername: "bluff"},
		Players: []api.TablePlayer{
			{ID: "player-1", Name: "alice"},
			{ID: "player-2", Name: "bob"},
		},
		Formats: []api.GameFormat{{ID: "format-1", Name: "rookie", RequiredEntry: 2000, Chips: []api.ChipDenomination{{ID: "chip-1", Color: "white", Value: 200}}}},
	}
	for _, screen := range []screen{playersScreen, formatsScreen} {
		model := base
		model.screen = screen
		view := model.View().Content
		for lineNumber, line := range strings.Split(view, "\n") {
			if lineWidth := ansi.StringWidth(line); lineWidth > model.width {
				t.Fatalf("screen %v line %d width = %d, exceeds terminal width %d:\n%s", screen, lineNumber, lineWidth, model.width, ansi.Strip(view))
			}
		}
	}
}

func TestEmptyStatesReserveOneRowBelowActionBars(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{}
	for name, view := range map[string]string{
		"tables":  model.tableSummaryList(100),
		"users":   model.userList(100),
		"players": model.playerList(100),
		"formats": model.formatList(100),
		"games":   model.gameList(100),
	} {
		if !strings.HasPrefix(ansi.Strip(view), "\n") {
			t.Fatalf("%s empty state has no spacer row: %q", name, ansi.Strip(view))
		}
	}
}

func TestTableCreatedMessageLeavesCompletedFormBeforeOpening(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = tableCreateScreen, true
	updated, _ := model.Update(tableCreatedMsg{table: api.TableSummary{ID: "table-1", Name: "#saturday-table"}})
	got := updated.(Model)
	if got.screen != tableDetailScreen {
		t.Fatalf("screen after table creation = %v, want table detail", got.screen)
	}
	if !got.loading {
		t.Fatal("new table should remain loading while it opens")
	}
}

func TestTableLoadedMessageOpensDetailFromTableList(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = tablesScreen, true
	updated, _ := model.Update(tableLoadedMsg{table: api.TableDetail{Table: api.TableSummary{ID: "table-1", Name: "#saturday-table"}}})
	got := updated.(Model)
	if got.screen != tableDetailScreen {
		t.Fatalf("screen after opening table = %v, want table detail", got.screen)
	}
	if got.loading || got.table == nil {
		t.Fatal("opened table did not finish loading")
	}
}

func TestTableGameMouseRegionsSelectHistoryRows(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 100, 36
	model.screen, model.loading = gamesScreen, false
	model.table = &api.TableDetail{Table: api.TableSummary{Name: "#Saturday table"}, Games: []api.TableGame{{ID: "g1"}, {ID: "g2"}}}
	regions := model.tableHitRegions()
	found := false
	for _, region := range regions {
		if strings.HasPrefix(region.value, "game:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("game history did not expose mouse hit regions")
	}
}

func TestGameHistoryMovementFollowsNewestFirstRows(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = gamesScreen, false
	model.table = &api.TableDetail{Games: []api.TableGame{
		{ID: "oldest"}, {ID: "middle"}, {ID: "newest"},
	}}

	tests := []struct {
		name  string
		start int
		key   string
		want  int
	}{
		{name: "down selects the next older game", start: 2, key: "down", want: 1},
		{name: "up selects the next newer game", start: 1, key: "up", want: 2},
		{name: "j selects the next older game", start: 1, key: "j", want: 0},
		{name: "k selects the next newer game", start: 1, key: "k", want: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := model
			current.gameIndex = test.start
			updated, _, handled := current.updateTableKey(test.key)
			got := updated.(Model)
			if !handled || got.gameIndex != test.want {
				t.Fatalf("handled=%v game index=%d, want %d", handled, got.gameIndex, test.want)
			}
		})
	}

	model.gameIndex = 2
	updated, _ := model.Update(tableScrollMsg{delta: 3})
	if got := updated.(Model).gameIndex; got != 0 {
		t.Fatalf("game index after scrolling down = %d, want 0", got)
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

func TestSlashSearchFiltersUsersAndEscClearsIt(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{users: []api.User{{Username: "bluff", Role: "admin"}, {Username: "dealer", Role: "member"}}}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = usersScreen, false
	model.users = []api.User{{Username: "bluff", Role: "admin"}, {Username: "dealer", Role: "member"}}
	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: '/', Text: "/"}))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))
	model = updated.(Model)
	list := model.userList(80)
	if !model.searchActive || model.searchQuery != "dea" || !strings.Contains(list, "@dealer") || strings.Contains(list, "@bluff") {
		t.Fatalf("search state/view = active=%v query=%q view=%q", model.searchActive, model.searchQuery, list)
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	cleared := updated.(Model)
	if cleared.searchActive || cleared.searchQuery != "" {
		t.Fatalf("search did not clear on escape: active=%v query=%q", cleared.searchActive, cleared.searchQuery)
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

func TestPublicTableNameRequiresHashPrefix(t *testing.T) {
	t.Parallel()
	if err := publicTableName("Saturday-table"); err != nil {
		t.Fatalf("valid public table name rejected: %v", err)
	}
	if canonical := canonicalTableName("Saturday-TABLE"); canonical != "#saturday-table" {
		t.Fatalf("canonical table name = %q, want #saturday-table", canonical)
	}
	if err := publicTableName("Saturday table"); err == nil {
		t.Fatal("table name with spaces accepted")
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

func TestCenteredInputRendersVisualTablePrefix(t *testing.T) {
	t.Parallel()
	value := ""
	input := newCenteredInput("Table name", "", "saturday-table", &value, 48, false, publicTableName).WithPrefix("#")
	input.WithTheme(huh.ThemeFunc(centeredFormTheme))
	input.WithKeyMap(huh.NewDefaultKeyMap())
	input.WithWidth(40)
	input.Focus()

	if view := ansi.Strip(input.View()); !strings.Contains(view, "#saturday-table") {
		t.Fatalf("table placeholder is missing visual prefix: %q", view)
	}
	for _, character := range "friday-table" {
		_, _ = input.Update(tea.KeyPressMsg(tea.Key{Code: character, Text: string(character)}))
	}
	if view := ansi.Strip(input.View()); !strings.Contains(view, "#friday-table") {
		t.Fatalf("table value is missing visual prefix: %q", view)
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

func lineContainsAll(view string, values ...string) bool {
	for _, line := range strings.Split(view, "\n") {
		matched := true
		for _, value := range values {
			if !strings.Contains(line, value) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
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

func TestStandingInsights(t *testing.T) {
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.table = &api.TableDetail{Players: []api.TablePlayer{{ID: "a", Standing: 80}, {ID: "b", Standing: 0}}, Games: []api.TableGame{
		{Date: "2026-01-01", Participants: []api.TableGameParticipant{{PlayerID: "a", RequiredEntry: 100, FinalValue: 150}}},
		{Date: "2026-01-02", Participants: []api.TableGameParticipant{{PlayerID: "a", RequiredEntry: 100, FinalValue: 130}}},
		{Date: "2026-01-03", Participants: []api.TableGameParticipant{{PlayerID: "b", RequiredEntry: 100, FinalValue: 100}}},
	}}
	stats := model.tablePlayerLedgerSummaries()
	if a := stats["a"]; a.change != 0 || a.games != 2 || a.wins != 2 || a.streak != 2 {
		t.Fatalf("skipped game insights: %+v", a)
	}
	if stats["b"].streak != 0 || stats["b"].wins != 0 {
		t.Fatal("break-even counted as a win")
	}
	model.screen = tableDetailScreen
	model.expandedChart = tableChartStandings
	model.standingEntryOffset = 1
	view := ansi.Strip(model.tableChipChartWithShortcut(140, model.standingEntryShortcuts()))
	for _, want := range []string{"CHANGE", "AVERAGE", "WIN %", "STREAK", "+30 cr", "+40 cr", "100%", "2W"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
	if len(model.table.Games) != 3 || model.table.Players[0].Standing != 80 {
		t.Fatal("render changed source data")
	}
	ranks := standingRanks([]api.TablePlayer{{ID: "a", Standing: 10}, {ID: "b", Standing: 10}, {ID: "c", Standing: 0}})
	if ranks["a"] != 1 || ranks["b"] != 1 || ranks["c"] != 3 {
		t.Fatalf("tie ranks: %v", ranks)
	}
}
