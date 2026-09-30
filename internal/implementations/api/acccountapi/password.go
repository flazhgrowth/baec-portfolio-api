package accountapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) ChangePassword(req request.Request, resp response.Response) {
	ctx, cancel := context.WithTimeout(req.GetContext(), time.Second*10)
	defer cancel()

	accountInfo, err := req.GetAccountInfo()
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	args := account.ChangePasswordRequest{}
	if err = req.DecodeBody(&args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}
	args.AccountInfo = accountInfo

	if err = args.Validate(); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	if err = api.accountSvc.ChangePassword(ctx, args); err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(nil, nil)
}
