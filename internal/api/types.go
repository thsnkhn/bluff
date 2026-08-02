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
	Players     []Player   `json:"players"`
	Templates   []Template `json:"templates"`
	CurrentGame *Game      `json:"currentGame"`
	Games       []Game     `json:"games"`
}
