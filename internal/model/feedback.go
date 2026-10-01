package model

// FeedbackRequest is the JSON body for the feedback endpoint.
//
// The execution fields describe the last compilation of exactly this code;
// the frontend leaves Compiled=false when the code changed since then.
type FeedbackRequest struct {
	Code           string `json:"code"`
	Compiled       bool   `json:"compiled"`
	Success        bool   `json:"success"`
	CompilerOutput string `json:"compiler_output"`
	Stdin          string `json:"stdin"`
	ProgramOutput  string `json:"program_output"`
	ExitCode       int    `json:"exit_code"`
	RunError       string `json:"run_error"`
}

// FeedbackResult holds the AI-generated pedagogical feedback.
type FeedbackResult struct {
	Feedback string `json:"feedback"`
	Model    string `json:"model"`
	TimeMs   int64  `json:"time_ms"`
	Error    string `json:"error,omitempty"`
}
