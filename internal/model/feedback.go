package model

// FeedbackRequest is the JSON body for the feedback endpoint.
type FeedbackRequest struct {
	Code           string `json:"code"`
	CompilerOutput string `json:"compiler_output"`
	Success        bool   `json:"success"`
}

// FeedbackResult holds the AI-generated pedagogical feedback.
type FeedbackResult struct {
	Feedback string `json:"feedback"`
	Model    string `json:"model"`
	TimeMs   int64  `json:"time_ms"`
	Error    string `json:"error,omitempty"`
}
