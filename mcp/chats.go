package mcp

import (
	"context"
	"time"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// ListChatsInput is the typed input for imessage_list_chats.
type ListChatsInput struct {
	Limit       int    `json:"limit,omitempty" jsonschema:"description=number of chats to return,minimum=1,maximum=500,default=20"`
	Offset      int    `json:"offset,omitempty" jsonschema:"description=pagination offset,minimum=0,default=0"`
	IncludeSMS  bool   `json:"include_sms,omitempty" jsonschema:"description=include SMS conversations alongside iMessage,default=false"`
	OnlyUnread  bool   `json:"only_unread,omitempty" jsonschema:"description=only return chats with unread incoming messages,default=false"`
	HandleQuery string `json:"handle_query,omitempty" jsonschema:"description=optional substring filter against participant phone/email"`
}

func listChats(ctx context.Context, c *imessage.Client, in ListChatsInput) (any, error) {
	res, err := c.ListChats(ctx, imessage.ChatListParams{
		Limit:       in.Limit,
		Offset:      in.Offset,
		IncludeSMS:  in.IncludeSMS,
		OnlyUnread:  in.OnlyUnread,
		HandleQuery: in.HandleQuery,
	})
	if err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	return mcptool.PageOf(res, "", limit), nil
}

// GetMessagesInput is the typed input for imessage_get_messages.
type GetMessagesInput struct {
	ChatID    int64  `json:"chat_id,omitempty" jsonschema:"description=chat ROWID from imessage_list_chats (one of chat_id/chat_guid/handle_id required)"`
	ChatGUID  string `json:"chat_guid,omitempty" jsonschema:"description=chat GUID from imessage_list_chats (one of chat_id/chat_guid/handle_id required)"`
	HandleID  int64  `json:"handle_id,omitempty" jsonschema:"description=handle ROWID for direct-message lookup"`
	Limit     int    `json:"limit,omitempty" jsonschema:"description=messages per page,minimum=1,maximum=500,default=50"`
	BeforeID  int64  `json:"before_id,omitempty" jsonschema:"description=pagination cursor: only return messages with ROWID < before_id"`
	SinceUnix int64  `json:"since_unix,omitempty" jsonschema:"description=lower bound on sent time, unix seconds"`
	UntilUnix int64  `json:"until_unix,omitempty" jsonschema:"description=upper bound on sent time, unix seconds"`
}

func getMessages(ctx context.Context, c *imessage.Client, in GetMessagesInput) (any, error) {
	p := imessage.MessageListParams{
		ChatID:   in.ChatID,
		ChatGUID: in.ChatGUID,
		HandleID: in.HandleID,
		Limit:    in.Limit,
		BeforeID: in.BeforeID,
	}
	if in.SinceUnix > 0 {
		p.Since = time.Unix(in.SinceUnix, 0)
	}
	if in.UntilUnix > 0 {
		p.Until = time.Unix(in.UntilUnix, 0)
	}
	res, err := c.GetMessages(ctx, p)
	if err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	return mcptool.PageOf(res, "", limit), nil
}

var chatTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, ListChatsInput](
		"imessage_list_chats",
		"List recent iMessage (and optionally SMS) conversations with participants and unread counts",
		"ListChats",
		listChats,
	),
	mcptool.Define[*imessage.Client, GetMessagesInput](
		"imessage_get_messages",
		"Fetch messages for a specific chat or handle, paginated and optionally bounded by time",
		"GetMessages",
		getMessages,
	),
}
