package mcp

import (
	"context"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// ReactInput is the typed input for imessage_react.
type ReactInput struct {
	ChatGUID    string `json:"chat_guid" jsonschema:"description=GUID of the chat containing the target message,required"`
	MessageGUID string `json:"message_guid" jsonschema:"description=GUID of the message being reacted to,required"`
	Kind        string `json:"kind" jsonschema:"description=tapback type (removal not supported via AppleScript),enum=love,enum=like,enum=dislike,enum=laugh,enum=emphasize,enum=question,required"`
	Confirm     bool   `json:"confirm" jsonschema:"description=must be true when the host enforces send-confirmation,required"`
}

func react(ctx context.Context, c *imessage.Client, in ReactInput) (any, error) {
	if err := c.React(ctx, imessage.ReactParams{
		ChatGUID:    in.ChatGUID,
		MessageGUID: in.MessageGUID,
		Kind:        imessage.Reaction(in.Kind),
		Confirm:     in.Confirm,
	}); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "fallback": "emoji_text"}, nil
}

var reactTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, ReactInput](
		"imessage_react",
		"Add a tapback reaction to a message (currently sent as the corresponding emoji due to AppleScript limits)",
		"React",
		react,
	),
}
