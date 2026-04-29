package mcp

import (
	"context"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// GetAttachmentInput is the typed input for imessage_get_attachment.
type GetAttachmentInput struct {
	GUID string `json:"guid" jsonschema:"description=attachment GUID from a Message.attachments entry,required"`
}

func getAttachment(ctx context.Context, c *imessage.Client, in GetAttachmentInput) (any, error) {
	return c.GetAttachment(ctx, in.GUID)
}

var attachmentTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, GetAttachmentInput](
		"imessage_get_attachment",
		"Load an attachment by GUID and return its bytes as base64 (subject to host size cap)",
		"GetAttachment",
		getAttachment,
	),
}
