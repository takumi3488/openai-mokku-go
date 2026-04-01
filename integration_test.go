package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"openai-mokku/api"
)

// newTestServer creates a test HTTP server using MockHandler + StreamingHandler.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler := &MockHandler{}
	ogenServer, err := api.NewServer(handler, api.WithPathPrefix("/v1"))
	if err != nil {
		t.Fatalf("api.NewServer: %v", err)
	}
	return httptest.NewServer(NewStreamingHandler(ogenServer))
}

// postJSON sends a POST request with a JSON body and returns the response.
func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

// mustDecodeJSON decodes a JSON response body into a map.
func mustDecodeJSON(t *testing.T, r io.Reader) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return m
}

// getChoices returns the choices array from a chat completion response map.
func getChoices(t *testing.T, result map[string]interface{}) []interface{} {
	t.Helper()
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Fatalf("expected non-empty choices array, got: %v", result["choices"])
	}
	return choices
}

// --- Chat Completions ---

func TestIntegration_ChatCompletion_Echo(t *testing.T) {
	// Given: a running server and a normal user message
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello world"}]}`

	// When: posting to chat completions
	resp := postJSON(t, srv.URL+"/v1/chat/completions", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: 200 OK with echo content containing original message
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	choices := getChoices(t, result)
	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content, _ := message["content"].(string)
	if !strings.Contains(content, "hello world") {
		t.Errorf("expected echo of 'hello world', got %q", content)
	}
	finishReason, _ := choice["finish_reason"].(string)
	if finishReason != "stop" {
		t.Errorf("expected finish_reason=stop, got %q", finishReason)
	}
}

func TestIntegration_ChatCompletion_LastUserMessageIsEchoed(t *testing.T) {
	// Given: multiple messages, only the last user message should be echoed
	srv := newTestServer(t)
	defer srv.Close()
	body := `{
		"model": "gpt-4o",
		"messages": [
			{"role": "user", "content": "first message"},
			{"role": "assistant", "content": "first response"},
			{"role": "user", "content": "last message"}
		]
	}`

	// When
	resp := postJSON(t, srv.URL+"/v1/chat/completions", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: content echoes "last message", not "first message"
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	choices := getChoices(t, result)
	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content, _ := message["content"].(string)
	if strings.Contains(content, "first message") && !strings.Contains(content, "last message") {
		t.Errorf("expected last user message to be echoed, got %q", content)
	}
	if !strings.Contains(content, "last message") {
		t.Errorf("expected 'last message' in echo, got %q", content)
	}
}

func TestIntegration_ChatCompletion_ResponseHasRequiredFields(t *testing.T) {
	// Given
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`

	// When
	resp := postJSON(t, srv.URL+"/v1/chat/completions", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: response has id, object, created, model, choices
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	for _, field := range []string{"id", "object", "created", "model", "choices"} {
		if result[field] == nil {
			t.Errorf("expected field %q in response", field)
		}
	}
	if obj, _ := result["object"].(string); obj != "chat.completion" {
		t.Errorf("expected object=chat.completion, got %q", obj)
	}
	if model, _ := result["model"].(string); model != "gpt-4o-mini" {
		t.Errorf("expected model=gpt-4o-mini, got %q", model)
	}
}

func TestIntegration_ChatCompletion_ToolCalls(t *testing.T) {
	// Given: a request with tools defined
	srv := newTestServer(t)
	defer srv.Close()
	body := `{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "get the weather"}],
		"tools": [{
			"type": "function",
			"function": {
				"name": "get_weather",
				"description": "Get weather for a location",
				"parameters": {"type": "object", "properties": {"location": {"type": "string"}}}
			}
		}]
	}`

	// When
	resp := postJSON(t, srv.URL+"/v1/chat/completions", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: finish_reason=tool_calls, message.tool_calls is non-empty
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	choices := getChoices(t, result)
	choice := choices[0].(map[string]interface{})
	finishReason, _ := choice["finish_reason"].(string)
	if finishReason != "tool_calls" {
		t.Errorf("expected finish_reason=tool_calls, got %q", finishReason)
	}
	message := choice["message"].(map[string]interface{})
	toolCalls, ok := message["tool_calls"].([]interface{})
	if !ok || len(toolCalls) == 0 {
		t.Error("expected non-empty tool_calls in message")
	}
	// Each tool call must have id, type, function
	tc := toolCalls[0].(map[string]interface{})
	for _, field := range []string{"id", "type", "function"} {
		if tc[field] == nil {
			t.Errorf("expected field %q in tool_call", field)
		}
	}
	fn := tc["function"].(map[string]interface{})
	if name, _ := fn["name"].(string); name != "get_weather" {
		t.Errorf("expected tool call name=get_weather, got %q", name)
	}
}

func TestIntegration_ChatCompletion_JSONSchema(t *testing.T) {
	// Given: a request with response_format=json_schema
	srv := newTestServer(t)
	defer srv.Close()
	body := `{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "give me a person"}],
		"response_format": {
			"type": "json_schema",
			"json_schema": {
				"name": "person",
				"schema": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"age":  {"type": "integer"}
					},
					"required": ["name", "age"]
				},
				"strict": true
			}
		}
	}`

	// When
	resp := postJSON(t, srv.URL+"/v1/chat/completions", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: 200 OK, finish_reason=stop, content is valid JSON with required fields
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	choices := getChoices(t, result)
	choice := choices[0].(map[string]interface{})
	finishReason, _ := choice["finish_reason"].(string)
	if finishReason != "stop" {
		t.Errorf("expected finish_reason=stop, got %q", finishReason)
	}
	message := choice["message"].(map[string]interface{})
	content, _ := message["content"].(string)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Errorf("expected valid JSON content, got %q: %v", content, err)
	}
	if _, ok := parsed["name"]; !ok {
		t.Error("expected 'name' field in JSON response")
	}
	if _, ok := parsed["age"]; !ok {
		t.Error("expected 'age' field in JSON response")
	}
}

func TestIntegration_ChatCompletion_JSONSchemaPriorityOverTools(t *testing.T) {
	// Given: both tools and json_schema response format
	srv := newTestServer(t)
	defer srv.Close()
	body := `{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "hi"}],
		"tools": [{"type": "function", "function": {"name": "fn", "description": "test", "parameters": {"type": "object"}}}],
		"response_format": {
			"type": "json_schema",
			"json_schema": {
				"name": "result",
				"schema": {"type": "object", "properties": {"value": {"type": "string"}}, "required": ["value"]},
				"strict": false
			}
		}
	}`

	// When
	resp := postJSON(t, srv.URL+"/v1/chat/completions", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: json_schema wins (finish_reason != tool_calls)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	choices := getChoices(t, result)
	choice := choices[0].(map[string]interface{})
	finishReason, _ := choice["finish_reason"].(string)
	if finishReason == "tool_calls" {
		t.Error("json_schema should take priority over tools, but got finish_reason=tool_calls")
	}
}

func TestIntegration_ChatCompletion_CreditError(t *testing.T) {
	// Given: the credit-error model name
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"credit-error","messages":[{"role":"user","content":"hi"}]}`

	// When
	resp := postJSON(t, srv.URL+"/v1/chat/completions", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: 402 Payment Required with insufficient_quota error type
	if resp.StatusCode != http.StatusPaymentRequired {
		t.Errorf("expected 402, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	errObj, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	errType, _ := errObj["type"].(string)
	if errType != "insufficient_quota" {
		t.Errorf("expected type=insufficient_quota, got %q", errType)
	}
}

// --- Responses API ---

func TestIntegration_Responses_PlainText(t *testing.T) {
	// Given: POST /v1/responses with plain text input
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"gpt-4o","input":"tell me a joke"}`

	// When
	resp := postJSON(t, srv.URL+"/v1/responses", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: 200 OK, status=completed, output contains a message item
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	for _, field := range []string{"id", "object", "status", "model", "output"} {
		if result[field] == nil {
			t.Errorf("expected field %q in response", field)
		}
	}
	if status, _ := result["status"].(string); status != "completed" {
		t.Errorf("expected status=completed, got %q", status)
	}
	output, ok := result["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("expected non-empty output array")
	}
	item := output[0].(map[string]interface{})
	if itemType, _ := item["type"].(string); itemType != "message" {
		t.Errorf("expected output[0].type=message, got %q", itemType)
	}
}

func TestIntegration_Responses_FunctionCall(t *testing.T) {
	// Given: POST /v1/responses with tools
	srv := newTestServer(t)
	defer srv.Close()
	body := `{
		"model": "gpt-4o",
		"input": "get the weather in Tokyo",
		"tools": [{
			"type": "function",
			"name": "get_weather",
			"description": "Get weather info",
			"parameters": {"type": "object", "properties": {"city": {"type": "string"}}}
		}]
	}`

	// When
	resp := postJSON(t, srv.URL+"/v1/responses", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: 200 OK, output contains function_call item
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	output, ok := result["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("expected non-empty output array")
	}
	item := output[0].(map[string]interface{})
	if itemType, _ := item["type"].(string); itemType != "function_call" {
		t.Errorf("expected output[0].type=function_call, got %q", itemType)
	}
}

func TestIntegration_Responses_StructuredOutput(t *testing.T) {
	// Given: POST /v1/responses with text.format=json_schema
	srv := newTestServer(t)
	defer srv.Close()
	body := `{
		"model": "gpt-4o",
		"input": "give me user data",
		"text": {
			"format": {
				"type": "json_schema",
				"name": "user",
				"schema": {
					"type": "object",
					"properties": {
						"username": {"type": "string"},
						"email":    {"type": "string"}
					},
					"required": ["username", "email"]
				}
			}
		}
	}`

	// When
	resp := postJSON(t, srv.URL+"/v1/responses", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: 200 OK, output contains message item with valid JSON content matching the schema
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	output, ok := result["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("expected non-empty output array")
	}
	item := output[0].(map[string]interface{})
	if itemType, _ := item["type"].(string); itemType != "message" {
		t.Errorf("expected output[0].type=message, got %q", itemType)
	}
	content, ok := item["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("expected non-empty content array")
	}
	contentItem := content[0].(map[string]interface{})
	text, _ := contentItem["text"].(string)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Errorf("expected valid JSON content, got %q: %v", text, err)
	}
	if _, ok := parsed["username"]; !ok {
		t.Error("expected 'username' field in JSON response")
	}
	if _, ok := parsed["email"]; !ok {
		t.Error("expected 'email' field in JSON response")
	}
}

func TestIntegration_Responses_HasUsage(t *testing.T) {
	// Given: any valid responses request
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"gpt-4o","input":"hello"}`

	// When
	resp := postJSON(t, srv.URL+"/v1/responses", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: usage field is present
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	if result["usage"] == nil {
		t.Error("expected usage field in responses response")
	}
}

// --- Embeddings ---

func TestIntegration_Embeddings_BasicStringInput(t *testing.T) {
	// Given: POST /v1/embeddings with a single string input
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"text-embedding-3-small","input":"hello world"}`

	// When
	resp := postJSON(t, srv.URL+"/v1/embeddings", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: 200 OK, data[0].embedding is a non-empty float array
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	if obj, _ := result["object"].(string); obj != "list" {
		t.Errorf("expected object=list, got %q", obj)
	}
	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		t.Fatal("expected non-empty data array")
	}
	item := data[0].(map[string]interface{})
	if itemObj, _ := item["object"].(string); itemObj != "embedding" {
		t.Errorf("expected data[0].object=embedding, got %q", itemObj)
	}
	embedding, ok := item["embedding"].([]interface{})
	if !ok || len(embedding) == 0 {
		t.Error("expected non-empty embedding vector")
	}
}

func TestIntegration_Embeddings_MultipleInputs(t *testing.T) {
	// Given: a request with three string inputs
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"text-embedding-3-small","input":["hello","world","foo"]}`

	// When
	resp := postJSON(t, srv.URL+"/v1/embeddings", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: data has exactly 3 embeddings
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	data, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array")
	}
	if len(data) != 3 {
		t.Errorf("expected 3 embeddings, got %d", len(data))
	}
}

func TestIntegration_Embeddings_DimensionsParameter(t *testing.T) {
	// Given: request with explicit dimensions=256
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"text-embedding-3-small","input":"hello","dimensions":256}`

	// When
	resp := postJSON(t, srv.URL+"/v1/embeddings", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: embedding vector has exactly 256 elements
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		t.Fatal("expected data array")
	}
	item := data[0].(map[string]interface{})
	embedding, ok := item["embedding"].([]interface{})
	if !ok {
		t.Fatal("expected embedding field")
	}
	if len(embedding) != 256 {
		t.Errorf("expected 256 dimensions, got %d", len(embedding))
	}
}

func TestIntegration_Embeddings_HasUsage(t *testing.T) {
	// Given: any valid embeddings request
	srv := newTestServer(t)
	defer srv.Close()
	body := `{"model":"text-embedding-3-small","input":"usage test"}`

	// When
	resp := postJSON(t, srv.URL+"/v1/embeddings", body)
	defer func() { _ = resp.Body.Close() }()

	// Then: usage field is present with prompt_tokens and total_tokens
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := mustDecodeJSON(t, resp.Body)
	usage, ok := result["usage"].(map[string]interface{})
	if !ok {
		t.Fatal("expected usage object in response")
	}
	if usage["prompt_tokens"] == nil {
		t.Error("expected prompt_tokens in usage")
	}
	if usage["total_tokens"] == nil {
		t.Error("expected total_tokens in usage")
	}
}
