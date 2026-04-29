package mcp

import (
	"context"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// WatchInput is the typed input for imessage_watch.
type WatchInput struct {
	SinceID int64 `json:"since_id,omitempty" jsonschema:"description=cursor from a previous call (omit or 0 to bootstrap a cursor without fetching messages),minimum=0,default=0"`
	ChatID  int64 `json:"chat_id,omitempty" jsonschema:"description=optional chat ROWID filter,minimum=0"`
	Limit   int   `json:"limit,omitempty" jsonschema:"description=max messages per call,minimum=1,maximum=500,default=100"`
}

func watch(ctx context.Context, c *imessage.Client, in WatchInput) (any, error) {
	return c.Watch(ctx, imessage.WatchParams{
		SinceID: in.SinceID,
		ChatID:  in.ChatID,
		Limit:   in.Limit,
	})
}

var watchTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, WatchInput](
		"imessage_watch",
		"Poll for new messages with ROWID > since_id; returns new messages plus a cursor to use next call",
		"Watch",
		watch,
	),
}
