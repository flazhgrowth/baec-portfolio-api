package specialentrysvc

import (
	"context"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/specialentry"
	"github.com/flazhgrowth/baec-portfolio-api/internal/pkg/random"
)

func (svc *service) CMSRegister(ctx context.Context, args specialentry.CMSRegisterRequest) (datum *specialentry.CMSRegisterResponse, err error) {
	logpath := baselogpath.With("CMSRegister")

	specialEntryData := &specialentry.SpecialEntry{
		Code:       random.GenerateRandomString(16),
		MaxEntries: args.MaxEntry,
		EntryCount: 0,
	}
	if err = svc.specialentryRepo.Insert(ctx, specialEntryData); err != nil {
		logpath.With("Insert").LogError(ctx, "failed on inserting data to special_entry", err)
		return nil, err
	}

	return &specialentry.CMSRegisterResponse{
		Token: specialEntryData.Code,
	}, nil
}
