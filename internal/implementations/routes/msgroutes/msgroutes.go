package msgroutes

import (
	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/msg"
	"github.com/flazhgrowth/baec-portfolio-api/internal/implementations/api"
	"github.com/flazhgrowth/fg-tamagochi/pkg/http/router"
)

var (
	tag = "Message"
)

func Routes(version router.Router, apis *api.APIs) {
	version.Group("/messages", func(msgR router.Router) {
		msgR.Get("/", apis.MsgAPI.ListMessages, &router.RouterDocs{
			Request:     msg.ListMessagesRequest{},
			Response:    msg.ListMessagesResponse{},
			Tags:        tag,
			Title:       "List Messages",
			Description: "List messages from visitors",
		})
		msgR.Post("/", apis.MsgAPI.Create, &router.RouterDocs{
			Request:     msg.CreateRequest{},
			Tags:        tag,
			Title:       "Leave Message",
			Description: "Leave message for the owner",
		})
	})
}
