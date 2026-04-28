# imessage-go

A Go client for macOS iMessage and SMS via the local Messages.app SQLite
database and AppleScript bridge — plus a drop-in MCP tool surface
compatible with [`teslashibe/mcptool`](https://github.com/teslashibe/mcptool)
hosts (Cursor, Claude Desktop, Claude Code, etc.).

```go
import "github.com/teslashibe/imessage-go"
```

## Install

```bash
go get github.com/teslashibe/imessage-go
```

## Permissions (one-time, macOS)

| Permission | Where | Why |
|---|---|---|
| **Full Disk Access** | System Settings → Privacy & Security → Full Disk Access | Read `~/Library/Messages/chat.db` |
| **Automation → Messages** | System Settings → Privacy & Security → Automation | Drive Messages.app via osascript to send |
| **iPhone Text Message Forwarding** | iPhone → Settings → Messages → Text Message Forwarding | Send SMS to non-Apple recipients |

`imessage_status` is the first tool to call — it surfaces every missing
permission with copy-paste fix instructions.

## Quick start

```go
client := imessage.New(imessage.Config{},
    imessage.WithRequireConfirm(true),
    imessage.WithAllowedRecipients([]string{"+14155551212", "alice@example.com"}),
)
defer client.Close()

chats, _ := client.ListChats(ctx, imessage.ChatListParams{Limit: 10, OnlyUnread: true})
for _, c := range chats {
    msgs, _ := client.GetMessages(ctx, imessage.MessageListParams{ChatGUID: c.GUID, Limit: 5})
    for _, m := range msgs {
        fmt.Printf("%s [%s]: %s\n", m.SenderName, m.SentAt.Format(time.Kitchen), m.Text)
    }
}

// Auto-routes iMessage vs SMS based on local message history.
_ = client.SendMessage(ctx, imessage.SendParams{
    Recipient: "+14155551212",
    Body:      "shipped",
    Confirm:   true,
})
```

## Capability surface

### V1 (MVP)
| Method | Tool | Notes |
|---|---|---|
| `Status` | `imessage_status` | Permission preflight |
| `ListChats` | `imessage_list_chats` | iMessage-only by default; opt in to SMS |
| `GetMessages` | `imessage_get_messages` | Pagination via `BeforeID`; time bounds |
| `Search` | `imessage_search` | Substring match + chat/handle/time/direction filters; falls back to scanning `attributedBody` for newer messages with NULL text |
| `SendMessage` | `imessage_send_message` | Auto iMessage/SMS routing; `Confirm` + allowlist guards |
| `ResolveContact` | `imessage_resolve_contact` | Address Book lookup by name/phone/email |

### V1.0.2
| Method | Tool | Notes |
|---|---|---|
| `GetAttachment` | `imessage_get_attachment` | Returns base64 (capped by `WithMaxAttachmentBytes`, default 25 MiB) |
| `React` | `imessage_react` | Tapback. Currently sent as the corresponding emoji because Messages.app's scripting bridge does not expose true tapback objects — the response includes `"fallback": "emoji_text"` so the agent knows |
| `SendAttachment` | `imessage_send_attachment` | File attachment + optional caption |
| `IsAvailableOnIMessage` | `imessage_check_imessage` | Best-effort signal based on local history |

### V2 (stretch)
| Method | Tool | Notes |
|---|---|---|
| `Watch` | `imessage_watch` | Cursor-based polling. First call with `since_id=0` returns the current `MAX(ROWID)` as a bootstrap cursor; subsequent calls return new messages |
| (option) `WithAllowedRecipients` | n/a | Per-recipient send guard, also enforced against every participant of a target chat |

## Cursor MCP install

Add to `~/.cursor/mcp.json` (global) or your project's `.cursor/mcp.json`.
This package ships only the tool surface — host applications wrap it.
The simplest pattern is a one-binary host that registers all
`teslashibe/*-go` providers and serves stdio MCP. If you don't have such
a host yet, the `linkedin-go` and `nextdoor-go` packages in this org use
the same wiring; whatever host loads them will load this too.

```json
{
  "mcpServers": {
    "imessage": {
      "command": "your-mcp-host-binary"
    }
  }
}
```

## Drift prevention

`mcp/mcp_test.go` runs `mcptool.Coverage` over `*imessage.Client`. If a
new exported method is added to the client without either being wrapped
by an MCP tool or being added to `mcp.Excluded` with a one-line reason,
CI fails. The MCP surface stays in lockstep with the package API.

## Notes & limitations

- **macOS only.** All non-trivial methods return `ErrUnsupportedOS` on
  other OSes.
- **`attributedBody` decoder is heuristic.** Modern macOS stores message
  text inside an Apple typedstream BLOB. We scan for the first NSString
  primitive after the class table; this works for >95% of real messages
  but is not a full typedstream parser. Plain `text` column is preferred
  when present.
- **Tapbacks.** The Messages.app scripting bridge does not expose true
  tapback objects. `React` sends the corresponding emoji as a regular
  message and signals this via `"fallback": "emoji_text"`.
- **Sending requires Messages.app to be running.** The host process can
  launch it via `osascript -e 'tell application "Messages" to activate'`
  before the first send.

## License

MIT
