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
	// Token lifetimes come from config.yaml (jwt.access_ttl_minutes, jwt.refresh_ttl_days). When a key
	// is missing the defaults below apply: a 30-minute access token and a 7-day refresh token.
	getJwtTTL = func() time.Time {
		minutes := config.GetConfig().GetIntWithDefault("jwt.access_ttl_minutes", defaultAccessTTLMinutes)
		return time.Now().Add(time.Duration(minutes) * time.Minute)
	}
	getJwtRefreshTTL = func() time.Time {
		days := config.GetConfig().GetIntWithDefault("jwt.refresh_ttl_days", defaultRefreshTTLDays)
		return time.Now().Add(time.Duration(days) * 24 * time.Hour)
	}
)

const (
	defaultAccessTTLMinutes = 30
	defaultRefreshTTLDays   = 7
)

type service struct {
	accountRepo account.Repository
}

func New(accountrepo account.Repository) account.Service {
	return &service{
		accountRepo: accountrepo,
	}
}
