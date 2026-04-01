package main

import (
	"testing"

	"openai-mokku/api"
)

// --- extractLastUserMessage ---

func TestExtractLastUserMessage_SingleUserMessage_ReturnsContent(t *testing.T) {
	// Given
	messages := []api.ChatCompletionRequestMessage{
		{Role: api.ChatCompletionRequestMessageRoleUser, Content: "hello"},
	}
	// When
	got := extractLastUserMessage(messages)
	// Then
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestExtractLastUserMessage_MultipleMessages_ReturnsLastUser(t *testing.T) {
	// Given: system, then two user messages interleaved with assistant
	messages := []api.ChatCompletionRequestMessage{
		{Role: api.ChatCompletionRequestMessageRoleSystem, Content: "you are helpful"},
		{Role: api.ChatCompletionRequestMessageRoleUser, Content: "first message"},
		{Role: api.ChatCompletionRequestMessageRoleAssistant, Content: "first response"},
		{Role: api.ChatCompletionRequestMessageRoleUser, Content: "second message"},
	}
	// When
	got := extractLastUserMessage(messages)
	// Then
	if got != "second message" {
		t.Errorf("expected 'second message', got %q", got)
	}
}

func TestExtractLastUserMessage_NoUserMessages_ReturnsEmpty(t *testing.T) {
	// Given: only system messages
	messages := []api.ChatCompletionRequestMessage{
		{Role: api.ChatCompletionRequestMessageRoleSystem, Content: "system only"},
	}
	// When
	got := extractLastUserMessage(messages)
	// Then
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractLastUserMessage_EmptyMessages_ReturnsEmpty(t *testing.T) {
	// Given: empty slice
	// When
	got := extractLastUserMessage([]api.ChatCompletionRequestMessage{})
	// Then
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
