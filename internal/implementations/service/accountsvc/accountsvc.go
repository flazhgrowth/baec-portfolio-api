package accountsvc

import "github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"

type service struct {
	accountRepo account.Repository
}

func New(accountrepo account.Repository) account.Service {
	return &service{
		accountRepo: accountrepo,
	}
}
	