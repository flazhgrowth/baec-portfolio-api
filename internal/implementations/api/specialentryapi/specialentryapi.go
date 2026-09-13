package specialentryapi

import "github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"

type api struct {
	specialentrySvc specialentry.SpecialEntryService
}

func New(specialentrysvc specialentry.SpecialEntryService) specialentry.SpecialEntryAPI {
	return &api{
		specialentrySvc: specialentrysvc,
	}
}
