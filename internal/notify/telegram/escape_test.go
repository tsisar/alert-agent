package telegram

import (
	"strings"
	"testing"
)

func TestEscapeMarkdownV2_PlainText(t *testing.T) {
	input := "CPU usage is 95.3% (high)."
	want := `CPU usage is 95\.3% \(high\)\.`
	got := escapeMarkdownV2(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestEscapeMarkdownV2_Bold(t *testing.T) {
	input := "Status: **CRITICAL**"
	want := `Status: *CRITICAL*`
	got := escapeMarkdownV2(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestEscapeMarkdownV2_BoldWithSpecialChars(t *testing.T) {
	input := "**error rate (5.2%)**"
	want := `*error rate \(5\.2%\)*`
	got := escapeMarkdownV2(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestEscapeMarkdownV2_InlineCode(t *testing.T) {
	input := "Run `kubectl get pods` to check."
	want := "Run `kubectl get pods` to check\\."
	got := escapeMarkdownV2(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestEscapeMarkdownV2_CodeBlock(t *testing.T) {
	input := "Result:\n```\nup{job=\"grafana\"} 1\n```\nDone."
	want := "Result:\n```\nup{job=\"grafana\"} 1\n```\nDone\\."
	got := escapeMarkdownV2(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestEscapeMarkdownV2_Headers(t *testing.T) {
	input := "## Investigation Report\n- item 1\n- item 2"
	want := `\#\# Investigation Report` + "\n" + `\- item 1` + "\n" + `\- item 2`
	got := escapeMarkdownV2(input)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestEscapeMarkdownV2_MixedContent(t *testing.T) {
	input := "## Alert\n**Firing** service `grafana` error rate = 8.3%\n```\nrate(errors[5m])\n```"
	want := "\\#\\# Alert\n*Firing* service `grafana` error rate \\= 8\\.3%\n```\nrate(errors[5m])\n```"
	got := escapeMarkdownV2(input)
	if got != want {
		t.Fatalf("want:\n%s\ngot:\n%s", want, got)
	}
}

func TestSplitMessage_Short(t *testing.T) {
	chunks := splitMessage("short message", 4096)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestSplitMessage_ExactLimit(t *testing.T) {
	msg := strings.Repeat("a", 4096)
	chunks := splitMessage(msg, 4096)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestSplitMessage_SplitsOnNewline(t *testing.T) {
	// Build a message just over the limit with newlines
	part1 := strings.Repeat("line\n", 800) // 4000 chars
	part2 := strings.Repeat("more\n", 200) // 1000 chars
	msg := part1 + part2

	chunks := splitMessage(msg, 4096)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if len(c) > 4096 {
			t.Fatalf("chunk %d exceeds limit: %d chars", i, len(c))
		}
	}
}

func TestSplitMessage_NoNewlines(t *testing.T) {
	msg := strings.Repeat("x", 5000)
	chunks := splitMessage(msg, 4096)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 4096 {
		t.Fatalf("first chunk should be 4096, got %d", len(chunks[0]))
	}
	if len(chunks[1]) != 904 {
		t.Fatalf("second chunk should be 904, got %d", len(chunks[1]))
	}
}
