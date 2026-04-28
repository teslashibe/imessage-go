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
		return "", fmt.Errorf("%w: %v: %s", ErrSendFailed, err, strings.TrimSpace(s))
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

// jsxSendToBuddy targets a specific buddy (handle) and lets Messages.app
// pick the right service. service may be "iMessage" or "SMS" (or "" for
// auto). When chatGuid is non-empty, sends to the existing chat directly.
const jsxSendToBuddy = `
(function() {
  var Messages = Application('Messages');
  Messages.includeStandardAdditions = true;
  var body = INPUT.body || '';
  var attachment = INPUT.attachment_path || '';
  var chatGuid = INPUT.chat_guid || '';
  var recipient = INPUT.recipient || '';
  var service = INPUT.service || '';

  var target = null;
  if (chatGuid) {
    var chats = Messages.chats.whose({ id: chatGuid });
    if (chats.length === 0) throw new Error('chat not found: ' + chatGuid);
    target = chats[0];
  } else if (recipient) {
    var svcs;
    if (service === 'SMS') {
      svcs = Messages.services.whose({ serviceType: 'SMS' });
    } else if (service === 'iMessage') {
      svcs = Messages.services.whose({ serviceType: 'iMessage' });
    } else {
      svcs = Messages.services.whose({ serviceType: 'iMessage' });
      if (svcs.length === 0) {
        svcs = Messages.services.whose({ serviceType: 'SMS' });
      }
    }
    if (svcs.length === 0) throw new Error('no enabled service for ' + (service || 'auto'));
    target = svcs[0].buddies.byName(recipient);
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
const jsxReact = `
(function() {
  var Messages = Application('Messages');
  var chatGuid = INPUT.chat_guid;
  var emoji = INPUT.emoji || '❤️';
  var chats = Messages.chats.whose({ id: chatGuid });
  if (chats.length === 0) throw new Error('chat not found: ' + chatGuid);
  Messages.send(emoji, { to: chats[0] });
  return JSON.stringify({ ok: true, fallback: 'emoji_text' });
})();
`
