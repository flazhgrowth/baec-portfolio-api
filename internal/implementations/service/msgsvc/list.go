package msgsvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/apierrors"
)

func (svc *service) ListMessages(ctx context.Context, args msg.ListMessagesRequest) (resp *msg.ListMessagesResponse, err error) {
	msgsFound, err := svc.msgRepo.Find(ctx, msg.MsgFilter{})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, apierrors.ErrorInternalServerError()
	}

	resp = &msg.ListMessagesResponse{}
	for _, msgFound := range msgsFound {
		resp.Messages = append(resp.Messages, msg.MessageResponse{
			Msg: msgFound.Message,
		})
	}

	return resp, nil
}
