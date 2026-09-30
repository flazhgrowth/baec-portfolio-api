package accountsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/account"
	"github.com/flazhgrowth/fg-tamagochi/pkg/config"
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

func TestRegisterValidation(t *testing.T) {
	stubTokens(t)
	// a repo that fails if it is ever asked anything: validation must run before the database
	repo := &fakeAccountRepo{err: errors.New("the database must not be reached for an invalid request")}
	svc := New(repo)

	for name, req := range map[string]account.LoginRequest{
		"empty body":        {},
		"short username":    {Username: "ab", Password: "hunter2222"},
		"bad characters":    {Username: "no spaces!", Password: "hunter2222"},
		"username too long": {Username: strings.Repeat("a", 21), Password: "hunter2222"},
		"short password":    {Username: "alex", Password: "12345"},
		"password too long": {Username: "alex", Password: strings.Repeat("a", 73)},
		"empty password":    {Username: "alex"},
	} {
		_, err := svc.Register(context.Background(), account.RegisterRequest{LoginRequest: req})
		var httpErr apierrors.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 422 || httpErr.Code != "VALIDATION_ERROR" || httpErr.Message == "" {
			t.Errorf("%s: want 422 VALIDATION_ERROR with a message, got %v", name, err)
		}
	}

	t.Run("a valid request is accepted and mixed case is normalised before validating", func(t *testing.T) {
		repo := &fakeAccountRepo{}
		_, err := New(repo).Register(context.Background(), account.RegisterRequest{LoginRequest: account.LoginRequest{Username: "Mixed_Case_9", Password: "hunter2222"}})
		if err != nil || len(repo.stored) != 1 || repo.stored[0].Username != "mixed_case_9" {
			t.Fatalf("err=%v stored=%+v", err, repo.stored)
		}
	})
}

// loadConfig points the process at a config.yaml with the given body, the way the server finds it.
func loadConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "etc", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "etc", "config", "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if err := config.New(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenLifetimeFromConfig(t *testing.T) {
	within := func(got time.Time, want time.Duration) bool {
		d := time.Until(got) - want
		return d > -time.Minute && d < time.Minute
	}

	t.Run("jwt.access_ttl_minutes sets the account token lifetime", func(t *testing.T) {
		loadConfig(t, "env: 'production'\njwt:\n  access_ttl_minutes: 43200\n")
		if got := getJwtTTL(); !within(got, 30*24*time.Hour) {
			t.Fatalf("want ~30 days, got %s", time.Until(got))
		}
	})

	t.Run("jwt.refresh_ttl_days sets the refresh token lifetime", func(t *testing.T) {
		loadConfig(t, "jwt:\n  refresh_ttl_days: 90\n")
		if got := getJwtRefreshTTL(); !within(got, 90*24*time.Hour) {
			t.Fatalf("want ~90 days, got %s", time.Until(got))
		}
	})

	t.Run("without the keys, today's defaults apply: 30 minutes and 7 days", func(t *testing.T) {
		loadConfig(t, "env: 'production'\n")
		if got := getJwtTTL(); !within(got, 30*time.Minute) {
			t.Fatalf("access: want ~30 minutes, got %s", time.Until(got))
		}
		if got := getJwtRefreshTTL(); !within(got, 7*24*time.Hour) {
			t.Fatalf("refresh: want ~7 days, got %s", time.Until(got))
		}
	})
}
