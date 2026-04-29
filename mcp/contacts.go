package mcp

import (
	"context"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// ResolveContactInput is the typed input for imessage_resolve_contact.
type ResolveContactInput struct {
	Query string `json:"query" jsonschema:"description=name substring or exact phone/email to match in macOS Contacts,required"`
}

func resolveContact(ctx context.Context, c *imessage.Client, in ResolveContactInput) (any, error) {
	return c.ResolveContact(ctx, in.Query)
}

// CheckIMessageInput is the typed input for imessage_check_imessage.
type CheckIMessageInput struct {
	Handle string `json:"handle" jsonschema:"description=phone (E.164) or email to check for prior iMessage activity,required"`
}

func checkIMessage(ctx context.Context, c *imessage.Client, in CheckIMessageInput) (any, error) {
	ok, err := c.IsAvailableOnIMessage(ctx, in.Handle)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"handle":     in.Handle,
		"normalized": imessage.NormalizeHandle(in.Handle),
		"available":  ok,
	}, nil
}

var contactTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, ResolveContactInput](
		"imessage_resolve_contact",
		"Resolve a name or handle to one or more macOS AddressBook contacts (with phones and emails)",
		"ResolveContact",
		resolveContact,
	),
	mcptool.Define[*imessage.Client, CheckIMessageInput](
		"imessage_check_imessage",
		"Check whether a handle has prior iMessage history locally (best-effort signal of iMessage availability)",
		"IsAvailableOnIMessage",
		checkIMessage,
	),
}
