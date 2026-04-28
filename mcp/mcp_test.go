package mcp_test

import (
	"reflect"
	"testing"

	imessage "github.com/teslashibe/imessage-go"
	immcp "github.com/teslashibe/imessage-go/mcp"
	"github.com/teslashibe/mcptool"
)

// TestEveryClientMethodIsWrappedOrExcluded fails when a new exported
// method is added to *imessage.Client without either being wrapped by an
// MCP tool or being added to immcp.Excluded with a reason.
func TestEveryClientMethodIsWrappedOrExcluded(t *testing.T) {
	rep := mcptool.Coverage(
		reflect.TypeOf(&imessage.Client{}),
		immcp.Provider{}.Tools(),
		immcp.Excluded,
	)
	if len(rep.Missing) > 0 {
		t.Fatalf("methods missing MCP exposure (add a tool or list in excluded.go): %v", rep.Missing)
	}
	if len(rep.UnknownExclusions) > 0 {
		t.Fatalf("excluded.go references methods that don't exist on *Client (rename?): %v", rep.UnknownExclusions)
	}
	if len(rep.Wrapped)+len(rep.Excluded) == 0 {
		t.Fatal("no wrapped or excluded methods detected — coverage helper is mis-configured")
	}
}

func TestToolsValidate(t *testing.T) {
	if err := mcptool.ValidateTools(immcp.Provider{}.Tools()); err != nil {
		t.Fatal(err)
	}
}

func TestPlatformName(t *testing.T) {
	if got := (immcp.Provider{}).Platform(); got != "imessage" {
		t.Errorf("Platform() = %q, want imessage", got)
	}
}

func TestToolsHaveImessagePrefix(t *testing.T) {
	const prefix = "imessage_"
	for _, tool := range (immcp.Provider{}).Tools() {
		if len(tool.Name) < len(prefix) || tool.Name[:len(prefix)] != prefix {
			t.Errorf("tool %q lacks %s prefix", tool.Name, prefix)
		}
	}
}
