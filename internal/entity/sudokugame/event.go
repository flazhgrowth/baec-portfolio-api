package sudokugame

const (
	EventMove              = "move"
	EventTurnExpired       = "turn_expired"
	EventTurnSkipped       = "turn_skipped"
	EventFaultLimitReached = "fault_limit_reached"
	EventPlayerJoined      = "player_joined"
	EventPlayerForfeited   = "player_forfeited"
	EventGameCompleted     = "game_completed"

	ResultCorrect   = "correct"
	ResultIncorrect = "incorrect"
)

type (
	// GameEvent describes one thing that just happened. It is a tagged union on
	// Type in the API, so each event only carries the fields that type defines.
	// Row/Col/Value/Points are pointers because 0 is a real value for them.
	// (player_connection and player_forfeited arrive with presence and forfeit.)
	GameEvent struct {
		Type      string `json:"type"`
		PlayerID  string `json:"player_id,omitempty"`
		Row       *int   `json:"row,omitempty"`
		Col       *int   `json:"col,omitempty"`
		Value     *int   `json:"value,omitempty"`
		Result    string `json:"result,omitempty"`
		Points    *int   `json:"points,omitempty"`
		SkipTurns *int   `json:"skip_turns,omitempty"`
	}

	// GameUpdate is the data of every SSE `update` message, and the body of
	// turn/expire and forfeit responses.
	GameUpdate struct {
		Game   GameResponse `json:"game"`
		Events []GameEvent  `json:"events"`
	}

	// StreamEventsRequest: GET /games/{gameId}/events?token=<player token>.
	StreamEventsRequest struct {
		GameID string `path:"gameId" pathtype:"string" query:"-" json:"-"`
		Token  string `query:"token" json:"-"`
	}

	// Subscription is a live feed of one game. Initial is the full current state
	// (with no events). Updates closes when the subscriber is dropped for being
	// too slow; the client is expected to reconnect and is resynced by Initial.
	Subscription struct {
		Initial GameUpdate
		Updates <-chan GameUpdate
		Close   func()
	}
)

// NewGameUpdate builds an update, guaranteeing events serialises as [] not null.
func NewGameUpdate(game GameResponse, events ...GameEvent) GameUpdate {
	if events == nil {
		events = []GameEvent{}
	}

	return GameUpdate{Game: game, Events: events}
}
