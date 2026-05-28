package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Notifier delivers alert reports to a Slack channel via the Slack Web API.
// It uses a Bot Token for sending messages and uploading files.
type Notifier struct {
	botToken string
	client   *http.Client
}

const (
	contentType   = "Content-Type"
	authorization = "Authorization"
)

func New(botToken string) *Notifier {
	return &Notifier{
		botToken: botToken,
		client:   &http.Client{},
	}
}

// Slack message text limit (approximately 40 000 characters).
const maxTextLen = 40000

func (n *Notifier) SendMessage(ctx context.Context, channel string, text string) error {
	converted := ConvertMarkdown(stripDataURIs(text))
	chunks := splitMessage(converted, maxTextLen)
	for i, chunk := range chunks {
		if err := n.postMessage(ctx, channel, chunk); err != nil {
			return fmt.Errorf("send slack message (part %d/%d): %w", i+1, len(chunks), err)
		}
	}
	return nil
}

func (n *Notifier) SendPhoto(ctx context.Context, channel string, photo []byte, caption string) error {
	return n.uploadFile(ctx, channel, photo, "screenshot.png", caption)
}

func (n *Notifier) SendDocument(ctx context.Context, channel string, doc []byte, filename string, caption string) error {
	return n.uploadFile(ctx, channel, doc, filename, caption)
}

// postMessage sends a text message to a channel using chat.postMessage.
func (n *Notifier) postMessage(ctx context.Context, channel, text string) error {
	payload := map[string]string{
		"channel": channel,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set(contentType, "application/json; charset=utf-8")
	req.Header.Set(authorization, "Bearer "+n.botToken)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to slack: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	return checkSlackResponse(resp)
}

// uploadFile uploads a file to a Slack channel using the new upload flow:
// 1. files.getUploadURLExternal — get a presigned upload URL
// 2. POST file content to the presigned URL
// 3. files.completeUploadExternal — finalize and share to the channel
func (n *Notifier) uploadFile(ctx context.Context, channel string, data []byte, filename, caption string) error {
	// Step 1: get upload URL
	form := fmt.Sprintf("filename=%s&length=%d", filename, len(data))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/files.getUploadURLExternal", strings.NewReader(form))
	if err != nil {
		return fmt.Errorf("create getUploadURLExternal request: %w", err)
	}
	req.Header.Set(contentType, "application/x-www-form-urlencoded")
	req.Header.Set(authorization, "Bearer "+n.botToken)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("getUploadURLExternal: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return fmt.Errorf("read getUploadURLExternal response: %w", err)
	}
	var uploadResp struct {
		OK        bool   `json:"ok"`
		Error     string `json:"error"`
		UploadURL string `json:"upload_url"`
		FileID    string `json:"file_id"`
	}
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return fmt.Errorf("parse getUploadURLExternal response: %w", err)
	}
	if !uploadResp.OK {
		return fmt.Errorf("getUploadURLExternal error: %s", uploadResp.Error)
	}

	// Step 2: upload file content to the presigned URL
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadResp.UploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	uploadReq.Header.Set(contentType, "application/octet-stream")

	uploadHTTPResp, err := n.client.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("upload file content: %w", err)
	}
	defer uploadHTTPResp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, uploadHTTPResp.Body)

	if uploadHTTPResp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload returned status %d", uploadHTTPResp.StatusCode)
	}

	// Step 3: complete upload and share to channel
	completePayload := map[string]any{
		"files": []map[string]string{
			{"id": uploadResp.FileID, "title": filename},
		},
		"channel_id": channel,
	}
	if caption != "" {
		completePayload["initial_comment"] = ConvertMarkdown(stripDataURIs(caption))
	}

	completeBody, err := json.Marshal(completePayload)
	if err != nil {
		return fmt.Errorf("marshal completeUploadExternal: %w", err)
	}

	completeReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/files.completeUploadExternal", bytes.NewReader(completeBody))
	if err != nil {
		return fmt.Errorf("create completeUploadExternal request: %w", err)
	}
	completeReq.Header.Set(contentType, "application/json; charset=utf-8")
	completeReq.Header.Set(authorization, "Bearer "+n.botToken)

	completeResp, err := n.client.Do(completeReq)
	if err != nil {
		return fmt.Errorf("completeUploadExternal: %w", err)
	}
	defer completeResp.Body.Close() //nolint:errcheck

	return checkSlackResponse(completeResp)
}

type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func checkSlackResponse(resp *http.Response) error {
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return fmt.Errorf("read slack response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned %d: %s", resp.StatusCode, string(respBody))
	}
	if len(respBody) == 0 {
		return nil
	}
	var sr slackResponse
	if err := json.Unmarshal(respBody, &sr); err != nil {
		return fmt.Errorf("parse slack response: %w", err)
	}
	if !sr.OK {
		return fmt.Errorf("slack API error: %s", sr.Error)
	}
	return nil
}

// dataURILinkRe matches Markdown image/link references with inline data: URIs,
// e.g. [Image](data:image/png;base64,iVBOR...) and replaces them with just [Image].
var dataURILinkRe = regexp.MustCompile(`\[([^\]]*)\]\(data:[^)]+\)`)

// stripDataURIs removes inline data: URI links (base64-encoded images) from text.
func stripDataURIs(text string) string {
	return dataURILinkRe.ReplaceAllString(text, "[$1]")
}

// ConvertMarkdown transforms standard Markdown (from LLM output) into Slack
// mrkdwn format. It preserves code blocks and inline code, converts **bold**
// to *bold*, and rewrites [text](url) links to <url|text>.
func ConvertMarkdown(text string) string {
	var b strings.Builder
	b.Grow(len(text))

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

		// Bold: **text** -> *text*
		if i+1 < len(text) && text[i] == '*' && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				b.WriteByte('*')
				b.WriteString(text[i+2 : i+2+end])
				b.WriteByte('*')
				i += 2 + end + 2
				continue
			}
		}

		// Markdown link: [text](url) -> <url|text>
		if text[i] == '[' {
			if linkLen := parseMarkdownLink(&b, text[i:]); linkLen > 0 {
				i += linkLen
				continue
			}
		}

		b.WriteByte(text[i])
		i++
	}

	return b.String()
}

// parseMarkdownLink tries to parse a [text](url) link at the beginning of s.
// On success it writes <url|text> into b and returns the number of bytes consumed.
// Returns 0 if s does not start with a valid Markdown link.
func parseMarkdownLink(b *strings.Builder, s string) int {
	closeBracket := strings.IndexByte(s[1:], ']')
	if closeBracket < 0 {
		return 0
	}
	afterBracket := 1 + closeBracket + 1
	if afterBracket >= len(s) || s[afterBracket] != '(' {
		return 0
	}
	closeParen := strings.IndexByte(s[afterBracket+1:], ')')
	if closeParen < 0 {
		return 0
	}
	linkText := s[1 : 1+closeBracket]
	url := s[afterBracket+1 : afterBracket+1+closeParen]
	b.WriteByte('<')
	b.WriteString(url)
	b.WriteByte('|')
	b.WriteString(linkText)
	b.WriteByte('>')
	return afterBracket + 1 + closeParen + 1
}

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
		if idx := strings.LastIndex(text[:cut], "\n"); idx > 0 {
			cut = idx + 1
		}

		chunks = append(chunks, strings.TrimRight(text[:cut], "\n"))
		text = text[cut:]
	}
	return chunks
}
