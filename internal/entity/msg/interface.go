package msg

import (
	"context"

	"github.com/flazhgrowth/fg-tamagochi/pkg/http/request"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/response"
)

type MsgRepository interface {
	Insert(ctx context.Context, datum *Msg) (err error)
	Find(ctx context.Context, filter MsgFilter) (data Msgs, err error)
}

type MsgService interface {
	Create(ctx context.Context, args CreateRequest) (err error)
	ListMessages(ctx context.Context, args ListMessagesRequest) (resp *ListMessagesResponse, err error)
}

type MsgAPI interface {
	Create(req request.Request, resp response.Response)
	ListMessages(req request.Request, resp response.Response)
}
