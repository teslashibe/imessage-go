package mcp

import (
	"context"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// StatusInput is the typed input for imessage_status. It takes no fields.
type StatusInput struct{}

func status(ctx context.Context, c *imessage.Client, _ StatusInput) (any, error) {
	return c.Status(ctx)
}

var statusTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, StatusInput](
		"imessage_status",
		"Check Full Disk Access, Automation, Messages.app and AddressBook readiness for iMessage tools",
		"Status",
		status,
	),
}
