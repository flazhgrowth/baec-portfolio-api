package sudokugameapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

// pingInterval keeps proxies from closing an idle stream.
const pingInterval = 10 * time.Second

// StreamEvents holds the connection open and pushes every change to the game as
// an SSE `update` message. Errors before the first byte use the normal JSON
// error response; once streaming, the only way out is the client leaving or the
// subscriber being dropped.
func (api *api) StreamEvents(req request.Request, resp response.Response) {
	args := sudokugame.StreamEventsRequest{}
	if err := req.DecodeURLParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}
	if err := req.DecodeQueryParam(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	// Only setting up is bounded; the stream itself lives as long as the client.
	setupCtx, cancelSetup := context.WithTimeout(req.GetContext(), 10*time.Second)
	sub, err := api.sudokuGameSvc.Subscribe(setupCtx, args)
	cancelSetup()
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}
	defer sub.Close()

	impl, ok := resp.(*response.ResponseImpl)
	if !ok {
		resp.RespondJSON(nil, apierrors.ErrorInternalServerError())
		return
	}
	controller := http.NewResponseController(impl.ResponseWriter)

	// The server's write timeout is meant for ordinary responses and would cut
	// the stream; lift it for this connection.
	if err = controller.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		resp.RespondJSON(nil, apierrors.ErrorInternalServerError())
		return
	}

	header := impl.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no") // tell nginx not to buffer the stream
	impl.WriteHeader(http.StatusOK)

	send := func(chunk string) bool {
		if _, err := fmt.Fprint(impl, chunk); err != nil {
			return false
		}

		return controller.Flush() == nil
	}

	if !sendUpdate(send, sub.Initial) {
		return
	}

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		select {
		case <-req.GetContext().Done():
			return
		case update, open := <-sub.Updates:
			if !open {
				// Dropped as too slow: end the stream so the client reconnects and resyncs.
				return
			}
			if !sendUpdate(send, update) {
				return
			}
		case <-ping.C:
			if !send(": ping\n\n") {
				return
			}
		}
	}
}

func sendUpdate(send func(string) bool, update sudokugame.GameUpdate) bool {
	data, err := json.Marshal(update)
	if err != nil {
		return false
	}

	return send(fmt.Sprintf("event: update\ndata: %s\n\n", data))
}
