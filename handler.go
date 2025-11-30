package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"openai-mokku/api"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("openai-mokku")

// MockHandler implements the api.Handler interface
type MockHandler struct{}

var _ api.Handler = (*MockHandler)(nil)

// CreateChatCompletion implements createChatCompletion operation.
func (h *MockHandler) CreateChatCompletion(ctx context.Context, req *api.CreateChatCompletionRequest) (*api.CreateChatCompletionResponse, error) {
	ctx, span := tracer.Start(ctx, "CreateChatCompletion.process")
	defer span.End()

	// Log full request as JSON
	reqJSON, _ := json.Marshal(req)
	span.SetAttributes(attribute.String("request.full_json", string(reqJSON)))

	// Get the last user message
	var lastUserMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == api.ChatCompletionRequestMessageRoleUser {
			lastUserMessage = req.Messages[i].Content
			break
		}
	}

	// Set basic request attributes
	attrs := []attribute.KeyValue{
		attribute.String("model", req.Model),
		attribute.Int("message_count", len(req.Messages)),
		attribute.String("last_user_message", lastUserMessage),
	}

	// Add optional parameters if set
	if req.Temperature.Set {
		attrs = append(attrs, attribute.Float64("temperature", req.Temperature.Value))
	}
	if req.TopP.Set {
		attrs = append(attrs, attribute.Float64("top_p", req.TopP.Value))
	}
	if req.N.Set {
		attrs = append(attrs, attribute.Int("n", req.N.Value))
	}
	if req.Stream.Set {
		attrs = append(attrs, attribute.Bool("stream", req.Stream.Value))
	}
	if req.MaxTokens.Set {
		attrs = append(attrs, attribute.Int("max_tokens", req.MaxTokens.Value))
	}
	if req.MaxCompletionTokens.Set {
		attrs = append(attrs, attribute.Int("max_completion_tokens", req.MaxCompletionTokens.Value))
	}
	if req.PresencePenalty.Set {
		attrs = append(attrs, attribute.Float64("presence_penalty", req.PresencePenalty.Value))
	}
	if req.FrequencyPenalty.Set {
		attrs = append(attrs, attribute.Float64("frequency_penalty", req.FrequencyPenalty.Value))
	}
	if req.User.Set {
		attrs = append(attrs, attribute.String("user", req.User.Value))
	}
	if req.Seed.Set {
		attrs = append(attrs, attribute.Int("seed", req.Seed.Value))
	}

	span.SetAttributes(attrs...)

	// Generate echo response
	echoMessage := generateEchoResponse(ctx, lastUserMessage)

	response := &api.CreateChatCompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String(),
		Object:  api.CreateChatCompletionResponseObjectChatCompletion,
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []api.ChatCompletionChoice{
			{
				Index: 0,
				Message: api.ChatCompletionResponseMessage{
					Role:    api.ChatCompletionResponseMessageRoleAssistant,
					Content: api.NewNilString(echoMessage),
				},
				FinishReason: api.ChatCompletionChoiceFinishReasonStop,
			},
		},
		Usage: api.NewOptCompletionUsage(api.CompletionUsage{
			PromptTokens:     len(lastUserMessage),
			CompletionTokens: len(echoMessage),
			TotalTokens:      len(lastUserMessage) + len(echoMessage),
		}),
		SystemFingerprint: api.NewOptString("fp_mock"),
	}

	// Log full response as JSON
	respJSON, _ := json.Marshal(response)
	span.SetAttributes(attribute.String("response.full_json", string(respJSON)))

	return response, nil
}

