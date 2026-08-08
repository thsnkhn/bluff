package api

// User is the authenticated Bluff account.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Session is returned after a successful login.
type Session struct {
	User      User   `json:"user"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

// HealthStatus is the lightweight service status returned during startup.
type HealthStatus struct {
	Status        string `json:"status"`
	ClientVersion string `json:"clientVersion"`
}

// ClientRelease describes a verified platform archive available to the updater.
type ClientRelease struct {
	Version     string `json:"version"`
	ReleaseURL  string `json:"releaseUrl"`
	AssetName   string `json:"assetName"`
	DownloadURL string `json:"downloadUrl"`
	SHA256      string `json:"sha256"`
}

// Invitation is a single-use code created by an administrator.
type Invitation struct {
	Code      string `json:"code"`
	CreatedAt string `json:"createdAt"`
}

// Player is a member represented in the game ledger.
type Player struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Standing int    `json:"standing"`
}

// Template describes the fixed entry amount for a game.
type Template struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RequiredEntry int    `json:"requiredEntry"`
	Active        bool   `json:"active"`
}

// TableSummary describes a table available to the signed-in account.
type TableSummary struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	HostUserID   string  `json:"hostUserId"`
	HostUsername string  `json:"hostUsername"`
	PlayerCount  int     `json:"playerCount"`
	FormatCount  int     `json:"formatCount"`
	GameCount    int     `json:"gameCount"`
	LastGameDate *string `json:"lastGameDate"`
}

// ChipDenomination describes a configured chip color and its value.
type ChipDenomination struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Color    string `json:"color"`
	Value    int    `json:"value"`
	Position int    `json:"position"`
}

// GameFormat is a table-scoped game format.
type GameFormat struct {
	ID            string             `json:"id"`
	TableID       string             `json:"tableId"`
	Name          string             `json:"name"`
	RequiredEntry int                `json:"requiredEntry"`
	Active        bool               `json:"active"`
	Chips         []ChipDenomination `json:"chips"`
}

// TablePlayer is a table-scoped player profile and standing.
type TablePlayer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	Active   bool   `json:"active"`
	Standing int    `json:"standing"`
}

// ChipCountInput is a player's count for one format denomination.
type ChipCountInput struct {
	DenominationID string `json:"denominationId"`
	Count          int    `json:"count"`
}

// GameParticipantInput identifies a player and their final chip counts.
type GameParticipantInput struct {
	PlayerID   string           `json:"playerId"`
	ChipCounts []ChipCountInput `json:"chipCounts"`
}

// TableGameParticipant contains the locked funding and final result.
type TableGameParticipant struct {
	PlayerID         string      `json:"playerId"`
	PlayerName       string      `json:"playerName"`
	StartingStanding int         `json:"startingStanding"`
	RequiredEntry    int         `json:"requiredEntry"`
	FromBalance      int         `json:"fromBalance"`
	FromBank         int         `json:"fromBank"`
	FinalValue       int         `json:"finalValue"`
	ProfitLoss       int         `json:"profitLoss"`
	EndingStanding   int         `json:"endingStanding"`
	ChipCounts       []ChipCount `json:"chipCounts"`
}

// ChipCount is a persisted chip count with its denomination snapshot.
type ChipCount struct {
	DenominationID string `json:"denominationId"`
	Label          string `json:"label"`
	Color          string `json:"color"`
	Value          int    `json:"value"`
	Count          int    `json:"count"`
}

// TableGame is a completed table game and its recorded results.
type TableGame struct {
	ID                 string                 `json:"id"`
	TableID            string                 `json:"tableId"`
	Date               string                 `json:"date"`
	Format             GameFormat             `json:"format"`
	Status             string                 `json:"status"`
	Participants       []TableGameParticipant `json:"participants"`
	ExpectedTableValue int                    `json:"expectedTableValue"`
	ActualTableValue   int                    `json:"actualTableValue"`
	Version            int                    `json:"version"`
	Remarks            string                 `json:"remarks,omitempty"`
}

// TableDetail is the complete read model for one table.
type TableDetail struct {
	Table     TableSummary  `json:"table"`
	CanManage bool          `json:"canManage"`
	Players   []TablePlayer `json:"players"`
	Formats   []GameFormat  `json:"formats"`
	Games     []TableGame   `json:"games"`
}

// Participant is a player's locked funding in a game.
type Participant struct {
	PlayerID         string `json:"playerId"`
	StartingStanding int    `json:"startingStanding"`
	RequiredEntry    int    `json:"requiredEntry"`
	FromBalance      int    `json:"fromBalance"`
	FromBank         int    `json:"fromBank"`
}

// Game is the dashboard representation of a recorded game.
type Game struct {
	ID                 string        `json:"id"`
	Date               string        `json:"date"`
	Status             string        `json:"status"`
	Template           Template      `json:"template"`
	Participants       []Participant `json:"participants"`
	ExpectedTableValue int           `json:"expectedTableValue"`
	Version            int           `json:"version"`
}

// Bootstrap contains all state required to render the initial dashboard.
type Bootstrap struct {
	Tables      []TableSummary `json:"tables"`
	Players     []Player       `json:"players"`
	Templates   []Template     `json:"templates"`
	CurrentGame *Game          `json:"currentGame"`
	Games       []Game         `json:"games"`
}
