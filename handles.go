package imessage

import (
	"context"
	"strings"
)

// normalizeHandle returns a canonical form of a phone number or email
// suitable for equality comparison. Phone numbers: strip whitespace,
// punctuation, parentheses; preserve leading '+'. Emails: lowercase, trim.
// Anything else: trim + lowercase.
func normalizeHandle(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return ""
	}
	if strings.Contains(h, "@") {
		return strings.ToLower(h)
	}
	var b strings.Builder
	for i, r := range h {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return strings.ToLower(h)
	}
	// Heuristic: a 10-digit US number without country code -> prepend +1.
	if len(out) == 10 && !strings.HasPrefix(out, "+") {
		return "+1" + out
	}
	if len(out) == 11 && strings.HasPrefix(out, "1") {
		return "+" + out
	}
	if !strings.HasPrefix(out, "+") {
		return "+" + out
	}
	return out
}

// recipientAllowed enforces WithAllowedRecipients. Returns nil when the
// allowlist is disabled or when handle is in it.
func (c *Client) recipientAllowed(handle string) error {
	if len(c.allowedHandles) == 0 {
		return nil
	}
	if _, ok := c.allowedHandles[normalizeHandle(handle)]; ok {
		return nil
	}
	return ErrRecipientNotAllowed
}

// chatGUIDAllowed returns nil when no allowlist is set or when every
// participant in the chat is on the allowlist. This protects against
// blasting an LLM-driven message into a chat that contains an off-list
// participant.
func (c *Client) chatGUIDAllowed(ctx context.Context, chatGUID string) error {
	if len(c.allowedHandles) == 0 || chatGUID == "" {
		return nil
	}
	parts, err := c.chatParticipants(ctx, chatGUID)
	if err != nil {
		return err
	}
	for _, p := range parts {
		if _, ok := c.allowedHandles[normalizeHandle(p.Identifier)]; !ok {
			return ErrRecipientNotAllowed
		}
	}
	return nil
}

// chatParticipants returns the handles for a chat by GUID. Used by both
// the allowlist guard and ListChats enrichment.
func (c *Client) chatParticipants(ctx context.Context, chatGUID string) ([]Handle, error) {
	db, err := c.database(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
SELECT h.ROWID, h.id, h.service
FROM chat c
JOIN chat_handle_join chj ON chj.chat_id = c.ROWID
JOIN handle h ON h.ROWID = chj.handle_id
WHERE c.guid = ?`
	rows, err := db.QueryContext(ctx, q, chatGUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Handle{}
	for rows.Next() {
		var h Handle
		var svc string
		if err := rows.Scan(&h.ID, &h.Identifier, &svc); err != nil {
			return nil, err
		}
		h.Service = Service(svc)
		out = append(out, h)
	}
	return out, rows.Err()
}
