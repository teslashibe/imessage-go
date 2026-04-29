package mcp

import (
	"context"

	imessage "github.com/teslashibe/imessage-go"
	"github.com/teslashibe/mcptool"
)

// ResolveContactInput is the typed input for imessage_resolve_contact.
type ResolveContactInput struct {
	Query string `json:"query" jsonschema:"description=name substring or exact phone/email to match in macOS Contacts,required"`
}

func resolveContact(ctx context.Context, c *imessage.Client, in ResolveContactInput) (any, error) {
	return c.ResolveContact(ctx, in.Query)
}

// CheckIMessageInput is the typed input for imessage_check_imessage.
type CheckIMessageInput struct {
	Handle string `json:"handle" jsonschema:"description=phone (E.164) or email to check for prior iMessage activity,required"`
}

func checkIMessage(ctx context.Context, c *imessage.Client, in CheckIMessageInput) (any, error) {
	ok, err := c.IsAvailableOnIMessage(ctx, in.Handle)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"handle":     in.Handle,
		"normalized": imessage.NormalizeHandle(in.Handle),
		"available":  ok,
	}, nil
}

// ListContactsInput is the typed input for imessage_list_contacts.
type ListContactsInput struct {
	Query  string `json:"query,omitempty" jsonschema:"description=optional filter: name or company substring"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=max contacts to return,minimum=1,maximum=500,default=50"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=pagination offset,minimum=0,default=0"`
}

func listContacts(ctx context.Context, c *imessage.Client, in ListContactsInput) (any, error) {
	return c.ListContacts(ctx, imessage.ListContactsParams{
		Query:  in.Query,
		Limit:  in.Limit,
		Offset: in.Offset,
	})
}

// CreateContactInput is the typed input for imessage_create_contact.
type CreateContactInput struct {
	FirstName string   `json:"firstName" jsonschema:"description=first name (at least firstName or lastName required),required"`
	LastName  string   `json:"lastName,omitempty" jsonschema:"description=last name"`
	Company   string   `json:"company,omitempty" jsonschema:"description=company or organization"`
	Phones    []string `json:"phones,omitempty" jsonschema:"description=phone numbers to add (E.164 preferred)"`
	Emails    []string `json:"emails,omitempty" jsonschema:"description=email addresses to add"`
	Confirm   bool     `json:"confirm" jsonschema:"description=must be true to execute the write,required"`
}

func createContact(ctx context.Context, c *imessage.Client, in CreateContactInput) (any, error) {
	return c.CreateContact(ctx, imessage.CreateContactParams{
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Company:   in.Company,
		Phones:    in.Phones,
		Emails:    in.Emails,
		Confirm:   in.Confirm,
	})
}

// UpdateContactInput is the typed input for imessage_update_contact.
type UpdateContactInput struct {
	ID        string   `json:"id" jsonschema:"description=Contacts.app unique ID of the person to update (from list_contacts),required"`
	FirstName *string  `json:"firstName,omitempty" jsonschema:"description=new first name (null to leave unchanged)"`
	LastName  *string  `json:"lastName,omitempty" jsonschema:"description=new last name (null to leave unchanged)"`
	Company   *string  `json:"company,omitempty" jsonschema:"description=new company (null to leave unchanged)"`
	AddPhones []string `json:"addPhones,omitempty" jsonschema:"description=phone numbers to add to the contact"`
	AddEmails []string `json:"addEmails,omitempty" jsonschema:"description=email addresses to add to the contact"`
	Confirm   bool     `json:"confirm" jsonschema:"description=must be true to execute the write,required"`
}

func updateContact(ctx context.Context, c *imessage.Client, in UpdateContactInput) (any, error) {
	return c.UpdateContact(ctx, imessage.UpdateContactParams{
		ID:        in.ID,
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Company:   in.Company,
		AddPhones: in.AddPhones,
		AddEmails: in.AddEmails,
		Confirm:   in.Confirm,
	})
}

// DeleteContactInput is the typed input for imessage_delete_contact.
type DeleteContactInput struct {
	ID      string `json:"id" jsonschema:"description=Contacts.app unique ID of the person to delete (from list_contacts),required"`
	Confirm bool   `json:"confirm" jsonschema:"description=must be true to execute the deletion,required"`
}

func deleteContact(ctx context.Context, c *imessage.Client, in DeleteContactInput) (any, error) {
	return c.DeleteContact(ctx, imessage.DeleteContactParams{
		ID:      in.ID,
		Confirm: in.Confirm,
	})
}

var contactTools = []mcptool.Tool{
	mcptool.Define[*imessage.Client, ResolveContactInput](
		"imessage_resolve_contact",
		"Resolve a name or handle to one or more macOS AddressBook contacts (with phones and emails)",
		"ResolveContact",
		resolveContact,
	),
	mcptool.Define[*imessage.Client, CheckIMessageInput](
		"imessage_check_imessage",
		"Check whether a handle has prior iMessage history locally (best-effort signal of iMessage availability)",
		"IsAvailableOnIMessage",
		checkIMessage,
	),
	mcptool.Define[*imessage.Client, ListContactsInput](
		"imessage_list_contacts",
		"List contacts from macOS Contacts.app with optional name/company filter and pagination",
		"ListContacts",
		listContacts,
	),
	mcptool.Define[*imessage.Client, CreateContactInput](
		"imessage_create_contact",
		"Create a new contact in macOS Contacts.app with name, phones, and emails",
		"CreateContact",
		createContact,
	),
	mcptool.Define[*imessage.Client, UpdateContactInput](
		"imessage_update_contact",
		"Update an existing contact in macOS Contacts.app (change name/company, add phones/emails)",
		"UpdateContact",
		updateContact,
	),
	mcptool.Define[*imessage.Client, DeleteContactInput](
		"imessage_delete_contact",
		"Delete a contact from macOS Contacts.app by ID",
		"DeleteContact",
		deleteContact,
	),
}
