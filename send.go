package imessage

import (
	"context"
	"fmt"
	"os"
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

	// Auto-route: if Recipient and no service specified, ask chat.db
	// whether we've ever reached this handle on iMessage. If so, prefer
	// it; otherwise fall back to SMS.
	service := p.Service
	if service == "" && p.Recipient != "" {
		ok, _ := c.IsAvailableOnIMessage(ctx, p.Recipient)
		if ok {
			service = ServiceIMessage
		} else {
			service = ServiceSMS
		}
	}

	_, err := c.runJXA(ctx, jsxSendToBuddy, map[string]any{
		"body":      p.Body,
		"chat_guid": p.ChatGUID,
		"recipient": p.Recipient,
		"service":   string(service),
	})
	return err
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
	if _, err := os.Stat(p.FilePath); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
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
	service := p.Service
	if service == "" && p.Recipient != "" {
		ok, _ := c.IsAvailableOnIMessage(ctx, p.Recipient)
		if ok {
			service = ServiceIMessage
		} else {
			service = ServiceSMS
		}
	}
	_, err := c.runJXA(ctx, jsxSendToBuddy, map[string]any{
		"body":            p.Caption,
		"attachment_path": p.FilePath,
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
	case ReactionRemoveLove, ReactionRemoveLike, ReactionRemoveOther:
		return "↩️"
	}
	return ""
}