// CreateCompletion implements createCompletion operation.
func (h *MockHandler) CreateCompletion(ctx context.Context, req *api.CreateCompletionRequest) (*api.CreateCompletionResponse, error) {
	ctx, span := tracer.Start(ctx, "CreateCompletion.process")
	defer span.End()

	// Get the prompt (simplified: just use string version)
	prompt := ""
	if req.Prompt.IsString() {
		prompt, _ = req.Prompt.GetString()
	} else if req.Prompt.IsStringArray() {
		arr, _ := req.Prompt.GetStringArray()
		if len(arr) > 0 {
			prompt = arr[0]
		}
	}

	// Set basic request attributes
	attrs := []attribute.KeyValue{
		attribute.String("model", req.Model),
		attribute.String("prompt", prompt),
	}

	// Add optional parameters if set
	if req.Temperature.Set {
		attrs = append(attrs, attribute.Float64("temperature", req.Temperature.Value))
	}
	if req.TopP.Set {
		attrs = append(attrs, attribute.Float64("top_p", req.TopP.Value))
	}
	if req.N.Set {
		attrs = append(attrs, attribute.Int("n", req.N.Value))
	}
	if req.Stream.Set {
		attrs = append(attrs, attribute.Bool("stream", req.Stream.Value))
	}
	if req.MaxTokens.Set {
		attrs = append(attrs, attribute.Int("max_tokens", req.MaxTokens.Value))
	}
	if req.Echo.Set {
		attrs = append(attrs, attribute.Bool("echo", req.Echo.Value))
	}
	if req.PresencePenalty.Set {
		attrs = append(attrs, attribute.Float64("presence_penalty", req.PresencePenalty.Value))
	}
	if req.FrequencyPenalty.Set {
		attrs = append(attrs, attribute.Float64("frequency_penalty", req.FrequencyPenalty.Value))
	}
	if req.BestOf.Set {
		attrs = append(attrs, attribute.Int("best_of", req.BestOf.Value))
	}
	if req.User.Set {
		attrs = append(attrs, attribute.String("user", req.User.Value))
	}
	if req.Seed.Set {
		attrs = append(attrs, attribute.Int("seed", req.Seed.Value))
	}

	span.SetAttributes(attrs...)

	// Generate echo response
	echoText := generateEchoResponse(ctx, prompt)

	response := &api.CreateCompletionResponse{
		ID:      "cmpl-" + uuid.New().String(),
		Object:  api.CreateCompletionResponseObjectTextCompletion,
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []api.CompletionChoice{
			{
				Index:        0,
				Text:         echoText,
				FinishReason: api.CompletionChoiceFinishReasonStop,
			},
		},
		Usage: api.NewOptCompletionUsage(api.CompletionUsage{
			PromptTokens:     len(prompt),
			CompletionTokens: len(echoText),
			TotalTokens:      len(prompt) + len(echoText),
		}),
		SystemFingerprint: api.NewOptString("fp_mock"),
	}

	return response, nil
}

// ListModels implements listModels operation.
func (h *MockHandler) ListModels(ctx context.Context) (*api.ListModelsResponse, error) {
	_, span := tracer.Start(ctx, "ListModels.process")
	defer span.End()

	return &api.ListModelsResponse{
		Object: api.ListModelsResponseObjectList,
		Data: []api.Model{
			{
				ID:      "mokku-echo-1",
				Object:  api.ModelObjectModel,
				Created: time.Now().Unix(),
				OwnedBy: "openai-mokku",
			},
			{
				ID:      "gpt-4o",
				Object:  api.ModelObjectModel,
				Created: time.Now().Unix(),
				OwnedBy: "openai-mokku",
			},
			{
				ID:      "gpt-4o-mini",
				Object:  api.ModelObjectModel,
				Created: time.Now().Unix(),
				OwnedBy: "openai-mokku",
			},
		},
	}, nil
}

// RetrieveModel implements retrieveModel operation.
func (h *MockHandler) RetrieveModel(ctx context.Context, params api.RetrieveModelParams) (*api.Model, error) {
	_, span := tracer.Start(ctx, "RetrieveModel.process")
	defer span.End()

	span.SetAttributes(attribute.String("model", params.Model))

	return &api.Model{
		ID:      params.Model,
		Object:  api.ModelObjectModel,
		Created: time.Now().Unix(),
		OwnedBy: "openai-mokku",
	}, nil
}

func generateEchoResponse(ctx context.Context, message string) string {
	_, span := tracer.Start(ctx, "generateEchoResponse")
	defer span.End()

	span.SetAttributes(attribute.String("input_message", message))

	echoResponse := fmt.Sprintf("Echo: %s", message)

	span.SetAttributes(attribute.String("echo_response", echoResponse))

	return echoResponse
}
