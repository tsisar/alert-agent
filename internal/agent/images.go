package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"

	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/extended-log-go/log"
)

// Notes appended to a tool result that returned an image, so the model knows
// the capture is kept and does not ask for the same panel again. Which one is
// used depends on whether the scenario delivers images at all.
const (
	imageNoteAttached = "\nImage captured; it is attached to the report automatically. Do not request the same image again or embed it in report text."
	imageNoteNotSent  = "\nImage captured, but this scenario does not send images, so it will not be delivered. Do not request the same image again or embed it in report text."
)

// imageExecutor belongs to one investigation and is used by its sequential tool
// loop. Cache only panel renders, never arbitrary tools that may have side effects
// or metric queries whose results may change during the investigation.
type imageExecutor struct {
	ToolExecutor
	note   string
	panels map[string]string
	seen   map[[sha256.Size]byte]struct{}
}

func newImageExecutor(executor ToolExecutor, sendImages bool) *imageExecutor {
	note := imageNoteNotSent
	if sendImages {
		note = imageNoteAttached
	}
	return &imageExecutor{
		ToolExecutor: executor,
		note:         note,
		panels:       make(map[string]string),
		seen:         make(map[[sha256.Size]byte]struct{}),
	}
}

// CallTool leaves cancellation to the wrapped executor and the agent loop: a
// call already under way when the investigation times out may still return an
// image, and the timeout report keeps it.
func (e *imageExecutor) CallTool(ctx context.Context, name string, args json.RawMessage) (*llm.ToolResult, error) {
	key := panelCallKey(name, args)
	if content, ok := e.panels[key]; key != "" && ok {
		log.Debugf("[agent] reused successful panel capture: tool=%s", name)
		return &llm.ToolResult{Content: content}, nil
	}
	result, err := e.ToolExecutor.CallTool(ctx, name, args)
	if err != nil || result == nil {
		return result, err
	}

	// Do not modify the executor's result or its image slice when filtering.
	filtered := &llm.ToolResult{Content: result.Content}
	hasImage := false
	for _, img := range result.Images {
		if len(img) == 0 {
			continue
		}
		hasImage = true
		hash := sha256.Sum256(img)
		if _, exists := e.seen[hash]; exists {
			log.Debugf("[agent] skipped duplicate image: tool=%s sha256=%x", name, hash)
			continue
		}
		e.seen[hash] = struct{}{}
		filtered.Images = append(filtered.Images, img)
	}
	if hasImage {
		filtered.Content += e.note
		if key != "" {
			e.panels[key] = filtered.Content
		}
	}
	return filtered, nil
}

func panelCallKey(name string, args json.RawMessage) string {
	if name != "get_panel_image" && !strings.HasSuffix(name, "__get_panel_image") {
		return ""
	}
	if !json.Valid(args) {
		return ""
	}
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.UseNumber() // Preserve large numbers instead of rounding through float64.
	if err := decoder.Decode(&value); err != nil || value == nil {
		return ""
	}
	// encoding/json sorts object keys, including nested dashboard variables.
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return name + ":" + string(canonical)
}
