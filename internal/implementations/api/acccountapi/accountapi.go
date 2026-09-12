package accountapi

import "github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"

type api struct {
	accountSvc account.Service
}

func New(accountsvc account.Service) account.API {
	return &api{
		accountSvc: accountsvc,
	}
}
	