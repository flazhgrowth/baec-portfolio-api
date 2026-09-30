package accountsvc

import (
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/config"
	"github.com/flazhgrowth/fg-tamagochi/pkg/vault"
	"github.com/flazhgrowth/fg-tamagopkg/logger"
)

var (
	baselogpath  logger.LogPath = "accountsvc"
	getJwtSecret                = func() string {
		return vault.GetVault().GetStringWithDefault("secret.jwt", "")
	}
	getJwtTTL = func() time.Time {
		return time.Now().Add(time.Duration(config.GetConfig().GetIntWithDefault("", 30)) * time.Minute)
	}
	getJwtRefreshTTL = func() time.Time {
		return time.Now().Add(time.Duration(config.GetConfig().GetIntWithDefault("", 7)*24) * time.Hour)
	}
)

type service struct {
	accountRepo account.Repository
}

func New(accountrepo account.Repository) account.Service {
	return &service{
		accountRepo: accountrepo,
	}
}
