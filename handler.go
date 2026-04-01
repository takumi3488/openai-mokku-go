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

const systemFingerprint = "fp_mock"

func marshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// MockHandler implements the api.Handler interface
type MockHandler struct{}

var _ api.Handler = (*MockHandler)(nil)

// CreateChatCompletion implements createChatCompletion operation.
func (h *MockHandler) CreateChatCompletion(ctx context.Context, req *api.CreateChatCompletionRequest) (*api.CreateChatCompletionResponse, error) {
	ctx, span := tracer.Start(ctx, "CreateChatCompletion.process")
	defer span.End()

	span.SetAttributes(attribute.String("request.full_json", marshalJSON(req)))

	lastUserMessage := extractLastUserMessage(req.Messages)

	attrs := []attribute.KeyValue{
		attribute.String("model", req.Model),
		attribute.Int("message_count", len(req.Messages)),
		attribute.String("last_user_message", lastUserMessage),
	}

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

	// Priority: ResponseFormat (json_schema/json_object) > Tools
	var choices []api.ChatCompletionChoice
	var completionLen int

	if req.ResponseFormat.Set &&
		(req.ResponseFormat.Value.Type == api.ChatCompletionResponseFormatTypeJSONSchema ||
			req.ResponseFormat.Value.Type == api.ChatCompletionResponseFormatTypeJSONObject) &&
		req.ResponseFormat.Value.JSONSchema.Set {
		jsonContent := generateJSONFromSchemaBytes(req.ResponseFormat.Value.JSONSchema.Value.Schema)
		completionLen = len(jsonContent)
		choices = []api.ChatCompletionChoice{
			{
				Index: 0,
				Message: api.ChatCompletionResponseMessage{
					Role:    api.ChatCompletionResponseMessageRoleAssistant,
					Content: api.NewNilString(jsonContent),
				},
				FinishReason: api.ChatCompletionChoiceFinishReasonStop,
			},
		}
	} else if len(req.Tools) > 0 {
		tool := req.Tools[0]
		argsMap := map[string]string{"input": lastUserMessage}
		argsBytes, _ := json.Marshal(argsMap)
		args := string(argsBytes)
		completionLen = len(args)
		choices = []api.ChatCompletionChoice{
			{
				Index: 0,
				Message: api.ChatCompletionResponseMessage{
					Role:    api.ChatCompletionResponseMessageRoleAssistant,
					Content: api.NewNilString(""),
					ToolCalls: []api.ChatCompletionMessageToolCall{
						{
							ID:   "call_" + uuid.New().String(),
							Type: api.ChatCompletionMessageToolCallTypeFunction,
							Function: api.ChatCompletionMessageToolCallFunction{
								Name:      tool.Function.Name,
								Arguments: args,
							},
						},
					},
				},
				FinishReason: api.ChatCompletionChoiceFinishReasonToolCalls,
			},
		}
	} else {
		echoMessage := generateEchoResponse(ctx, lastUserMessage)
		completionLen = len(echoMessage)
		choices = []api.ChatCompletionChoice{
			{
				Index: 0,
				Message: api.ChatCompletionResponseMessage{
					Role:    api.ChatCompletionResponseMessageRoleAssistant,
					Content: api.NewNilString(echoMessage),
				},
				FinishReason: api.ChatCompletionChoiceFinishReasonStop,
			},
		}
	}

	response := &api.CreateChatCompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String(),
		Object:  api.CreateChatCompletionResponseObjectChatCompletion,
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: choices,
		Usage: api.NewOptCompletionUsage(api.CompletionUsage{
			PromptTokens:     len(lastUserMessage),
			CompletionTokens: completionLen,
			TotalTokens:      len(lastUserMessage) + completionLen,
		}),
		SystemFingerprint: api.NewOptString(systemFingerprint),
	}

	span.SetAttributes(attribute.String("response.full_json", marshalJSON(response)))

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

	attrs := []attribute.KeyValue{
		attribute.String("model", req.Model),
		attribute.String("prompt", prompt),
	}

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
		SystemFingerprint: api.NewOptString(systemFingerprint),
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

