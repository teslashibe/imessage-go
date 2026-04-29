package imessage

import "time"

// Service identifies the underlying delivery channel for a message.
type Service string

const (
	ServiceIMessage Service = "iMessage"
	ServiceSMS      Service = "SMS"
)

// Reaction enumerates iMessage tapback types.
type Reaction string

const (
	ReactionLove        Reaction = "love"
	ReactionLike        Reaction = "like"
	ReactionDislike     Reaction = "dislike"
	ReactionLaugh       Reaction = "laugh"
	ReactionEmphasize   Reaction = "emphasize"
	ReactionQuestion    Reaction = "question"
	ReactionRemoveLove  Reaction = "remove_love"
	ReactionRemoveLike  Reaction = "remove_like"
	ReactionRemoveOther Reaction = "remove_other"
)

// Chat is a conversation thread (one-on-one or group).
type Chat struct {
	ID             int64     `json:"id"`               // ROWID in chat table
	GUID           string    `json:"guid"`             // chat.guid (used for sending)
	DisplayName    string    `json:"displayName"`      // group chat name, may be empty
	Service        Service   `json:"service"`          // iMessage / SMS
	IsGroup        bool      `json:"isGroup"`
	Participants   []Handle  `json:"participants,omitempty"`
	LastActivityAt time.Time `json:"lastActivityAt,omitempty"`
	UnreadCount    int       `json:"unreadCount,omitempty"`
}

// Handle is a participant identifier (phone number or email).
type Handle struct {
	ID          int64   `json:"id"`               // ROWID in handle table
	Identifier  string  `json:"identifier"`       // raw value: +14155551212 or alice@example.com
	Service     Service `json:"service"`
	DisplayName string  `json:"displayName,omitempty"` // resolved from Address Book
}

// Message is a single message in a chat.
type Message struct {
	ID            int64       `json:"id"`     // ROWID in message table
	GUID          string      `json:"guid"`
	ChatID        int64       `json:"chatId"`
	ChatGUID      string      `json:"chatGuid,omitempty"`
	Text          string      `json:"text"`
	SenderHandle  string      `json:"senderHandle,omitempty"` // empty when IsFromMe
	SenderName    string      `json:"senderName,omitempty"`
	IsFromMe      bool        `json:"isFromMe"`
	Service       Service     `json:"service"`
	SentAt        time.Time   `json:"sentAt"`
	ReadAt        time.Time   `json:"readAt,omitempty"`
	DeliveredAt   time.Time   `json:"deliveredAt,omitempty"`
	IsRead        bool        `json:"isRead"`
	IsDelivered   bool        `json:"isDelivered"`
	IsReply       bool        `json:"isReply,omitempty"`
	ReplyToGUID   string      `json:"replyToGuid,omitempty"`
	IsTapback     bool        `json:"isTapback,omitempty"`
	TapbackKind   Reaction    `json:"tapbackKind,omitempty"`
	TapbackTarget string      `json:"tapbackTargetGuid,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`
}

// Attachment is a single attached file referenced by a message. Data is
// only populated by GetAttachment.
type Attachment struct {
	ID        int64  `json:"id"`     // ROWID in attachment table
	GUID      string `json:"guid"`
	MessageID int64  `json:"messageId,omitempty"`
	Filename  string `json:"filename"`
	MIMEType  string `json:"mimeType,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Path      string `json:"path,omitempty"`     // absolute path on disk
	DataB64   string `json:"dataBase64,omitempty"` // populated by GetAttachment
}

// Contact is a resolved Address Book entry.
type Contact struct {
	Name    string   `json:"name"`
	Phones  []string `json:"phones,omitempty"`
	Emails  []string `json:"emails,omitempty"`
	Company string   `json:"company,omitempty"`
}

// StatusReport summarises whether the host process can read chat.db and
// drive Messages.app.
type StatusReport struct {
	OS                   string `json:"os"`
	ChatDBPath           string `json:"chatDbPath"`
	ChatDBReadable       bool   `json:"chatDbReadable"`
	ChatDBError          string `json:"chatDbError,omitempty"`
	AddressBookReadable  bool   `json:"addressBookReadable"`
	AddressBookError     string `json:"addressBookError,omitempty"`
	OsascriptPresent     bool   `json:"osascriptPresent"`
	MessagesAppRunning   bool   `json:"messagesAppRunning"`
	AutomationGranted    bool   `json:"automationGranted"`
	AutomationError      string `json:"automationError,omitempty"`
	HelpFullDiskAccess   string `json:"helpFullDiskAccess,omitempty"`
	HelpAutomation       string `json:"helpAutomation,omitempty"`
}

// --- Param structs ---

// ChatListParams configures ListChats.
type ChatListParams struct {
	Limit       int    // default 20
	Offset      int
	IncludeSMS  bool   // default false (iMessage only)
	OnlyUnread  bool
	HandleQuery string // optional: filter chats containing this handle/name
}

// MessageListParams configures GetMessages.
type MessageListParams struct {
	ChatID    int64  // one of ChatID, ChatGUID, or HandleID is required
	ChatGUID  string
	HandleID  int64
	Limit     int    // default 50, max 500
	BeforeID  int64  // pagination: only messages with ROWID < BeforeID
	Since     time.Time
	Until     time.Time
}

// SearchParams configures Search.
type SearchParams struct {
	Query    string    // required
	ChatID   int64     // optional filter
	Handle   string    // optional filter (phone/email)
	Since    time.Time
	Until    time.Time
	Limit    int       // default 50, max 500
	Offset   int
	FromMe   *bool     // nil = both, true = only mine, false = only theirs
}

// SendParams configures SendMessage. Provide one of ChatGUID or Recipient.
type SendParams struct {
	ChatGUID  string  // existing chat.guid (preferred for groups)
	Recipient string  // single phone/email — auto-routed iMessage/SMS
	Body      string  // required, plain text
	Service   Service // optional override; default = auto
	Confirm   bool    // required when WithRequireConfirm is true
}

// SendAttachmentParams configures SendAttachment.
type SendAttachmentParams struct {
	ChatGUID  string
	Recipient string
	FilePath  string  // absolute path to a file on disk
	Caption   string  // optional message body sent alongside
	Service   Service
	Confirm   bool
}

// ReactParams configures React.
type ReactParams struct {
	ChatGUID    string   // required
	MessageGUID string   // required: target message GUID
	Kind        Reaction // required
	Confirm     bool
}

// WatchParams configures Watch.
type WatchParams struct {
	SinceID int64  // return messages with ROWID > SinceID
	ChatID  int64  // optional chat filter
	Limit   int    // default 100
}

// WatchResult is the return shape for Watch — items plus a cursor to feed
// back into the next call.
type WatchResult struct {
	Messages []Message `json:"messages"`
	Cursor   int64     `json:"cursor"` // pass back as SinceID next call
}
