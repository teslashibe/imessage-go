package imessage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Watch returns messages with ROWID > p.SinceID, plus a new cursor to feed
// back into the next call. Designed for poll-based "what's new since X"
// loops driven by the agent.
//
// On first use, callers typically call Watch with SinceID=0 to fetch the
// current MAX(ROWID) (returned in WatchResult.Cursor with no messages),
// then poll with that cursor.
func (c *Client) Watch(ctx context.Context, p WatchParams) (WatchResult, error) {
	db, err := c.database(ctx)
	if err != nil {
		return WatchResult{}, err
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	// Bootstrap path: SinceID == 0 — return the current MAX(ROWID) so the
	// caller can start polling from "now".
	if p.SinceID <= 0 {
		var max sql.NullInt64
		if err := db.QueryRowContext(ctx, `SELECT MAX(ROWID) FROM message`).Scan(&max); err != nil {
			return WatchResult{}, err
		}
		return WatchResult{Cursor: max.Int64}, nil
	}

	conds := []string{`m.ROWID > ?`}
	args := []any{p.SinceID}
	if p.ChatID != 0 {
		conds = append(conds, `cmj.chat_id = ?`)
		args = append(args, p.ChatID)
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
ORDER BY m.ROWID ASC
LIMIT ?`, strings.Join(conds, " AND "))
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return WatchResult{}, err
	}
	defer rows.Close()

	contacts := c.resolveContactsMap(ctx)
	msgs, err := scanMessages(rows, contacts)
	if err != nil {
		return WatchResult{}, err
	}
	if msgs == nil {
		// Always return a non-nil slice so JSON encodes as [] not null;
		// agents iterating the result expect an array.
		msgs = []Message{}
	}

	cursor := p.SinceID
	for _, m := range msgs {
		if m.ID > cursor {
			cursor = m.ID
		}
	}
	return WatchResult{Messages: msgs, Cursor: cursor}, nil
}
