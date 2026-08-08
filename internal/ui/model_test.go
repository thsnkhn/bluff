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
func (f fakeAPI) PreviewTableGame(context.Context, string, string, string, string, string, []api.GameParticipantInput) (api.TableGame, error) {
	return api.TableGame{}, errors.New("not implemented")
}
func (f fakeAPI) RecordTableGame(context.Context, string, string, string, string, string, []api.GameParticipantInput) (api.TableDetail, error) {
	return api.TableDetail{}, errors.New("not implemented")
}
func (f fakeAPI) Logout(context.Context, string) error { return nil }

type fakeStore struct {
	token string
	err   error
}

type fakeInstaller struct {
	release api.ClientRelease
}

func (f *fakeInstaller) Install(_ context.Context, release api.ClientRelease) error {
	f.release = release
	return nil
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
	model.recordSelected = map[string]bool{}
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

func TestTableWorkspaceOverviewUsesChartsAndLocalNavigation(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.width, model.height = 120, 40
	model.screen, model.loading = tableDetailScreen, false
	model.table = &api.TableDetail{
		Table:   api.TableSummary{Name: "#saturday-table", HostUsername: "bluff"},
		Players: []api.TablePlayer{{Name: "Alice", Standing: 120}, {Name: "Bob", Standing: -40}},
	}
	view := ansi.Strip(model.View().Content)
	for _, want := range []string{"PLAYERS", "FORMATS", "GAMES", "Player standings", "Chip values"} {
		if !strings.Contains(view, want) {
			t.Fatalf("overview is missing %q:\n%s", want, view)
		}
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
