package accountsvc

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

type fakeAccountRepo struct {
	account.Repository
	found  *account.Account
	err    error
	filter account.AccountFilter
}

func (repo *fakeAccountRepo) Get(_ context.Context, filter account.AccountFilter) (*account.Account, error) {
	repo.filter = filter
	if repo.err != nil {
		return nil, repo.err
	}
	if repo.found == nil || !filter.ID.Valid || filter.ID.V != repo.found.ID {
		return nil, sql.ErrNoRows
	}
	return repo.found, nil
}

func TestMe(t *testing.T) {
	created := time.Date(2026, 9, 30, 19, 37, 6, 0, time.FixedZone("WIB", 7*3600))
	repo := &fakeAccountRepo{found: &account.Account{
		BaseULIDModel: entity.BaseULIDModel{ID: "u1", CreatedAt: created},
		Username:      "Alex",
		Password:      "never-returned",
		Salt:          "never-returned",
		Name:          "Alex",
	}}
	svc := New(repo)

	t.Run("returns exactly id, username and created_at, in UTC", func(t *testing.T) {
		user, err := svc.Me(context.Background(), "u1")
		if err != nil {
			t.Fatal(err)
		}
		if user.ID != "u1" || user.Username != "Alex" || !user.CreatedAt.Equal(created) || user.CreatedAt.Location() != time.UTC {
			t.Fatalf("%+v", user)
		}
		if !repo.filter.ID.Valid || repo.filter.ID.V != "u1" || repo.filter.Username.Valid {
			t.Fatalf("must look the account up by id, not username: %+v", repo.filter)
		}
	})

	t.Run("an account that no longer exists is 401 INVALID_TOKEN", func(t *testing.T) {
		_, err := svc.Me(context.Background(), "gone")
		var httpErr apierrors.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 401 || httpErr.Code != "INVALID_TOKEN" {
			t.Fatalf("%v", err)
		}
	})

	t.Run("a database failure is a 500, not a 401", func(t *testing.T) {
		svc := New(&fakeAccountRepo{err: errors.New("db down")})
		_, err := svc.Me(context.Background(), "u1")
		var httpErr apierrors.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 500 {
			t.Fatalf("a flaky database must not log users out: %v", err)
		}
	})
}
