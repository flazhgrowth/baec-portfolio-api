package accountsvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

// Token signing reads the vault and config, which tests do not load.
func stubTokens(t *testing.T) {
	t.Helper()
	secret, ttl, refresh := getJwtSecret, getJwtTTL, getJwtRefreshTTL
	getJwtSecret = func() string { return "test-secret-0123456789abcdef0123" }
	getJwtTTL = func() time.Time { return time.Now().Add(30 * time.Minute) }
	getJwtRefreshTTL = func() time.Time { return time.Now().Add(7 * 24 * time.Hour) }
	t.Cleanup(func() { getJwtSecret, getJwtTTL, getJwtRefreshTTL = secret, ttl, refresh })
}

func TestRegisterLowercasesUsername(t *testing.T) {
	stubTokens(t)
	repo := &fakeAccountRepo{}
	svc := New(repo)
	ctx := context.Background()

	registered, err := svc.Register(ctx, account.RegisterRequest{LoginRequest: account.LoginRequest{Username: "Alex", Password: "hunter2222"}})
	if err != nil {
		t.Fatal(err)
	}
	if registered.Username != "alex" || repo.stored[0].Username != "alex" {
		t.Fatalf("the username must be stored and returned lowercase: response %q, stored %q", registered.Username, repo.stored[0].Username)
	}

	t.Run("so the account can log in, whatever case the user types", func(t *testing.T) {
		for _, typed := range []string{"Alex", "ALEX", "alex"} {
			logged, err := svc.Login(ctx, account.LoginRequest{Username: typed, Password: "hunter2222"})
			if err != nil {
				t.Fatalf("login as %q: %v", typed, err)
			}
			if logged.ID != registered.ID || logged.Username != "alex" {
				t.Fatalf("login as %q: %+v", typed, logged)
			}
		}
	})

	t.Run("the wrong password still fails", func(t *testing.T) {
		_, err := svc.Login(ctx, account.LoginRequest{Username: "Alex", Password: "nope"})
		var httpErr apierrors.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 401 {
			t.Fatalf("%v", err)
		}
	})

	t.Run("usernames are unique regardless of case", func(t *testing.T) {
		for _, typed := range []string{"alex", "Alex", "ALEX", "aLeX"} {
			_, err := svc.Register(ctx, account.RegisterRequest{LoginRequest: account.LoginRequest{Username: typed, Password: "hunter2222"}})
			var httpErr apierrors.HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != 409 {
				t.Fatalf("registering %q again must conflict, got %v", typed, err)
			}
		}
		if len(repo.stored) != 1 {
			t.Fatalf("no duplicate may be stored: %d accounts", len(repo.stored))
		}
	})
}
