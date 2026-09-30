package sudokugamesvc

import (
	"sync"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
)

// subscriberBuffer is how many updates a stream may fall behind before it is cut.
const subscriberBuffer = 32

// hub fans game updates out to the open event streams of each game.
//
// It lives in memory, so it only reaches streams held by this process: the API
// must run as a single replica until this is backed by something shared
// (e.g. Postgres LISTEN/NOTIFY).
type hub struct {
	mu   sync.Mutex
	subs map[string]map[chan sudokugame.GameUpdate]struct{}
}

func newHub() *hub {
	return &hub{subs: map[string]map[chan sudokugame.GameUpdate]struct{}{}}
}

// subscribe returns a channel of updates for gameID and an idempotent cancel.
// The channel is closed by cancel, or by publish if the subscriber is too slow.
func (h *hub) subscribe(gameID string) (updates <-chan sudokugame.GameUpdate, cancel func()) {
	ch := make(chan sudokugame.GameUpdate, subscriberBuffer)

	h.mu.Lock()
	if h.subs[gameID] == nil {
		h.subs[gameID] = map[chan sudokugame.GameUpdate]struct{}{}
	}
	h.subs[gameID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.remove(gameID, ch)
	}
}

// publish delivers update to every stream of the game without ever blocking.
// A stream whose buffer is full is closed rather than skipped: every update
// carries the full state, so a client that reconnects is resynced, whereas one
// that silently missed an update would keep showing stale state.
func (h *hub) publish(gameID string, update sudokugame.GameUpdate) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.subs[gameID] {
		select {
		case ch <- update:
		default:
			h.remove(gameID, ch)
		}
	}
}

// remove drops ch and closes it. The caller holds h.mu, and it is what makes
// closing safe: sends only happen in publish, also under h.mu.
func (h *hub) remove(gameID string, ch chan sudokugame.GameUpdate) {
	subs, ok := h.subs[gameID]
	if !ok {
		return
	}
	if _, ok = subs[ch]; !ok {
		return
	}

	delete(subs, ch)
	close(ch)
	if len(subs) == 0 {
		delete(h.subs, gameID)
	}
}

func (h *hub) count(gameID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.subs[gameID])
}
