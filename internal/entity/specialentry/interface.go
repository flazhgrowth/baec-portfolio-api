package specialentry

import (
	"context"

	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

type SpecialEntryRepository interface {
	Get(ctx context.Context, filter SpecialEntryFilter) (datum *SpecialEntry, err error)
	Insert(ctx context.Context, datum *SpecialEntry) (err error)
	Update(ctx context.Context, fields SpecialEntryFields, filter SpecialEntryFilter) (err error)
}

type SpecialEntryService interface {
	CMSRegister(ctx context.Context, args CMSRegisterRequest) (datum *CMSRegisterResponse, err error)
	Validate(ctx context.Context, args ValidateRequest) (err error)
}

type SpecialEntryAPI interface {
	CMSRegister(req request.Request, resp response.Response)
	Validate(req request.Request, resp response.Response)
}
