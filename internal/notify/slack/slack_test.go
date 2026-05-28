package slack

import (
	"strings"
	"testing"
)

func TestConvertMarkdown_Bold(t *testing.T) {
	input := "Status: **CRITICAL**"
	want := "Status: *CRITICAL*"
	got := ConvertMarkdown(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestConvertMarkdown_Link(t *testing.T) {
	input := "See [dashboard](https://grafana.example.com/d/abc) for details."
	want := "See <https://grafana.example.com/d/abc|dashboard> for details."
	got := ConvertMarkdown(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestConvertMarkdown_BoldAndLink(t *testing.T) {
	input := "**Alert**: check [panel](http://grafana/d/123?panelId=5)"
	want := "*Alert*: check <http://grafana/d/123?panelId=5|panel>"
	got := ConvertMarkdown(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestConvertMarkdown_InlineCode(t *testing.T) {
	input := "Run `kubectl get pods` to check."
	want := "Run `kubectl get pods` to check."
	got := ConvertMarkdown(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestConvertMarkdown_CodeBlock(t *testing.T) {
	input := "Result:\n```\nup{job=\"grafana\"} 1\n```\nDone."
	want := input
	got := ConvertMarkdown(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestConvertMarkdown_PlainText(t *testing.T) {
	input := "CPU usage is 95.3% on web-1."
	got := ConvertMarkdown(input)
	if got != input {
		t.Fatalf("plain text should be unchanged, got %q", got)
	}
}

func TestConvertMarkdown_MixedContent(t *testing.T) {
	input := "## Alert\n**Firing** service `grafana` error [link](http://example.com)\n```\nrate(errors[5m])\n```"
	want := "## Alert\n*Firing* service `grafana` error <http://example.com|link>\n```\nrate(errors[5m])\n```"
	got := ConvertMarkdown(input)
	if got != want {
		t.Fatalf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestConvertMarkdown_BracketNotLink(t *testing.T) {
	input := "array[0] = value"
	got := ConvertMarkdown(input)
	if got != input {
		t.Fatalf("non-link brackets should be unchanged, got %q", got)
	}
}

func TestStripDataURIs(t *testing.T) {
	input := "#### Panel Screenshots\n- **S3 Bucket Size (Panel 93)**: [Image](data:image/png;base64,/9j/4AAQSkZJRgABAQ" + strings.Repeat("A", 5000) + ")\n- **Host Disk Space (Panel 97)**: [Chart](data:image/jpeg;base64,abc123)\n\nDone."
	want := "#### Panel Screenshots\n- **S3 Bucket Size (Panel 93)**: [Image]\n- **Host Disk Space (Panel 97)**: [Chart]\n\nDone."
	got := stripDataURIs(input)
	if got != want {
		t.Fatalf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestStripDataURIs_NoMatch(t *testing.T) {
	input := "See [dashboard](https://grafana.example.com/d/abc) for details."
	got := stripDataURIs(input)
	if got != input {
		t.Fatalf("regular links should be unchanged, got %q", got)
	}
}

func TestSplitMessage_Short(t *testing.T) {
	chunks := splitMessage("short message", maxTextLen)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestSplitMessage_SplitsOnNewline(t *testing.T) {
	part1 := strings.Repeat("line\n", 8000)
	part2 := strings.Repeat("more\n", 2000)
	msg := part1 + part2

	chunks := splitMessage(msg, maxTextLen)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if len(c) > maxTextLen {
			t.Fatalf("chunk %d exceeds limit: %d chars", i, len(c))
		}
	}
}
