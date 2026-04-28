package imessage

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// GetAttachment loads an attachment by its GUID and returns base64-encoded
// bytes (subject to WithMaxAttachmentBytes). Path on disk is also returned
// for callers that prefer to read the file themselves.
func (c *Client) GetAttachment(ctx context.Context, guid string) (Attachment, error) {
	if strings.TrimSpace(guid) == "" {
		return Attachment{}, fmt.Errorf("%w: guid is required", ErrInvalidParams)
	}
	db, err := c.database(ctx)
	if err != nil {
		return Attachment{}, err
	}
	const q = `
SELECT a.ROWID, a.guid, COALESCE(a.transfer_name, ''), COALESCE(a.mime_type, ''),
       COALESCE(a.total_bytes, 0), COALESCE(a.filename, '')
FROM attachment a
WHERE a.guid = ?`
	var att Attachment
	var path string
	if err := db.QueryRowContext(ctx, q, guid).Scan(
		&att.ID, &att.GUID, &att.Filename, &att.MIMEType, &att.Size, &path,
	); err != nil {
		return Attachment{}, fmt.Errorf("%w: attachment %s", ErrNotFound, guid)
	}
	att.Path = expandTilde(path)
	if att.Path == "" {
		return att, ErrAttachmentMissingFile
	}
	f, err := os.Open(att.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return att, ErrAttachmentMissingFile
		}
		return att, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return att, err
	}
	if c.maxAttachBytes > 0 && info.Size() > c.maxAttachBytes {
		return att, fmt.Errorf("%w: %d bytes > cap %d", ErrAttachmentTooLarge, info.Size(), c.maxAttachBytes)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return att, err
	}
	att.Size = int64(len(data))
	att.DataB64 = base64.StdEncoding.EncodeToString(data)
	return att, nil
}

func expandTilde(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return home + p[1:]
}

// attachmentsForMessages returns a map of message ROWID -> []Attachment for
// the supplied message IDs. Used to enrich GetMessages / Search.
func (c *Client) attachmentsForMessages(ctx context.Context, ids []int64) (map[int64][]Attachment, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	db, err := c.database(ctx)
	if err != nil {
		return nil, err
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, v := range ids {
		args[i] = v
	}
	q := fmt.Sprintf(`
SELECT maj.message_id, a.ROWID, a.guid,
       COALESCE(a.transfer_name, ''), COALESCE(a.mime_type, ''),
       COALESCE(a.total_bytes, 0), COALESCE(a.filename, '')
FROM message_attachment_join maj
JOIN attachment a ON a.ROWID = maj.attachment_id
WHERE maj.message_id IN (%s)`, placeholders)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]Attachment{}
	for rows.Next() {
		var msgID int64
		var a Attachment
		var path string
		if err := rows.Scan(&msgID, &a.ID, &a.GUID, &a.Filename, &a.MIMEType, &a.Size, &path); err != nil {
			continue
		}
		a.Path = expandTilde(path)
		a.MessageID = msgID
		out[msgID] = append(out[msgID], a)
	}
	return out, rows.Err()
}