// CreateResponse implements createResponse operation.
func (h *MockHandler) CreateResponse(ctx context.Context, req *api.CreateResponseRequest) (*api.CreateResponseResponse, error) {
	ctx, span := tracer.Start(ctx, "CreateResponse.process")
	defer span.End()

	span.SetAttributes(attribute.String("request.full_json", marshalJSON(req)))

	var output []api.ResponseOutputItem
	var outputText string

	if len(req.Tools) > 0 {
		// Return function_call output when tools are present
		tool := req.Tools[0]
		argsMap := map[string]string{"input": req.Input}
		argsBytes, _ := json.Marshal(argsMap)
		args := string(argsBytes)
		outputText = args
		output = []api.ResponseOutputItem{
			{
				Type:      api.ResponseOutputItemTypeFunctionCall,
				ID:        api.NewOptString("call_" + uuid.New().String()),
				CallID:    api.NewOptString("call_" + uuid.New().String()),
				Name:      api.NewOptString(tool.Name),
				Arguments: api.NewOptString(args),
			},
		}
	} else {
		var msgText string
		if req.Text.Set &&
			req.Text.Value.Format.Set &&
			(req.Text.Value.Format.Value.Type == api.ResponseTextFormatTypeJSONSchema ||
				req.Text.Value.Format.Value.Type == api.ResponseTextFormatTypeJSONObject) {
			msgText = generateJSONFromSchemaBytes(req.Text.Value.Format.Value.Schema)
		} else {
			msgText = generateEchoResponse(ctx, req.Input)
		}
		outputText = msgText
		output = []api.ResponseOutputItem{
			{
				Type: api.ResponseOutputItemTypeMessage,
				ID:   api.NewOptString("msg-" + uuid.New().String()),
				Role: api.NewOptString("assistant"),
				Content: []api.ResponseOutputContent{
					{
						Type: api.ResponseOutputContentTypeOutputText,
						Text: msgText,
					},
				},
			},
		}
	}

	response := &api.CreateResponseResponse{
		ID:        "resp-" + uuid.New().String(),
		Object:    api.CreateResponseResponseObjectResponse,
		CreatedAt: api.NewOptInt64(time.Now().Unix()),
		Status:    api.CreateResponseResponseStatusCompleted,
		Model:     req.Model,
		Output:    output,
		Usage: api.ResponseUsage{
			InputTokens:  len(req.Input),
			OutputTokens: len(outputText),
			TotalTokens:  len(req.Input) + len(outputText),
		},
	}

	span.SetAttributes(attribute.String("response.full_json", marshalJSON(response)))

	return response, nil
}

// CreateEmbedding implements createEmbedding operation.
func (h *MockHandler) CreateEmbedding(ctx context.Context, req *api.CreateEmbeddingRequest) (*api.CreateEmbeddingResponse, error) {
	_, span := tracer.Start(ctx, "CreateEmbedding.process")
	defer span.End()

	span.SetAttributes(attribute.String("request.full_json", marshalJSON(req)))

	inputs := normalizeInputStrings(req.Input)

	dimensions := defaultEmbeddingDimensions
	if req.Dimensions.Set {
		dimensions = int(req.Dimensions.Value)
	}

	embeddings := make([]api.Embedding, len(inputs))
	totalTokens := 0
	for i, text := range inputs {
		vector := generateVector(text, dimensions)
		embeddings[i] = api.Embedding{
			Index:     i,
			Object:    api.EmbeddingObjectEmbedding,
			Embedding: vector,
		}
		totalTokens += len(text)
	}

	response := &api.CreateEmbeddingResponse{
		Object: api.CreateEmbeddingResponseObjectList,
		Data:   embeddings,
		Model:  req.Model,
		Usage: api.EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}

	span.SetAttributes(attribute.String("response.full_json", marshalJSON(response)))

	return response, nil
}

func generateEchoResponse(ctx context.Context, message string) string {
	_, span := tracer.Start(ctx, "generateEchoResponse")
	defer span.End()

	span.SetAttributes(attribute.String("input_message", message))

	echoResponse := fmt.Sprintf("Echo: %s", message)

	span.SetAttributes(attribute.String("echo_response", echoResponse))

	return echoResponse
}
