package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"openai-mokku/api"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// CreditErrorModelName is the model name that triggers a 402 credit error
const CreditErrorModelName = "credit-error"

// modelRequest is used to extract just the model field from any completion request
type modelRequest struct {
	Model string `json:"model"`
}

// readBodyAndCheckCreditError reads the request body, checks if the model triggers a credit error,
// and returns the body for further processing. Returns nil and true if the request was handled (error written).
func readBodyAndCheckCreditError(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil, true
	}
	_ = r.Body.Close()

	var req modelRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return nil, true
	}

	if req.Model == CreditErrorModelName {
		writeCreditError(w)
		return nil, true
	}

	return body, false
}

// OpenAIError represents an OpenAI API error response
type OpenAIError struct {
	Error OpenAIErrorDetail `json:"error"`
}

// OpenAIErrorDetail represents the error detail in an OpenAI API error response
type OpenAIErrorDetail struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    string  `json:"code"`
}

// ChatCompletionChunk represents a streaming response chunk
type ChatCompletionChunk struct {
	ID                string                    `json:"id"`
	Object            string                    `json:"object"`
	Created           int64                     `json:"created"`
	Model             string                    `json:"model"`
	SystemFingerprint string                    `json:"system_fingerprint,omitempty"`
	Choices           []ChatCompletionChunkChoice `json:"choices"`
}

// ChatCompletionChunkChoice represents a choice in a streaming chunk
type ChatCompletionChunkChoice struct {
	Index        int                      `json:"index"`
	Delta        ChatCompletionChunkDelta `json:"delta"`
	FinishReason *string                  `json:"finish_reason"`
}

// ChatCompletionChunkDelta represents the delta content in a streaming chunk
type ChatCompletionChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// StreamingHandler wraps the ogen server and handles streaming requests
type StreamingHandler struct {
	ogenServer http.Handler
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(ogenServer http.Handler) *StreamingHandler {
	return &StreamingHandler{
		ogenServer: ogenServer,
	}
}

// ServeHTTP implements http.Handler
func (h *StreamingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Intercept POST /v1/chat/completions
	if r.Method == http.MethodPost && r.URL.Path == "/v1/chat/completions" {
		body, handled := readBodyAndCheckCreditError(w, r)
		if handled {
			return
		}

		var req api.CreateChatCompletionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Failed to parse request body", http.StatusBadRequest)
			return
		}

		// Check if streaming is requested
		if req.Stream.Set && req.Stream.Value {
			h.handleStreamingRequest(w, r, &req)
			return
		}

		// For non-streaming requests, reconstruct the body and pass to ogen server
		r.Body = io.NopCloser(newBytesReader(body))
	}

	// Intercept POST /v1/completions for credit error simulation
	if r.Method == http.MethodPost && r.URL.Path == "/v1/completions" {
		body, handled := readBodyAndCheckCreditError(w, r)
		if handled {
			return
		}

		// Reconstruct the body and pass to ogen server
		r.Body = io.NopCloser(newBytesReader(body))
	}

	// Pass to ogen server for other requests
	h.ogenServer.ServeHTTP(w, r)
}

// bytesReader is a simple bytes.Reader wrapper
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data, pos: 0}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// handleStreamingRequest handles streaming chat completion requests
func (h *StreamingHandler) handleStreamingRequest(w http.ResponseWriter, r *http.Request, req *api.CreateChatCompletionRequest) {
	ctx, span := tracer.Start(r.Context(), "CreateChatCompletion.streaming")
	defer span.End()

	// Log full request as JSON
	reqJSON, _ := json.Marshal(req)
	span.SetAttributes(attribute.String("request.full_json", string(reqJSON)))
	span.SetAttributes(attribute.Bool("stream", true))

	// Get the last user message
	var lastUserMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == api.ChatCompletionRequestMessageRoleUser {
			lastUserMessage = req.Messages[i].Content
			break
		}
	}

	span.SetAttributes(
		attribute.String("model", req.Model),
		attribute.Int("message_count", len(req.Messages)),
		attribute.String("last_user_message", lastUserMessage),
	)

	// Generate echo response
	echoMessage := generateEchoResponseForStreaming(ctx, lastUserMessage)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	completionID := "chatcmpl-" + uuid.New().String()
	created := time.Now().Unix()

	// Send first chunk with role
	firstChunk := ChatCompletionChunk{
		ID:                completionID,
		Object:            "chat.completion.chunk",
		Created:           created,
		Model:             req.Model,
		SystemFingerprint: "fp_mock",
		Choices: []ChatCompletionChunkChoice{
			{
				Index: 0,
				Delta: ChatCompletionChunkDelta{
					Role: "assistant",
				},
				FinishReason: nil,
			},
		},
	}

	if err := writeSSEChunk(w, firstChunk); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	flusher.Flush()

	// Send content chunk
	contentChunk := ChatCompletionChunk{
		ID:                completionID,
		Object:            "chat.completion.chunk",
		Created:           created,
		Model:             req.Model,
		SystemFingerprint: "fp_mock",
		Choices: []ChatCompletionChunkChoice{
			{
				Index: 0,
				Delta: ChatCompletionChunkDelta{
					Content: echoMessage,
				},
				FinishReason: nil,
			},
		},
	}

	if err := writeSSEChunk(w, contentChunk); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	flusher.Flush()

	// Send final chunk with finish_reason
	finishReason := "stop"
	finalChunk := ChatCompletionChunk{
		ID:                completionID,
		Object:            "chat.completion.chunk",
		Created:           created,
		Model:             req.Model,
		SystemFingerprint: "fp_mock",
		Choices: []ChatCompletionChunkChoice{
			{
				Index:        0,
				Delta:        ChatCompletionChunkDelta{},
				FinishReason: &finishReason,
			},
		},
	}

	if err := writeSSEChunk(w, finalChunk); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	flusher.Flush()

	// Send [DONE] marker
	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	span.SetAttributes(attribute.String("response.echo_message", echoMessage))
}

// writeCreditError writes a 402 credit error response
func writeCreditError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)

	errorResp := OpenAIError{
		Error: OpenAIErrorDetail{
			Message: "You exceeded your current quota, please check your plan and billing details. For more information on this error, read the docs: https://platform.openai.com/docs/guides/error-codes/api-errors.",
			Type:    "insufficient_quota",
			Param:   nil,
			Code:    "insufficient_quota",
		},
	}

	_ = json.NewEncoder(w).Encode(errorResp)
}

// writeSSEChunk writes a chunk in SSE format
func writeSSEChunk(w http.ResponseWriter, chunk ChatCompletionChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

// generateEchoResponseForStreaming generates echo response with tracing
func generateEchoResponseForStreaming(ctx context.Context, message string) string {
	_, span := tracer.Start(ctx, "generateEchoResponse")
	defer span.End()

	span.SetAttributes(attribute.String("input_message", message))

	echoResponse := fmt.Sprintf("Echo: %s", message)

	span.SetAttributes(attribute.String("echo_response", echoResponse))

	return echoResponse
}
