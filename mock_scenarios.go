package main

import "openai-mokku/api"

// extractLastUserMessage returns the content of the last user-role message.
func extractLastUserMessage(messages []api.ChatCompletionRequestMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == api.ChatCompletionRequestMessageRoleUser {
			return messages[i].Content
		}
	}
	return ""
}
