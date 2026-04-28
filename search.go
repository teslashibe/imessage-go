package imessage

import (
	"context"
	"fmt"
	"strings"
)

// Search performs full-text search across message bodies. Filters by chat,
// handle, time range, and from-me direction. The query is matched as a
// case-insensitive substring against the decoded text column.
func (c *Client) Search(ctx context.Context, p SearchParams) ([]Message, error) {
	if strings.TrimSpace(p.Query) == "" {
		return nil, fmt.Errorf("%w: query is required", ErrInvalidParams)
	}
	db, err := c.database(ctx)
	if err != nil {
		return nil, err
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	conds := []string{`(LOWER(m.text) LIKE ?)`}
	args := []any{"%" + strings.ToLower(p.Query) + "%"}

	if p.ChatID != 0 {
		conds = append(conds, `cmj.chat_id = ?`)
		args = append(args, p.ChatID)
	}
	if h := strings.TrimSpace(p.Handle); h != "" {
		conds = append(conds, `h.id = ?`)
		args = append(args, h)
	}
	if !p.Since.IsZero() {
		conds = append(conds, `m.date >= ?`)
		args = append(args, toAppleDate(p.Since))
	}
	if !p.Until.IsZero() {
		conds = append(conds, `m.date <= ?`)
		args = append(args, toAppleDate(p.Until))
	}
	if p.FromMe != nil {
		if *p.FromMe {
			conds = append(conds, `m.is_from_me = 1`)
		} else {
			conds = append(conds, `m.is_from_me = 0`)
		}
	}

	q := fmt.Sprintf(`
SELECT m.ROWID, m.guid, COALESCE(cmj.chat_id, 0),
       COALESCE(m.text, ''), m.attributedBody,
       COALESCE(h.id, ''), m.is_from_me,
       COALESCE(m.service, ''),
       m.date, m.date_read, m.date_delivered,
       m.is_read, m.is_delivered,
       COALESCE(m.associated_message_guid, ''), m.associated_message_type,
       COALESCE(m.thread_originator_guid, '')
FROM message m
LEFT JOIN chat_message_join cmj ON cmj.message_id = m.ROWID
LEFT JOIN handle h ON h.ROWID = m.handle_id
WHERE %s
ORDER BY m.date DESC
LIMIT ? OFFSET ?`, strings.Join(conds, " AND "))

	args = append(args, limit, p.Offset)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := c.resolveContactsMap(ctx)
	out, err := scanMessages(rows, contacts)
	if err != nil {
		return nil, err
	}

	// Note: text-column search misses messages stored only in attributedBody
	// (newer macOS). For those, fall back to in-memory filtering of the
	// most recent N messages where text was empty pre-decode.
	if len(out) < limit {
		extra, err := c.searchAttributedBodies(ctx, p, limit-len(out))
		if err == nil {
			seen := map[int64]bool{}
			for _, m := range out {
				seen[m.ID] = true
			}
			for _, m := range extra {
				if !seen[m.ID] {
					out = append(out, m)
				}
			}
		}
	}
	return out, nil
}

// searchAttributedBodies scans the most recent N messages with NULL text
// and applies the substring filter to the decoded attributedBody. Bounded
// to keep this from being a full-table scan.
func (c *Client) searchAttributedBodies(ctx context.Context, p SearchParams, want int) ([]Message, error) {
	db, err := c.database(ctx)
	if err != nil {
		return nil, err
	}
	const scanLimit = 5000

	conds := []string{`(m.text IS NULL OR m.text = '')`, `m.attributedBody IS NOT NULL`}
	var args []any
	if p.ChatID != 0 {
		conds = append(conds, `cmj.chat_id = ?`)
		args = append(args, p.ChatID)
	}
	if !p.Since.IsZero() {
		conds = append(conds, `m.date >= ?`)
		args = append(args, toAppleDate(p.Since))
	}
	if !p.Until.IsZero() {
		conds = append(conds, `m.date <= ?`)
		args = append(args, toAppleDate(p.Until))
	}

	q := fmt.Sprintf(`
SELECT m.ROWID, m.guid, COALESCE(cmj.chat_id, 0),
       COALESCE(m.text, ''), m.attributedBody,
       COALESCE(h.id, ''), m.is_from_me,
       COALESCE(m.service, ''),
       m.date, m.date_read, m.date_delivered,
       m.is_read, m.is_delivered,
       COALESCE(m.associated_message_guid, ''), m.associated_message_type,
       COALESCE(m.thread_originator_guid, '')
FROM message m
LEFT JOIN chat_message_join cmj ON cmj.message_id = m.ROWID
LEFT JOIN handle h ON h.ROWID = m.handle_id
WHERE %s
ORDER BY m.date DESC
LIMIT %d`, strings.Join(conds, " AND "), scanLimit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := c.resolveContactsMap(ctx)
	all, err := scanMessages(rows, contacts)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(p.Query)
	var out []Message
	for _, m := range all {
		if !strings.Contains(strings.ToLower(m.Text), needle) {
			continue
		}
		out = append(out, m)
		if len(out) >= want {
			break
		}
	}
	return out, nil
}
