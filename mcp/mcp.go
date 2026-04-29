// Package mcp exposes the imessage-go [imessage.Client] surface as a set
// of MCP (Model Context Protocol) tools that any host application can
// mount on its own MCP server.
//
// Usage from a host application:
//
//	import (
//	    "github.com/teslashibe/mcptool"
//	    imessage "github.com/teslashibe/imessage-go"
//	    immcp "github.com/teslashibe/imessage-go/mcp"
//	)
//
//	client := imessage.New(imessage.Config{}, imessage.WithRequireConfirm(true))
//	for _, tool := range immcp.Provider{}.Tools() {
//	    // register tool with your MCP server, passing client as the client arg
//	}
//
// All tools wrap exported methods on *imessage.Client. The [Excluded] map
// documents methods intentionally not exposed via MCP. The coverage test
// in mcp_test.go fails if a new exported method is added without either
// being wrapped by a tool or appearing in [Excluded].
package mcp

import "github.com/teslashibe/mcptool"

// Provider implements [mcptool.Provider] for imessage-go. The zero value
// is ready to use.
type Provider struct{}

// Platform returns "imessage".
func (Provider) Platform() string { return "imessage" }

// Tools returns every imessage-go MCP tool, in registration order.
func (Provider) Tools() []mcptool.Tool {
	out := make([]mcptool.Tool, 0,
		len(statusTools)+len(chatTools)+len(searchTools)+
			len(sendTools)+len(reactTools)+len(contactTools)+
			len(attachmentTools)+len(watchTools))
	out = append(out, statusTools...)
	out = append(out, chatTools...)
	out = append(out, searchTools...)
	out = append(out, sendTools...)
	out = append(out, reactTools...)
	out = append(out, contactTools...)
	out = append(out, attachmentTools...)
	out = append(out, watchTools...)
	return out
}
