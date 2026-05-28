package telegram

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"
)

type Notifier struct {
	bot *tele.Bot
}

func New(token string) (*Notifier, error) {
	bot, err := tele.NewBot(tele.Settings{
		Token:   token,
		Offline: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	return &Notifier{bot: bot}, nil
}

const maxMessageLen = 4096

func (n *Notifier) SendMessage(_ context.Context, chatID string, text string) error {
	recipient, err := parseRecipient(chatID)
	if err != nil {
		return err
	}
	escaped := escapeMarkdownV2(text)
	chunks := splitMessage(escaped, maxMessageLen)
	for i, chunk := range chunks {
		_, err = n.bot.Send(recipient, chunk, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
		if err != nil {
			return fmt.Errorf("send telegram message (part %d/%d): %w", i+1, len(chunks), err)
		}
	}
	return nil
}

func (n *Notifier) SendPhoto(_ context.Context, chatID string, photo []byte, caption string) error {
	recipient, err := parseRecipient(chatID)
	if err != nil {
		return err
	}
	p := &tele.Photo{
		File:    tele.FromReader(bytes.NewReader(photo)),
		Caption: escapeMarkdownV2(caption),
	}
	_, err = n.bot.Send(recipient, p, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
	if err != nil {
		return fmt.Errorf("send telegram photo: %w", err)
	}
	return nil
}

func (n *Notifier) SendDocument(_ context.Context, chatID string, doc []byte, filename string, caption string) error {
	recipient, err := parseRecipient(chatID)
	if err != nil {
		return err
	}
	d := &tele.Document{
		File:     tele.FromReader(bytes.NewReader(doc)),
		FileName: filename,
		Caption:  escapeMarkdownV2(caption),
	}
	_, err = n.bot.Send(recipient, d, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
	if err != nil {
		return fmt.Errorf("send telegram document: %w", err)
	}
	return nil
}

func parseRecipient(chatID string) (tele.Recipient, error) {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID %q: %w", chatID, err)
	}
	return tele.ChatID(id), nil
}

// escapeMarkdownV2 converts standard Markdown (from LLM) to Telegram MarkdownV2.
// It preserves code blocks and inline code, and escapes special characters elsewhere.
func escapeMarkdownV2(text string) string {
	var b strings.Builder
	b.Grow(len(text) + len(text)/4)

	i := 0
	for i < len(text) {
		// Fenced code block: ```...```
		if i+2 < len(text) && text[i] == '`' && text[i+1] == '`' && text[i+2] == '`' {
			end := strings.Index(text[i+3:], "```")
			if end != -1 {
				block := text[i : i+3+end+3]
				b.WriteString(block)
				i += len(block)
				continue
			}
		}

		// Inline code: `...`
		if text[i] == '`' {
			end := strings.IndexByte(text[i+1:], '`')
			if end != -1 {
				span := text[i : i+1+end+1]
				b.WriteString(span)
				i += len(span)
				continue
			}
		}

		// Bold: **text** → *text*
		if i+1 < len(text) && text[i] == '*' && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				inner := escapeChars(text[i+2:i+2+end], boldSafe)
				b.WriteString("*")
				b.WriteString(inner)
				b.WriteString("*")
				i += 2 + end + 2
				continue
			}
		}

		// Escape special characters in plain text
		if isSpecialChar(text[i]) {
			b.WriteByte('\\')
		}
		b.WriteByte(text[i])
		i++
	}

	return b.String()
}

// Characters that must be escaped in MarkdownV2 plain text.
const specialChars = `_*[]()~>#+\-=|{}.!`

// boldSafe is the set of chars to escape inside bold spans (everything except *).
const boldSafe = `_[]()~>#+\-=|{}.!`

func isSpecialChar(c byte) bool {
	return strings.IndexByte(specialChars, c) >= 0
}

// splitMessage splits text into chunks that fit within maxLen,
// preferring to break at newline boundaries.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}

		cut := maxLen
		// Look for the last newline within the limit
		if idx := strings.LastIndex(text[:cut], "\n"); idx > 0 {
			cut = idx + 1
		}

		chunks = append(chunks, strings.TrimRight(text[:cut], "\n"))
		text = text[cut:]
	}
	return chunks
}

func escapeChars(s string, chars string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(chars, s[i]) >= 0 {
			b.WriteByte('\\')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
