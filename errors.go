package imessage

import "errors"

var (
	ErrUnsupportedOS         = errors.New("imessage: macOS-only operation")
	ErrChatDBNotFound        = errors.New("imessage: chat.db not found at expected path")
	ErrChatDBPermission      = errors.New("imessage: cannot read chat.db (grant Full Disk Access to your terminal/IDE)")
	ErrAutomationPermission  = errors.New("imessage: Messages.app automation denied (grant in System Settings > Privacy & Security > Automation)")
	ErrInvalidParams         = errors.New("imessage: invalid parameters")
	ErrNotFound              = errors.New("imessage: not found")
	ErrConfirmRequired       = errors.New("imessage: confirm=true required for send-style operations")
	ErrRecipientNotAllowed   = errors.New("imessage: recipient is not in the configured allowlist")
	ErrMessageEmpty          = errors.New("imessage: message body is empty")
	ErrAttachmentTooLarge    = errors.New("imessage: attachment exceeds configured size cap")
	ErrAttachmentMissingFile = errors.New("imessage: attachment file is missing on disk")
	ErrSendFailed            = errors.New("imessage: send via Messages.app failed")
	ErrJXAFailed             = errors.New("imessage: JXA bridge call failed")
	ErrParseFailed           = errors.New("imessage: failed to parse database row")
	ErrUnknownService        = errors.New("imessage: unknown service (must be iMessage or SMS)")
)
