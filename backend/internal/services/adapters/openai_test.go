package adapters

import (
	"strings"
	"testing"
)

func TestExtractResponseMetaSupportsResponsesUsage(t *testing.T) {
	meta := NewOpenAIAdapter().ExtractResponseMeta([]byte(`{
		"model": "gpt-4.1-mini",
		"usage": {
			"input_tokens": 12,
			"output_tokens": 8,
			"total_tokens": 20
		}
	}`))
	if meta.Model != "gpt-4.1-mini" {
		t.Fatalf("model = %q", meta.Model)
	}
	if meta.PromptTokens != 12 || meta.CompletionTokens != 8 || meta.TotalTokens != 20 {
		t.Fatalf("tokens = prompt:%d completion:%d total:%d", meta.PromptTokens, meta.CompletionTokens, meta.TotalTokens)
	}
}

func TestExtractResponseMetaSupportsNestedResponsesUsage(t *testing.T) {
	meta := NewOpenAIAdapter().ExtractResponseMeta([]byte(`{
		"type": "response.completed",
		"response": {
			"model": "gpt-5.5",
			"usage": {
				"input_tokens": 21,
				"output_tokens": 9,
				"total_tokens": 30
			}
		}
	}`))
	if meta.Model != "gpt-5.5" {
		t.Fatalf("model = %q", meta.Model)
	}
	if meta.PromptTokens != 21 || meta.CompletionTokens != 9 || meta.TotalTokens != 30 {
		t.Fatalf("tokens = prompt:%d completion:%d total:%d", meta.PromptTokens, meta.CompletionTokens, meta.TotalTokens)
	}
}

func TestCopyAndExtractStreamMeta(t *testing.T) {
	source := strings.NewReader(strings.Join([]string{
		`event: response.created`,
		`data: {"type":"response.created","response":{"model":"gpt-5.5"}}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"model":"gpt-5.5","usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}}}`,
		``,
	}, "\n"))
	var output strings.Builder

	meta, err := CopyAndExtractStreamMeta(&output, source)
	if err != nil {
		t.Fatalf("copy stream: %v", err)
	}
	if !strings.Contains(output.String(), "response.completed") {
		t.Fatalf("stream was not copied: %q", output.String())
	}
	if meta.Model != "gpt-5.5" || meta.PromptTokens != 11 || meta.CompletionTokens != 7 || meta.TotalTokens != 18 {
		t.Fatalf("meta = %+v", meta)
	}
}
