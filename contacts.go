package imessage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResolveContact searches the macOS Address Book for entries matching
// query (substring match against name; exact match against
// normalized phone or email). Returns deduplicated contacts.
func (c *Client) ResolveContact(ctx context.Context, query string) ([]Contact, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("%w: query is required", ErrInvalidParams)
	}
	files, err := c.findAddressBookDBs()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	qNorm := normalizeHandle(query)

	var out []Contact
	seen := map[string]int{} // canonical key -> index in out

	for _, f := range files {
		entries, err := readAddressBook(ctx, f)
		if err != nil {
			continue // tolerate per-source failures
		}
		for _, e := range entries {
			match := false
			if q != "" && (strings.Contains(strings.ToLower(e.Name), q) ||
				strings.Contains(strings.ToLower(e.Company), q)) {
				match = true
			}
			if !match {
				for _, p := range e.Phones {
					if normalizeHandle(p) == qNorm {
						match = true
						break
					}
				}
			}
			if !match {
				for _, em := range e.Emails {
					if strings.EqualFold(em, query) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
			key := strings.ToLower(e.Name) + "|" + e.Company
			if idx, ok := seen[key]; ok {
				out[idx].Phones = mergeUnique(out[idx].Phones, e.Phones)
				out[idx].Emails = mergeUnique(out[idx].Emails, e.Emails)
				continue
			}
			seen[key] = len(out)
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// IsAvailableOnIMessage reports whether the given handle has been seen
// communicating over iMessage in the local message history. This is the
// best signal Messages.app exposes locally without round-tripping Apple's
// Identity Services. When the handle has never been messaged, returns
// false with no error.
func (c *Client) IsAvailableOnIMessage(ctx context.Context, handle string) (bool, error) {
	if strings.TrimSpace(handle) == "" {
		return false, fmt.Errorf("%w: handle is required", ErrInvalidParams)
	}
	db, err := c.database(ctx)
	if err != nil {
		return false, err
	}
	norm := normalizeHandle(handle)
	const q = `
SELECT COUNT(*)
FROM handle
WHERE service = 'iMessage' AND (id = ? OR id = ? OR LOWER(id) = LOWER(?))`
	var n int
	if err := db.QueryRowContext(ctx, q, handle, norm, handle).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// findAddressBookDBs returns absolute paths to all AddressBook-v22.abcddb
// (or older v21) databases under the configured AddressBook sources dir.
func (c *Client) findAddressBookDBs() ([]string, error) {
	var out []string
	err := filepath.Walk(c.addressBookDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		base := info.Name()
		if strings.HasPrefix(base, "AddressBook-v") && strings.HasSuffix(base, ".abcddb") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return out, nil
}

func readAddressBook(ctx context.Context, path string) ([]Contact, error) {
	dsn := "file:" + path + "?mode=ro&_pragma=query_only(true)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	const q = `
SELECT
  COALESCE(r.ZFIRSTNAME, '') || ' ' || COALESCE(r.ZLASTNAME, '') AS name,
  COALESCE(r.ZORGANIZATION, '') AS org,
  GROUP_CONCAT(DISTINCT p.ZFULLNUMBER) AS phones,
  GROUP_CONCAT(DISTINCT e.ZADDRESSNORMALIZED) AS emails
FROM ZABCDRECORD r
LEFT JOIN ZABCDPHONENUMBER p ON p.ZOWNER = r.Z_PK
LEFT JOIN ZABCDEMAILADDRESS e ON e.ZOWNER = r.Z_PK
GROUP BY r.Z_PK`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Contact
	for rows.Next() {
		var name, org sql.NullString
		var phones, emails sql.NullString
		if err := rows.Scan(&name, &org, &phones, &emails); err != nil {
			continue
		}
		c := Contact{
			Name:    strings.TrimSpace(name.String),
			Company: strings.TrimSpace(org.String),
		}
		if phones.Valid && phones.String != "" {
			c.Phones = splitAndTrim(phones.String, ",")
		}
		if emails.Valid && emails.String != "" {
			c.Emails = splitAndTrim(emails.String, ",")
		}
		if c.Name == "" && len(c.Phones) == 0 && len(c.Emails) == 0 {
			continue
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// resolveContactsMap loads (and caches) a normalized handle -> name lookup
// across every AddressBook source. Used to enrich Message.SenderName and
// Handle.DisplayName.
func (c *Client) resolveContactsMap(ctx context.Context) map[string]string {
	c.contactsOnce.Do(func() {
		files, err := c.findAddressBookDBs()
		if err != nil {
			c.contactsErr = err
			return
		}
		m := map[string]string{}
		for _, f := range files {
			entries, err := readAddressBook(ctx, f)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.Name == "" {
					continue
				}
				for _, p := range e.Phones {
					m[normalizeHandle(p)] = e.Name
				}
				for _, em := range e.Emails {
					m[normalizeHandle(em)] = e.Name
				}
			}
		}
		c.contactsMap = m
	})
	return c.contactsMap
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func mergeUnique(a, b []string) []string {
	seen := map[string]struct{}{}
	for _, x := range a {
		seen[x] = struct{}{}
	}
	for _, x := range b {
		if _, ok := seen[x]; !ok {
			seen[x] = struct{}{}
			a = append(a, x)
		}
	}
	return a
}
