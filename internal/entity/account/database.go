package account

import (
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/entity"
	"github.com/flazhgrowth/fg-tamagochi/pkg/db/table"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
	"github.com/flazhgrowth/fg-tamagopkg/password"
)

var (
	/*
		CREATE TABLE IF NOT EXISTS accounts (
			"id" varchar(32) PRIMARY KEY,
			"username" VARCHAR(128) NOT NULL,
			"password" VARCHAR(256) NOT NULL,
			"name" VARCHAR(256) NOT NULL,
			"salt" VARCHAR(64) NOT NULL,
			"created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			"updated_at" TIMESTAMPTZ
		);
	*/
	AccountTable table.Table = table.Table{
		Name:          "accounts",
		SelectColumns: []string{"id", "username", "password", "name", "salt", "created_at", "updated_at"},
		InsertColumns: []string{"id", "username", "password", "name", "salt"},
	}
)

type (
	Account struct {
		entity.BaseULIDModel
		Username string `db:"username"`
		Password string `db:"password"`
		Name     string `db:"name"`
		Salt     string `db:"salt"`
	}
	Accounts []Account
)

func (datum *Account) ValidatePassword(plain string) error {
	if !password.Assert(plain, datum.Password, datum.Salt) {
		return apierrors.ErrorUnauthorized("invalid credentials").WithCode("invalid_credentials")
	}

	return nil
}
