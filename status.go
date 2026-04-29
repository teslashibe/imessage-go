package imessage

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
)

// Status reports whether the host has the necessary macOS permissions and
// whether Messages.app is reachable. Designed to be the first tool an
// agent calls when troubleshooting.
func (c *Client) Status(ctx context.Context) (StatusReport, error) {
	rep := StatusReport{
		OS:                 runtime.GOOS,
		ChatDBPath:         c.chatDBPath,
		HelpFullDiskAccess: "Open System Settings → Privacy & Security → Full Disk Access → enable for your terminal/IDE (Cursor, iTerm, Terminal). Restart the host process.",
		HelpAutomation:     "Open System Settings → Privacy & Security → Automation → enable Messages for your terminal/IDE.",
	}
	if runtime.GOOS != "darwin" {
		rep.ChatDBError = ErrUnsupportedOS.Error()
		return rep, nil
	}

	// chat.db readability
	if _, err := os.Stat(c.chatDBPath); err == nil {
		_, dbErr := c.database(ctx)
		if dbErr == nil {
			rep.ChatDBReadable = true
		} else {
			rep.ChatDBError = dbErr.Error()
		}
	} else {
		rep.ChatDBError = err.Error()
	}

	// AddressBook readability (best-effort: presence + at least one .abcddb)
	if files, err := c.findAddressBookDBs(); err != nil {
		rep.AddressBookError = err.Error()
	} else if len(files) == 0 {
		rep.AddressBookError = "no AddressBook-v*.abcddb files under " + c.addressBookDir
	} else {
		rep.AddressBookReadable = true
	}

	// osascript binary
	if _, err := os.Stat(c.osascriptPath); err == nil {
		rep.OsascriptPresent = true
	}

	// Messages.app probe + automation grant probe (cheap, idempotent)
	out, err := c.runJXA(ctx, `
		var Messages = Application('Messages');
		var running = Messages.running();
		return JSON.stringify({running: running});
	`, struct{}{})
	switch {
	case err == nil:
		rep.MessagesAppRunning = strings.Contains(out, `"running":true`)
		rep.AutomationGranted = true
	case errors.Is(err, ErrAutomationPermission):
		rep.AutomationGranted = false
		rep.AutomationError = err.Error()
	default:
		rep.AutomationError = err.Error()
	}
	return rep, nil
}
