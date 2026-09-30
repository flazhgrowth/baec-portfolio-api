package accountapi

import (
	"context"
	"net/http"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api/contractresp"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) Me(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*2)
	defer cancel()

	accountInfo, err := req.GetAccountInfo()
	if err != nil {
		contractresp.Error(resp, err)
		return
	}

	user, err := api.accountSvc.Me(ctx, accountInfo.ID)
	if err != nil {
		contractresp.Error(resp, err)
		return
	}

	contractresp.JSON(resp, http.StatusOK, user)
}
