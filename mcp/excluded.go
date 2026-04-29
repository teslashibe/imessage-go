package mcp

// Excluded enumerates exported methods on *imessage.Client that are
// intentionally not exposed via MCP. Each entry must have a non-empty
// reason.
//
// The coverage test in mcp_test.go fails if any exported method on
// *Client is neither wrapped by a Tool nor present in this map (or vice-
// versa: if an entry here doesn't correspond to a real method).
var Excluded = map[string]string{
	"Close": "lifecycle method owned by the host application; not a callable agent tool",
}
