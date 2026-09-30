package routes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes/accountroutes"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes/guestroutes"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes/msgroutes"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes/noteroutes"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes/specialentryroutes"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/routes/sudokugameroutes"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

func v1(version router.Router, api *api.APIs) {
	guestroutes.Routes(version, api)
	specialentryroutes.Routes(version, api)
	msgroutes.Routes(version, api)
	noteroutes.Routes(version, api)
}

func v1Sudous(version router.Router, apis *api.APIs) {
	accountroutes.Routes(version, apis)
	sudokugameroutes.Routes(version, apis)
}
