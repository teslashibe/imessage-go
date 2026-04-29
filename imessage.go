// Package imessage provides a Go client for macOS iMessage and SMS via the
// Messages.app local SQLite database and AppleScript bridge.
//
// It supports listing conversations, reading and searching messages,
// sending iMessages and SMS (with auto-routing), tapback reactions,
// attachment fetch and send, contact resolution from the macOS Address
// Book, and a polling watcher for new messages.
//
// Reads come from ~/Library/Messages/chat.db (requires Full Disk Access for
// the host process). Sends go through Messages.app via osascript with JXA
// (requires Automation permission for Messages).
//
// macOS only. Requires Messages.app to be signed in to iMessage. SMS
// fallback requires an iPhone with Text Message Forwarding enabled.
package imessage

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// Config holds optional path overrides. The zero value uses macOS defaults.
type Config struct {
	// ChatDBPath overrides ~/Library/Messages/chat.db. Useful for tests.
	ChatDBPath string
	// AddressBookDir overrides ~/Library/Application Support/AddressBook/Sources.
	AddressBookDir string
	// AttachmentsDir overrides ~/Library/Messages/Attachments.
	AttachmentsDir string
}

// Client is a macOS iMessage client. Reads run against chat.db; sends run
// through Messages.app via osascript. Safe for concurrent use.
type Client struct {
	chatDBPath     string
	addressBookDir string
	attachmentsDir string

	osascriptPath  string
	confirmSends   bool
	allowedHandles map[string]struct{}
	maxAttachBytes int64

	dbOnce sync.Once
	dbErr  error
	db     *sql.DB

	contactsOnce sync.Once
	contactsErr  error
	contactsMap  map[string]string // normalized handle -> display name
}

const (
	defaultMaxAttachBytes = 25 * 1024 * 1024 // 25MB cap for base64 round-trips
	defaultOsascriptPath  = "/usr/bin/osascript"
)

// New constructs a Client. macOS-only; on other platforms most methods will
// return ErrUnsupportedOS.
func New(cfg Config, opts ...Option) *Client {
	home, _ := os.UserHomeDir()
	c := &Client{
		chatDBPath:     firstNonEmpty(cfg.ChatDBPath, filepath.Join(home, "Library", "Messages", "chat.db")),
		addressBookDir: firstNonEmpty(cfg.AddressBookDir, filepath.Join(home, "Library", "Application Support", "AddressBook", "Sources")),
		attachmentsDir: firstNonEmpty(cfg.AttachmentsDir, filepath.Join(home, "Library", "Messages", "Attachments")),
		osascriptPath:  defaultOsascriptPath,
		confirmSends:   true,
		maxAttachBytes: defaultMaxAttachBytes,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Option configures a Client.
type Option func(*Client)

// WithOsascriptPath overrides the default /usr/bin/osascript binary path.
func WithOsascriptPath(p string) Option {
	return func(c *Client) {
		if p != "" {
			c.osascriptPath = p
		}
	}
}

// WithRequireConfirm controls whether send-style methods require an explicit
// confirm flag set by the caller. Default: true (safer).
func WithRequireConfirm(require bool) Option {
	return func(c *Client) { c.confirmSends = require }
}

// WithAllowedRecipients restricts send-style methods to a fixed set of
// recipient handles (phone numbers in E.164 form or email addresses). When
// set, sends to any other handle return ErrRecipientNotAllowed. Pass nil
// or empty to disable the allowlist.
func WithAllowedRecipients(handles []string) Option {
	return func(c *Client) {
		if len(handles) == 0 {
			c.allowedHandles = nil
			return
		}
		m := make(map[string]struct{}, len(handles))
		for _, h := range handles {
			m[normalizeHandle(h)] = struct{}{}
		}
		c.allowedHandles = m
	}
}

// WithMaxAttachmentBytes caps the size of attachments returned by
// GetAttachment as base64. Default: 25 MiB. Pass 0 for no cap.
func WithMaxAttachmentBytes(n int64) Option {
	return func(c *Client) { c.maxAttachBytes = n }
}

// Close releases the SQLite handle. Safe to call multiple times.
func (c *Client) Close() error {
	if c.db != nil {
		err := c.db.Close()
		c.db = nil
		return err
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
