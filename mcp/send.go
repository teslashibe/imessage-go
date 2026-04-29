package mcp

import (
	"context"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// SendMessageInput is the typed input for imessage_send_message.
type SendMessageInput struct {
	ChatGUID  string `json:"chat_guid,omitempty" jsonschema:"description=existing chat GUID (mutually exclusive with recipient)"`
	Recipient string `json:"recipient,omitempty" jsonschema:"description=phone number (E.164) or email for a 1:1 conversation (mutually exclusive with chat_guid)"`
	Body      string `json:"body" jsonschema:"description=plain-text message to send,required"`
	Service   string `json:"service,omitempty" jsonschema:"description=force a delivery channel: iMessage or SMS. Omit for auto-routing,enum=iMessage,enum=SMS"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true when the host enforces send-confirmation,required"`
}

func sendMessage(ctx context.Context, c *imessage.Client, in SendMessageInput) (any, error) {
	if err := c.SendMessage(ctx, imessage.SendParams{
		ChatGUID:  in.ChatGUID,
		Recipient: in.Recipient,
		Body:      in.Body,
		Service:   imessage.Service(in.Service),
		Confirm:   in.Confirm,
	}); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// SendAttachmentInput is the typed input for imessage_send_attachment.
type SendAttachmentInput struct {
	ChatGUID  string `json:"chat_guid,omitempty" jsonschema:"description=existing chat GUID (mutually exclusive with recipient)"`
	Recipient string `json:"recipient,omitempty" jsonschema:"description=phone (E.164) or email for a 1:1 conversation (mutually exclusive with chat_guid)"`
	FilePath  string `json:"file_path" jsonschema:"description=absolute path to a local file to attach,required"`
	Caption   string `json:"caption,omitempty" jsonschema:"description=optional message body sent alongside the attachment"`
	Service   string `json:"service,omitempty" jsonschema:"description=force iMessage or SMS. Omit for auto,enum=iMessage,enum=SMS"`
	Confirm   bool   `json:"confirm" jsonschema:"description=must be true when the host enforces send-confirmation,required"`
}

func sendAttachment(ctx context.Context, c *imessage.Client, in SendAttachmentInput) (any, error) {
	if err := c.SendAttachment(ctx, imessage.SendAttachmentParams{
		ChatGUID:  in.ChatGUID,
		Recipient: in.Recipient,
		FilePath:  in.FilePath,
		Caption:   in.Caption,
		Service:   imessage.Service(in.Service),
		Confirm:   in.Confirm,
	}); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

var sendTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, SendMessageInput](
		"imessage_send_message",
		"Send a plain-text iMessage or SMS to an existing chat or new recipient (auto-routes when service is omitted)",
		"SendMessage",
		sendMessage,
	),
	mcptool.Define[*imessage.Client, SendAttachmentInput](
		"imessage_send_attachment",
		"Send a file attachment via iMessage or SMS, with an optional caption",
		"SendAttachment",
		sendAttachment,
	),
}
