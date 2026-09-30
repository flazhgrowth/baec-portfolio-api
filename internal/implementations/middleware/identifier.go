package middleware

import "github.com/flazhgrowth/fg-tamagochi/pkg/http/middleware"

var (
	MIDDLEWARE_CMS_KEY middleware.HTTPMiddleware = "cms"
	MIDDLEWARE_AUTH    middleware.HTTPMiddleware = "app_auth"
)
