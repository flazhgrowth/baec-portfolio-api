package accountapi

import (
	"context"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

func (api *api) Me(req request.Request, resp response.Response) {
	_, cancel := context.WithTimeout(req.GetContext(), time.Second)
	defer cancel()

	accountInfo, err := req.GetAccountInfo()
	if err != nil {
		resp.RespondJSON(nil, err)
		return
	}

	resp.RespondJSON(account.MeResponse{
		AccountInfo: accountInfo,
	}, nil)
}
