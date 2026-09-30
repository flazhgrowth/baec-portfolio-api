package account

import (
	"context"

	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

type (
	API interface {
		Login(req request.Request, resp response.Response)
		Register(req request.Request, resp response.Response)
		Me(req request.Request, resp response.Response)
		Logout(req request.Request, resp response.Response)
	}

	Service interface {
		Login(ctx context.Context, args LoginRequest) (resp *LoginResponse, err error)
		Register(ctx context.Context, args RegisterRequest) (resp *LoginResponse, err error)
		// Me resolves the account behind a token. An account that no longer exists is
		// 401 INVALID_TOKEN: the token outlived it.
		Me(ctx context.Context, id string) (resp *UserResponse, err error)
	}

	Repository interface {
		Get(ctx context.Context, filter AccountFilter) (datum *Account, err error)
		Find(ctx context.Context, filter AccountFilter, sorter AccountSorter) (data Accounts, err error)
		Insert(ctx context.Context, datum *Account) (err error)
		Update(ctx context.Context, fields AccountUpdateFields, filter AccountFilter) (err error)
		Delete(ctx context.Context, filter AccountFilter) (err error)
	}
)
