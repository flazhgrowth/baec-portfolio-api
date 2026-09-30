package accountapi

import (
	"net/http"

	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/contractresp"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

// Logout is deliberately a no-op. Account tokens are stateless JWTs, so there is
// nothing on the server to invalidate: the client logs out by discarding its
// token, and the token keeps verifying until it expires. The contract makes
// logout idempotent and never an error, so it is public and always answers 204,
// whatever token (or none) came with it.
func (api *api) Logout(req request.Request, resp response.Response) {
	contractresp.NoContent(resp, http.StatusNoContent)
}
