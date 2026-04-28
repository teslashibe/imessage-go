package imessage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
)

// db opens chat.db lazily in read-only mode and caches the handle.
func (c *Client) database(ctx context.Context) (*sql.DB, error) {
	c.dbOnce.Do(func() {
		if runtime.GOOS != "darwin" {
			c.dbErr = ErrUnsupportedOS
			return
		}
		info, err := os.Stat(c.chatDBPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.dbErr = fmt.Errorf("%w: %s", ErrChatDBNotFound, c.chatDBPath)
				return
			}
			if errors.Is(err, os.ErrPermission) {
				c.dbErr = fmt.Errorf("%w: %s", ErrChatDBPermission, c.chatDBPath)
				return
			}
			c.dbErr = fmt.Errorf("imessage: stat chat.db: %w", err)
			return
		}
		if info.IsDir() {
			c.dbErr = fmt.Errorf("%w: %s is a directory", ErrChatDBNotFound, c.chatDBPath)
			return
		}

		// Read-only, immutable mode + WAL-tolerant via mode=ro and
		// _query_only=true. We use file: URL form so query params apply.
		dsn := "file:" + url.PathEscape(c.chatDBPath) + "?mode=ro&_pragma=query_only(true)&_pragma=busy_timeout(2000)"
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			c.dbErr = fmt.Errorf("imessage: open chat.db: %w", err)
			return
		}
		// Ping to surface permission errors eagerly.
		if err := db.PingContext(ctx); err != nil {
			db.Close()
			if errors.Is(err, os.ErrPermission) || isPermissionError(err) {
				c.dbErr = fmt.Errorf("%w: %s", ErrChatDBPermission, c.chatDBPath)
				return
			}
			c.dbErr = fmt.Errorf("imessage: ping chat.db: %w", err)
			return
		}
		c.db = db
	})
	return c.db, c.dbErr
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for _, marker := range []string{"permission denied", "operation not permitted", "unable to open database"} {
		if containsFold(s, marker) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	// simple lowercase compare
	ls := toLower(s)
	lsub := toLower(sub)
	for i := 0; i+len(lsub) <= len(ls); i++ {
		if ls[i:i+len(lsub)] == lsub {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
