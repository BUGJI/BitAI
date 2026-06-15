package handlers

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"time"

	bithttp "bitapi/backend/internal/http/respond"
	"bitapi/backend/internal/services/adapters"
	gatewaysvc "bitapi/backend/internal/services/gateway"
	"github.com/gin-gonic/gin"
)

type GatewayHandler struct {
	gateway *gatewaysvc.Service
}

func NewGatewayHandler(gateway *gatewaysvc.Service) *GatewayHandler {
	return &GatewayHandler{gateway: gateway}
}

func (h *GatewayHandler) ChatCompletions(c *gin.Context) {
	key := bearer(c.GetHeader("Authorization"))
	if key == "" {
		bithttp.Fail(c, http.StatusUnauthorized, "缺少接口密钥")
		return
	}
	principal, err := h.gateway.Authenticate(key)
	if err != nil {
		bithttp.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "请求体格式错误")
		return
	}
	chat, err := h.gateway.BeginChat(principal, body)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if chat.Stream {
		h.streamChat(c, chat)
		return
	}
	result, err := h.gateway.ProxyPreparedChat(chat, c.Request.Header)
	if err != nil {
		bithttp.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	for name, values := range result.Header {
		lower := strings.ToLower(name)
		if lower == "content-length" || lower == "transfer-encoding" || lower == "connection" {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
	c.Writer.Header().Set("X-BitAPI-Request-ID", result.RequestID)
	c.Data(result.StatusCode, "application/json", result.Body)
}

func (h *GatewayHandler) Responses(c *gin.Context) {
	key := bearer(c.GetHeader("Authorization"))
	if key == "" {
		bithttp.Fail(c, http.StatusUnauthorized, "缺少接口密钥")
		return
	}
	principal, err := h.gateway.Authenticate(key)
	if err != nil {
		bithttp.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "请求体格式错误")
		return
	}
	ctx, err := h.gateway.BeginResponses(principal, body)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if ctx.Stream {
		h.streamResponses(c, ctx)
		return
	}
	result, err := h.gateway.ProxyPreparedResponses(ctx, c.Request.Header)
	if err != nil {
		bithttp.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	for name, values := range result.Header {
		lower := strings.ToLower(name)
		if lower == "content-length" || lower == "transfer-encoding" || lower == "connection" {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
	c.Writer.Header().Set("X-BitAPI-Request-ID", result.RequestID)
	c.Data(result.StatusCode, "application/json", result.Body)
}

func (h *GatewayHandler) Models(c *gin.Context) {
	key := bearer(c.GetHeader("Authorization"))
	if key == "" {
		bithttp.Fail(c, http.StatusUnauthorized, "缺少接口密钥")
		return
	}
	principal, err := h.gateway.Authenticate(key)
	if err != nil {
		bithttp.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	models, err := h.gateway.Models(principal)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	data := make([]gin.H, 0, len(models))
	for _, model := range models {
		data = append(data, gin.H{"id": model, "object": "model", "owned_by": "bitapi"})
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
}

func (h *GatewayHandler) streamResponses(c *gin.Context, ctx gatewaysvc.ResponsesContext) {
	start := time.Now()
	resp, err := h.gateway.ProxyResponsesStream(ctx, c.Request.Header)
	if err != nil {
		h.gateway.RecordResponsesUsage(ctx, adapters.ResponseMeta{}, http.StatusBadGateway, time.Since(start).Milliseconds(), err.Error())
		bithttp.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	for name, values := range resp.Header {
		lower := strings.ToLower(name)
		if lower == "content-length" || lower == "transfer-encoding" || lower == "connection" {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
	c.Writer.Header().Set("X-BitAPI-Request-ID", ctx.RequestID)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Status(resp.StatusCode)

	writer := c.Writer
	flusher, _ := writer.(http.Flusher)
	streamMeta, copyErr := adapters.CopyAndExtractStreamMeta(writer, resp.Body)
	if flusher != nil {
		flusher.Flush()
	}
	errMsg := ""
	if copyErr != nil {
		errMsg = copyErr.Error()
	}
	h.gateway.RecordResponsesUsage(ctx, streamMeta, resp.StatusCode, time.Since(start).Milliseconds(), errMsg)
}

func (h *GatewayHandler) streamChat(c *gin.Context, chat gatewaysvc.ChatContext) {
	start := time.Now()
	resp, err := h.gateway.ProxyChatStream(chat, c.Request.Header)
	if err != nil {
		h.gateway.RecordUsage(chat, adapters.ResponseMeta{}, http.StatusBadGateway, time.Since(start).Milliseconds(), err.Error())
		bithttp.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	for name, values := range resp.Header {
		lower := strings.ToLower(name)
		if lower == "content-length" || lower == "transfer-encoding" || lower == "connection" {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
	c.Writer.Header().Set("X-BitAPI-Request-ID", chat.RequestID)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Status(resp.StatusCode)

	writer := c.Writer
	flusher, _ := writer.(http.Flusher)
	reader := bufio.NewReader(resp.Body)
	_, copyErr := io.Copy(writer, reader)
	if flusher != nil {
		flusher.Flush()
	}
	errMsg := ""
	if copyErr != nil {
		errMsg = copyErr.Error()
	}
	h.gateway.RecordUsage(chat, adapters.ResponseMeta{}, resp.StatusCode, time.Since(start).Milliseconds(), errMsg)
}

func bearer(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return ""
}
