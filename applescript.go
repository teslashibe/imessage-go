package imessage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// runJXA runs a JavaScript-for-Automation script via osascript with the
// supplied input passed as the JSON string assigned to a global `INPUT`
// variable. Returns stdout (trimmed). Args are never interpolated into the
// script body — all dynamic data goes through INPUT.
func (c *Client) runJXA(ctx context.Context, script string, input any) (string, error) {
	payload, err := marshalForJS(input)
	if err != nil {
		return "", err
	}
	full := fmt.Sprintf("var INPUT = %s;\n%s", payload, script)
	cmd := exec.CommandContext(ctx, c.osascriptPath, "-l", "JavaScript", "-e", full)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s := string(out)
		if isAutomationDenied(s) {
			return "", fmt.Errorf("%w: %s", ErrAutomationPermission, strings.TrimSpace(s))
		}
		return "", fmt.Errorf("%w: %v: %s", ErrJXAFailed, err, strings.TrimSpace(s))
	}
	return strings.TrimSpace(string(out)), nil
}

// marshalForJS encodes v as JSON suitable for direct embedding inside a
// JavaScript source file. Go's json.Marshal does not escape U+2028 (LINE
// SEPARATOR) or U+2029 (PARAGRAPH SEPARATOR); these are valid JSON string
// bytes but terminate JS string literals, so an unsanitized message body
// containing them silently breaks the JXA script. We post-process the
// marshaled bytes to escape them.
func marshalForJS(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	b = bytes.ReplaceAll(b, []byte("\u2028"), []byte(`\u2028`))
	b = bytes.ReplaceAll(b, []byte("\u2029"), []byte(`\u2029`))
	return string(b), nil
}

func isAutomationDenied(s string) bool {
	for _, m := range []string{"-1743", "Not authorized to send Apple events", "User canceled"} {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

// jsxSendToBuddy targets a specific buddy (handle) or existing chat.
// service may be "iMessage" or "SMS" (or "" for auto). When chatGuid is
// non-empty, sends to that chat (1:1 OR group); otherwise resolves
// recipient against the local chat list.
//
// macOS Tahoe (26+) removed services/buddies entirely AND requires the
// "any;" prefix on chat IDs. SQLite stores GUIDs as "iMessage;+;chat..."
// but Messages.app only accepts "any;+;chat..." on Tahoe. We try both
// the raw GUID and a "any;"-rewritten variant for compatibility.
const jsxSendToBuddy = `
(function() {
  var Messages = Application('Messages');
  Messages.includeStandardAdditions = true;
  var body = INPUT.body || '';
  var attachment = INPUT.attachment_path || '';
  var chatGuid = INPUT.chat_guid || '';
  var recipient = INPUT.recipient || '';
  var service = INPUT.service || '';

  function chatGuidVariants(raw) {
    var variants = [raw];
    // SQLite-style "iMessage;+;chat..." or "iMessage;-;+phone" → "any;..."
    var m = raw.match(/^(iMessage|SMS);([+-]);(.+)$/);
    if (m) {
      variants.push('any;' + m[2] + ';' + m[3]);
    } else if (raw.indexOf(';') === -1) {
      // Bare "chat..." identifier: prepend any;+;
      variants.push('any;+;' + raw);
    }
    return variants;
  }

  function findChatByGuid(raw) {
    var variants = chatGuidVariants(raw);
    for (var i = 0; i < variants.length; i++) {
      var found = Messages.chats.whose({ id: variants[i] });
      if (found.length > 0) return found[0];
    }
    return null;
  }

  function findChatByRecipient(rcpt, svc) {
    var prefixes = [];
    if (svc === 'SMS') {
      prefixes = ['SMS;-;', 'any;-;'];
    } else if (svc === 'iMessage') {
      prefixes = ['iMessage;-;', 'any;-;'];
    } else {
      prefixes = ['any;-;', 'iMessage;-;', 'SMS;-;'];
    }
    for (var i = 0; i < prefixes.length; i++) {
      var id = prefixes[i] + rcpt;
      var found = Messages.chats.whose({ id: id });
      if (found.length > 0) return found[0];
    }
    return null;
  }

  function findServicesBuddy(rcpt, svc) {
    try {
      var all = Messages.services();
      var wantType = (svc === 'SMS') ? 'SMS' : 'iMessage';
      var matched = [];
      for (var i = 0; i < all.length; i++) {
        if (all[i].serviceType() === wantType) matched.push(all[i]);
      }
      if (matched.length === 0 && svc === '') {
        for (var i = 0; i < all.length; i++) {
          if (all[i].serviceType() === 'SMS') matched.push(all[i]);
        }
      }
      if (matched.length > 0) return matched[0].buddies.byName(rcpt);
    } catch(e) {}
    return null;
  }

  var target = null;
  if (chatGuid) {
    target = findChatByGuid(chatGuid);
    if (!target) throw new Error('chat not found: ' + chatGuid + ' (tried raw + any;-prefixed variants)');
  } else if (recipient) {
    target = findChatByRecipient(recipient, service);
    if (!target) {
      target = findServicesBuddy(recipient, service);
    }
    if (!target) {
      throw new Error('no chat found for ' + recipient + ' — send a message from Messages.app first to establish the thread, then retry');
    }
  } else {
    throw new Error('chat_guid or recipient required');
  }

  if (attachment) {
    Messages.send(Path(attachment), { to: target });
  }
  if (body) {
    Messages.send(body, { to: target });
  }
  return JSON.stringify({ ok: true });
})();
`

// jsxReact issues a tapback against an existing message. Tapbacks are not
// well exposed via Messages.app's scripting bridge in current macOS;
// when unsupported we fall back to a textual indicator (e.g. "❤️").
//
// Uses the same chat-GUID variant lookup as jsxSendToBuddy so SQLite-style
// "iMessage;+;chat..." GUIDs work on macOS Tahoe (which only accepts
// "any;+;chat...").
const jsxReact = `
(function() {
  var Messages = Application('Messages');
  var chatGuid = INPUT.chat_guid;
  var emoji = INPUT.emoji || '❤️';

  function chatGuidVariants(raw) {
    var variants = [raw];
    var m = raw.match(/^(iMessage|SMS);([+-]);(.+)$/);
    if (m) {
      variants.push('any;' + m[2] + ';' + m[3]);
    } else if (raw.indexOf(';') === -1) {
      variants.push('any;+;' + raw);
    }
    return variants;
  }

  var target = null;
  var variants = chatGuidVariants(chatGuid);
  for (var i = 0; i < variants.length; i++) {
    var found = Messages.chats.whose({ id: variants[i] });
    if (found.length > 0) { target = found[0]; break; }
  }
  if (!target) throw new Error('chat not found: ' + chatGuid);
  Messages.send(emoji, { to: target });
  return JSON.stringify({ ok: true, fallback: 'emoji_text' });
})();
`
