package specialentryroutes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/middleware"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

var (
	tag    = "Special"
	cmsTag = "CMS"
)

func Routes(version router.Router, apis *api.APIs) {
	version.Group("/specials", func(special router.Router) {
		special.Post("/validate", apis.SpecialEntryAPI.Validate, &router.RouterDocs{
			Request:     specialentry.ValidateRequest{},
			Tags:        tag,
			Title:       "Validate Special Room Token",
			Description: "Validate Special Room token. A valid token would permit you to the special room",
		})
	})

	version.Group("/cms", func(cms router.Router) {
		cms.Use(middleware.MIDDLEWARE_CMS_KEY)
		cms.Post("/specials/validate", apis.SpecialEntryAPI.CMSRegister, &router.RouterDocs{
			Security:    router.SecAuths{router.SecurityAPIKey},
			Request:     specialentry.CMSRegisterRequest{},
			Response:    specialentry.CMSRegisterResponse{},
			Tags:        cmsTag,
			Title:       "[CMS] Register New Special Entry Code",
			Description: "[CMS] Register new Special Entry code",
		})
	})
}
