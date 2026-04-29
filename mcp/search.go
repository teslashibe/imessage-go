package mcp

import (
	"context"
	"time"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// SearchInput is the typed input for imessage_search.
type SearchInput struct {
	Query     string `json:"query" jsonschema:"description=case-insensitive substring to search for in message bodies,required"`
	ChatID    int64  `json:"chat_id,omitempty" jsonschema:"description=optional chat ROWID filter"`
	Handle    string `json:"handle,omitempty" jsonschema:"description=optional sender handle filter (phone/email exact match)"`
	SinceUnix int64  `json:"since_unix,omitempty" jsonschema:"description=lower bound on sent time, unix seconds"`
	UntilUnix int64  `json:"until_unix,omitempty" jsonschema:"description=upper bound on sent time, unix seconds"`
	Limit     int    `json:"limit,omitempty" jsonschema:"description=results per page,minimum=1,maximum=500,default=50"`
	Offset    int    `json:"offset,omitempty" jsonschema:"description=pagination offset,minimum=0,default=0"`
	OnlyMine  *bool  `json:"only_mine,omitempty" jsonschema:"description=true=only my messages\\, false=only theirs\\, omit=both"`
}

func search(ctx context.Context, c *imessage.Client, in SearchInput) (any, error) {
	p := imessage.SearchParams{
		Query:  in.Query,
		ChatID: in.ChatID,
		Handle: in.Handle,
		Limit:  in.Limit,
		Offset: in.Offset,
		FromMe: in.OnlyMine,
	}
	if in.SinceUnix > 0 {
		p.Since = time.Unix(in.SinceUnix, 0)
	}
	if in.UntilUnix > 0 {
		p.Until = time.Unix(in.UntilUnix, 0)
	}
	res, err := c.Search(ctx, p)
	if err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	return mcptool.PageOf(res, "", limit), nil
}

var searchTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, SearchInput](
		"imessage_search",
		"Full-text substring search across message bodies, with chat/handle/time/direction filters",
		"Search",
		search,
	),
}
