package imessage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ListChats returns recent chats ordered by last message activity.
func (c *Client) ListChats(ctx context.Context, p ChatListParams) ([]Chat, error) {
	db, err := c.database(ctx)
	if err != nil {
		return nil, err
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}

	var conds []string
	var args []any
	if !p.IncludeSMS {
		conds = append(conds, "c.service_name = 'iMessage'")
	}
	if p.OnlyUnread {
		conds = append(conds, `EXISTS (
			SELECT 1 FROM chat_message_join cmj
			JOIN message m ON m.ROWID = cmj.message_id
			WHERE cmj.chat_id = c.ROWID AND m.is_from_me = 0 AND m.is_read = 0
		)`)
	}
	if q := strings.TrimSpace(p.HandleQuery); q != "" {
		conds = append(conds, `EXISTS (
			SELECT 1 FROM chat_handle_join chj
			JOIN handle h ON h.ROWID = chj.handle_id
			WHERE chj.chat_id = c.ROWID AND h.id LIKE ?
		)`)
		args = append(args, "%"+q+"%")
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	q := fmt.Sprintf(`
SELECT
  c.ROWID, c.guid, COALESCE(c.display_name, ''), COALESCE(c.service_name, ''),
  (SELECT COUNT(*) FROM chat_handle_join chj WHERE chj.chat_id = c.ROWID) AS pcount,
  COALESCE((SELECT MAX(m.date) FROM chat_message_join cmj
            JOIN message m ON m.ROWID = cmj.message_id
            WHERE cmj.chat_id = c.ROWID), 0) AS last_date,
  (SELECT COUNT(*) FROM chat_message_join cmj
            JOIN message m ON m.ROWID = cmj.message_id
            WHERE cmj.chat_id = c.ROWID AND m.is_from_me = 0 AND m.is_read = 0) AS unread
FROM chat c
%s
ORDER BY last_date DESC
LIMIT ? OFFSET ?`, where)

	args = append(args, limit, p.Offset)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var ch Chat
		var pcount int
		var lastDate int64
		if err := rows.Scan(&ch.ID, &ch.GUID, &ch.DisplayName, (*string)(&ch.Service),
			&pcount, &lastDate, &ch.UnreadCount); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
		}
		ch.IsGroup = pcount > 1
		ch.LastActivityAt = fromAppleDate(lastDate)
		chats = append(chats, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Enrich participants + display names.
	contactsMap := c.resolveContactsMap(ctx)
	for i := range chats {
		parts, err := c.chatParticipants(ctx, chats[i].GUID)
		if err == nil {
			for j := range parts {
				if name := contactsMap[normalizeHandle(parts[j].Identifier)]; name != "" {
					parts[j].DisplayName = name
				}
			}
			chats[i].Participants = parts
		}
		if chats[i].DisplayName == "" && len(chats[i].Participants) > 0 {
			names := make([]string, 0, len(chats[i].Participants))
			for _, h := range chats[i].Participants {
				if h.DisplayName != "" {
					names = append(names, h.DisplayName)
				} else {
					names = append(names, h.Identifier)
				}
			}
			chats[i].DisplayName = strings.Join(names, ", ")
		}
	}
	return chats, nil
}

// GetMessages returns paginated messages for a chat. Specify exactly one
// of ChatID, ChatGUID, or HandleID.
func (c *Client) GetMessages(ctx context.Context, p MessageListParams) ([]Message, error) {
	db, err := c.database(ctx)
	if err != nil {
		return nil, err
	}
	if p.ChatID == 0 && p.ChatGUID == "" && p.HandleID == 0 {
		return nil, fmt.Errorf("%w: ChatID, ChatGUID, or HandleID is required", ErrInvalidParams)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	chatID := p.ChatID
	if chatID == 0 && p.ChatGUID != "" {
		if err := db.QueryRowContext(ctx, `SELECT ROWID FROM chat WHERE guid = ?`, p.ChatGUID).Scan(&chatID); err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("%w: chat guid %s", ErrNotFound, p.ChatGUID)
			}
			return nil, err
		}
	}

	var conds []string
	var args []any
	if chatID != 0 {
		conds = append(conds, `cmj.chat_id = ?`)
		args = append(args, chatID)
	}
	if p.HandleID != 0 {
		conds = append(conds, `m.handle_id = ?`)
		args = append(args, p.HandleID)
	}
	if p.BeforeID != 0 {
		conds = append(conds, `m.ROWID < ?`)
		args = append(args, p.BeforeID)
	}
	if !p.Since.IsZero() {
		conds = append(conds, `m.date >= ?`)
		args = append(args, toAppleDate(p.Since))
	}
	if !p.Until.IsZero() {
		conds = append(conds, `m.date <= ?`)
		args = append(args, toAppleDate(p.Until))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
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
%s
ORDER BY m.date DESC
LIMIT ?`, where)

	args = append(args, limit)

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

	// Hydrate attachments per message.
	if len(out) > 0 {
		atts, err := c.attachmentsForMessages(ctx, messageIDs(out))
		if err == nil {
			for i := range out {
				out[i].Attachments = atts[out[i].ID]
			}
		}
	}
	return out, nil
}

func messageIDs(msgs []Message) []int64 {
	out := make([]int64, len(msgs))
	for i, m := range msgs {
		out[i] = m.ID
	}
	return out
}

// scanMessages decodes rows from the wide SELECT used by GetMessages,
// Search, and Watch.
func scanMessages(rows *sql.Rows, contacts map[string]string) ([]Message, error) {
	var out []Message
	for rows.Next() {
		var m Message
		var attributedBody []byte
		var date, dateRead, dateDelivered int64
		var assocType int
		var senderID string
		if err := rows.Scan(
			&m.ID, &m.GUID, &m.ChatID,
			&m.Text, &attributedBody,
			&senderID, &m.IsFromMe,
			(*string)(&m.Service),
			&date, &dateRead, &dateDelivered,
			&m.IsRead, &m.IsDelivered,
			&m.TapbackTarget, &assocType,
			&m.ReplyToGUID,
		); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
		}
		if m.Text == "" && len(attributedBody) > 0 {
			m.Text = decodeAttributedBody(attributedBody)
		}
		m.SenderHandle = senderID
		if !m.IsFromMe && senderID != "" {
			if name := contacts[normalizeHandle(senderID)]; name != "" {
				m.SenderName = name
			}
		}
		m.SentAt = fromAppleDate(date)
		m.ReadAt = fromAppleDate(dateRead)
		m.DeliveredAt = fromAppleDate(dateDelivered)
		m.IsReply = m.ReplyToGUID != ""
		if assocType >= 2000 && assocType <= 3005 {
			m.IsTapback = true
			m.TapbackKind = tapbackFromAssocType(assocType)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func tapbackFromAssocType(t int) Reaction {
	switch t {
	case 2000:
		return ReactionLove
	case 2001:
		return ReactionLike
	case 2002:
		return ReactionDislike
	case 2003:
		return ReactionLaugh
	case 2004:
		return ReactionEmphasize
	case 2005:
		return ReactionQuestion
	case 3000:
		return ReactionRemoveLove
	case 3001:
		return ReactionRemoveLike
	default:
		return ReactionRemoveOther
	}
}
