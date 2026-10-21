package model

// ChatMessage represents a single message in the pedagogical chat conversation.
type ChatMessage struct {
	Role string `json:"role"` // "user" or "assistant"
	Content string `json:"content"` // Text of the message
}

// ChatRequest is the request payload for student-LLM interactive chat.
type ChatRequest struct {
	AssignmentID string `json:"assignment_id,omitempty"` // Context of the current assignment
	Code string `json:"code,omitempty"` // Current student code
	Stdin string `json:"stdin,omitempty"` // Current input
	Messages []ChatMessage `json:"messages"` // Conversation history including the new user message
}
