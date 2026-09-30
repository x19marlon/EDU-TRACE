package model

// CompileRequest is the JSON body sent by the frontend.
type CompileRequest struct {
	Code  string `json:"code"`
	Stdin string `json:"stdin"`
}

// CompileResult holds the outcome of compiling and running C++ code.
type CompileResult struct {
	Success        bool   `json:"success"`
	CompilerOutput string `json:"compiler_output"`
	ProgramOutput  string `json:"program_output"`
	ExitCode       int    `json:"exit_code"`
	CompileTimeMs  int64  `json:"compile_time_ms"`
	RunTimeMs      int64  `json:"run_time_ms"`
	Error          string `json:"error,omitempty"`
}

// ErrorResponse is returned for client or server errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
