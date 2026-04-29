package imessage

import (
	"context"
	"fmt"
	"strings"
)

// CreateContactParams configures CreateContact.
type CreateContactParams struct {
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName,omitempty"`
	Company   string   `json:"company,omitempty"`
	Phones    []string `json:"phones,omitempty"`
	Emails    []string `json:"emails,omitempty"`
	Confirm   bool     `json:"confirm"`
}

// CreateContactResult is the return shape of CreateContact.
type CreateContactResult struct {
	OK        bool   `json:"ok"`
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName,omitempty"`
}

// CreateContact creates a new entry in macOS Contacts.app via JXA.
func (c *Client) CreateContact(ctx context.Context, p CreateContactParams) (*CreateContactResult, error) {
	if strings.TrimSpace(p.FirstName) == "" && strings.TrimSpace(p.LastName) == "" {
		return nil, fmt.Errorf("%w: firstName or lastName required", ErrInvalidParams)
	}
	if c.confirmSends && !p.Confirm {
		return nil, ErrConfirmRequired
	}
	out, err := c.runJXA(ctx, jsxCreateContact, map[string]any{
		"first_name": p.FirstName,
		"last_name":  p.LastName,
		"company":    p.Company,
		"phones":     p.Phones,
		"emails":     p.Emails,
	})
	if err != nil {
		return nil, err
	}
	return parseJXAResult[CreateContactResult](out)
}

// UpdateContactParams configures UpdateContact. ID is the Contacts.app
// unique identifier for the person to update.
type UpdateContactParams struct {
	ID        string   `json:"id"`
	FirstName *string  `json:"firstName,omitempty"`
	LastName  *string  `json:"lastName,omitempty"`
	Company   *string  `json:"company,omitempty"`
	AddPhones []string `json:"addPhones,omitempty"`
	AddEmails []string `json:"addEmails,omitempty"`
	Confirm   bool     `json:"confirm"`
}

// UpdateContactResult is the return shape of UpdateContact.
type UpdateContactResult struct {
	OK        bool   `json:"ok"`
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName,omitempty"`
}

// UpdateContact modifies an existing Contacts.app entry by ID.
func (c *Client) UpdateContact(ctx context.Context, p UpdateContactParams) (*UpdateContactResult, error) {
	if strings.TrimSpace(p.ID) == "" {
		return nil, fmt.Errorf("%w: id required", ErrInvalidParams)
	}
	if c.confirmSends && !p.Confirm {
		return nil, ErrConfirmRequired
	}
	out, err := c.runJXA(ctx, jsxUpdateContact, map[string]any{
		"id":         p.ID,
		"first_name": p.FirstName,
		"last_name":  p.LastName,
		"company":    p.Company,
		"add_phones": p.AddPhones,
		"add_emails": p.AddEmails,
	})
	if err != nil {
		return nil, err
	}
	return parseJXAResult[UpdateContactResult](out)
}

// DeleteContactParams configures DeleteContact.
type DeleteContactParams struct {
	ID      string `json:"id"`
	Confirm bool   `json:"confirm"`
}

// DeleteContactResult is the return shape of DeleteContact.
type DeleteContactResult struct {
	OK bool   `json:"ok"`
	ID string `json:"id"`
}

// DeleteContact removes a contact from Contacts.app by ID.
func (c *Client) DeleteContact(ctx context.Context, p DeleteContactParams) (*DeleteContactResult, error) {
	if strings.TrimSpace(p.ID) == "" {
		return nil, fmt.Errorf("%w: id required", ErrInvalidParams)
	}
	if c.confirmSends && !p.Confirm {
		return nil, ErrConfirmRequired
	}
	out, err := c.runJXA(ctx, jsxDeleteContact, map[string]any{
		"id": p.ID,
	})
	if err != nil {
		return nil, err
	}
	return parseJXAResult[DeleteContactResult](out)
}

// ListContactsParams configures ListContacts.
type ListContactsParams struct {
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// ContactEntry is a single entry returned by ListContacts.
type ContactEntry struct {
	ID        string   `json:"id"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName,omitempty"`
	Company   string   `json:"company,omitempty"`
	Phones    []string `json:"phones,omitempty"`
	Emails    []string `json:"emails,omitempty"`
}

// ListContactsResult is the return shape of ListContacts.
type ListContactsResult struct {
	Contacts []ContactEntry `json:"contacts"`
	Total    int            `json:"total"`
}

// ListContacts returns contacts from Contacts.app via JXA with optional
// name/company filter and pagination.
func (c *Client) ListContacts(ctx context.Context, p ListContactsParams) (*ListContactsResult, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	out, err := c.runJXA(ctx, jsxListContacts, map[string]any{
		"query":  strings.TrimSpace(p.Query),
		"limit":  limit,
		"offset": p.Offset,
	})
	if err != nil {
		return nil, err
	}
	return parseJXAResult[ListContactsResult](out)
}
