package imessage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SendMessage sends a plain-text message either to an existing chat (by
// GUID) or to a single recipient (auto-routed iMessage/SMS).
//
// When WithRequireConfirm is true (default), p.Confirm must be true. When
// WithAllowedRecipients is set, every target handle must be on the allowlist.
func (c *Client) SendMessage(ctx context.Context, p SendParams) error {
	if strings.TrimSpace(p.Body) == "" {
		return ErrMessageEmpty
	}
	if p.ChatGUID == "" && p.Recipient == "" {
		return fmt.Errorf("%w: ChatGUID or Recipient required", ErrInvalidParams)
	}
	if p.ChatGUID != "" && p.Recipient != "" {
		return fmt.Errorf("%w: ChatGUID and Recipient are mutually exclusive", ErrInvalidParams)
	}
	if c.confirmSends && !p.Confirm {
		return ErrConfirmRequired
	}
	if p.Service != "" && p.Service != ServiceIMessage && p.Service != ServiceSMS {
		return ErrUnknownService
	}

	if p.Recipient != "" {
		if err := c.recipientAllowed(p.Recipient); err != nil {
			return err
		}
	} else {
		if err := c.chatGUIDAllowed(ctx, p.ChatGUID); err != nil {
			return err
		}
	}

	service, err := c.routeService(ctx, p.Service, p.Recipient)
	if err != nil {
		return err
	}

	_, err = c.runJXA(ctx, jsxSendToBuddy, map[string]any{
		"body":      p.Body,
		"chat_guid": p.ChatGUID,
		"recipient": p.Recipient,
		"service":   string(service),
	})
	return err
}

// routeService chooses the iMessage/SMS channel for a 1:1 send. When
// service is explicitly set, returns it unchanged. When the recipient is
// known to iMessage history, returns iMessage. When the DB is unreachable
// (e.g. no Full Disk Access), defaults to iMessage rather than silently
// falling back to SMS — SMS may incur carrier charges and is irreversible
// once dispatched, so the safe default on uncertainty is iMessage.
func (c *Client) routeService(ctx context.Context, override Service, recipient string) (Service, error) {
	if override != "" {
		return override, nil
	}
	if recipient == "" {
		// Sending to an existing chat: let Messages.app pick.
		return "", nil
	}
	ok, err := c.IsAvailableOnIMessage(ctx, recipient)
	if err != nil {
		// DB lookup failed (likely no FDA). Default to iMessage to avoid
		// inadvertent SMS charges; the user can pass Service=SMS to force.
		return ServiceIMessage, nil
	}
	if ok {
		return ServiceIMessage, nil
	}
	return ServiceSMS, nil
}

// SendAttachment sends a file (with optional caption) to an existing chat
// or a single recipient.
func (c *Client) SendAttachment(ctx context.Context, p SendAttachmentParams) error {
	if strings.TrimSpace(p.FilePath) == "" {
		return fmt.Errorf("%w: FilePath required", ErrInvalidParams)
	}
	if p.ChatGUID == "" && p.Recipient == "" {
		return fmt.Errorf("%w: ChatGUID or Recipient required", ErrInvalidParams)
	}
	if c.confirmSends && !p.Confirm {
		return ErrConfirmRequired
	}
	absPath, err := filepath.Abs(p.FilePath)
	if err != nil {
		return fmt.Errorf("%w: resolve attachment path: %v", ErrInvalidParams, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: attachment %s is a directory", ErrInvalidParams, absPath)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%w: attachment %s is empty", ErrInvalidParams, absPath)
	}
	if p.Recipient != "" {
		if err := c.recipientAllowed(p.Recipient); err != nil {
			return err
		}
	} else {
		if err := c.chatGUIDAllowed(ctx, p.ChatGUID); err != nil {
			return err
		}
	}
	service, err := c.routeService(ctx, p.Service, p.Recipient)
	if err != nil {
		return err
	}
	_, err = c.runJXA(ctx, jsxSendToBuddy, map[string]any{
		"body":            p.Caption,
		"attachment_path": absPath,
		"chat_guid":       p.ChatGUID,
		"recipient":       p.Recipient,
		"service":         string(service),
	})
	return err
}

// React adds a tapback to a message. macOS scripting doesn't expose true
// tapback objects, so this currently sends the corresponding emoji as a
// regular message — the tapback semantics are preserved visually but the
// message lands as a separate item rather than as a real reaction. The
// JXA reply includes "fallback: emoji_text" to make this explicit.
func (c *Client) React(ctx context.Context, p ReactParams) error {
	if p.ChatGUID == "" || p.MessageGUID == "" {
		return fmt.Errorf("%w: ChatGUID and MessageGUID required", ErrInvalidParams)
	}
	if c.confirmSends && !p.Confirm {
		return ErrConfirmRequired
	}
	if err := c.chatGUIDAllowed(ctx, p.ChatGUID); err != nil {
		return err
	}
	switch p.Kind {
	case ReactionRemoveLove, ReactionRemoveLike, ReactionRemoveOther:
		// Removing a tapback requires sending an associated_message_type
		// 3xxx event that AppleScript cannot construct. Sending a generic
		// emoji would silently mislead the user, so refuse explicitly.
		return fmt.Errorf("%w: tapback removal (%s) is not supported via AppleScript; use Messages.app directly", ErrInvalidParams, p.Kind)
	}
	emoji := tapbackEmoji(p.Kind)
	if emoji == "" {
		return fmt.Errorf("%w: unknown tapback %q", ErrInvalidParams, p.Kind)
	}
	_, err := c.runJXA(ctx, jsxReact, map[string]any{
		"chat_guid":    p.ChatGUID,
		"message_guid": p.MessageGUID,
		"emoji":        emoji,
	})
	return err
}

func tapbackEmoji(r Reaction) string {
	switch r {
	case ReactionLove:
		return "❤️"
	case ReactionLike:
		return "👍"
	case ReactionDislike:
		return "👎"
	case ReactionLaugh:
		return "😂"
	case ReactionEmphasize:
		return "‼️"
	case ReactionQuestion:
		return "❓"
	}
	return ""
}
