package adapters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type RequestMeta struct {
	Model  string
	Stream bool
}

type ResponseMeta struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type OpenAIAdapter struct {
	client *http.Client
}

func NewOpenAIAdapter() *OpenAIAdapter {
	return &OpenAIAdapter{client: &http.Client{Timeout: 10 * time.Minute}}
}

func (a *OpenAIAdapter) ExtractRequestMeta(body []byte) RequestMeta {
	var payload struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	_ = json.Unmarshal(body, &payload)
	return RequestMeta{Model: payload.Model, Stream: payload.Stream}
}

func (a *OpenAIAdapter) ExtractStream(body []byte) bool {
	var payload struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &payload)
	return payload.Stream
}

func (a *OpenAIAdapter) Proxy(baseURL, credential, path string, body []byte, headers http.Header) (*http.Response, []byte, error) {
	req, err := a.newRequest(baseURL, credential, path, body, headers)
	if err != nil {
		return nil, nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return resp, nil, err
	}
	return resp, respBody, nil
}

func (a *OpenAIAdapter) ProxyStream(baseURL, credential, path string, body []byte, headers http.Header) (*http.Response, error) {
	req, err := a.newRequest(baseURL, credential, path, body, headers)
	if err != nil {
		return nil, err
	}
	return a.client.Do(req)
}

func (a *OpenAIAdapter) RewriteModel(body []byte, model string) ([]byte, error) {
	if model == "" {
		return body, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["model"] = model
	return json.Marshal(payload)
}

func (a *OpenAIAdapter) ExtractResponseMeta(body []byte) ResponseMeta {
	type usagePayload struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		InputTokens      int `json:"input_tokens"`
		OutputTokens     int `json:"output_tokens"`
	}
	var payload struct {
		Model    string       `json:"model"`
		Usage    usagePayload `json:"usage"`
		Response struct {
			Model string       `json:"model"`
			Usage usagePayload `json:"usage"`
		} `json:"response"`
	}
	_ = json.Unmarshal(body, &payload)
	model := payload.Model
	if model == "" {
		model = payload.Response.Model
	}
	usage := payload.Usage
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 && usage.InputTokens == 0 && usage.OutputTokens == 0 {
		usage = payload.Response.Usage
	}
	promptTokens := usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = usage.InputTokens
	}
	completionTokens := usage.CompletionTokens
	if completionTokens == 0 {
		completionTokens = usage.OutputTokens
	}
	total := usage.TotalTokens
	if total == 0 {
		total = promptTokens + completionTokens
	}
	return ResponseMeta{
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      total,
	}
}

func (a *OpenAIAdapter) ExtractStreamEventMeta(line []byte) ResponseMeta {
	data := strings.TrimSpace(string(line))
	if !strings.HasPrefix(data, "data:") {
		return ResponseMeta{}
	}
	data = strings.TrimSpace(strings.TrimPrefix(data, "data:"))
	if data == "" || data == "[DONE]" {
		return ResponseMeta{}
	}
	return a.ExtractResponseMeta([]byte(data))
}

func MergeResponseMeta(current, next ResponseMeta) ResponseMeta {
	if next.Model != "" {
		current.Model = next.Model
	}
	if next.PromptTokens > 0 {
		current.PromptTokens = next.PromptTokens
	}
	if next.CompletionTokens > 0 {
		current.CompletionTokens = next.CompletionTokens
	}
	if next.TotalTokens > 0 {
		current.TotalTokens = next.TotalTokens
	}
	return current
}

func CopyAndExtractStreamMeta(dst io.Writer, src io.Reader) (ResponseMeta, error) {
	adapter := NewOpenAIAdapter()
	var meta ResponseMeta
	scanner := bufio.NewScanner(src)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if _, err := dst.Write(line); err != nil {
			return meta, err
		}
		if _, err := dst.Write([]byte("\n")); err != nil {
			return meta, err
		}
		meta = MergeResponseMeta(meta, adapter.ExtractStreamEventMeta(line))
	}
	if err := scanner.Err(); err != nil {
		return meta, err
	}
	return meta, nil
}

func (a *OpenAIAdapter) newRequest(baseURL, credential, path string, body []byte, headers http.Header) (*http.Request, error) {
	target := upstreamURL(baseURL, path)
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+credential)
	if organization := headers.Get("OpenAI-Organization"); organization != "" {
		req.Header.Set("OpenAI-Organization", organization)
	}
	if project := headers.Get("OpenAI-Project"); project != "" {
		req.Header.Set("OpenAI-Project", project)
	}
	return req, nil
}

func upstreamURL(baseURL, path string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}
	return base + path
}
