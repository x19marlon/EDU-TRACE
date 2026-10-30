package model

// CompileRequest is the JSON body sent by the frontend.
type CompileRequest struct {
	Code string `json:"code"`
	Stdin string `json:"stdin"`
	// Clase y taller elegidos por el estudiante: si está inscrito, la compilación
	// se guarda como intento para que su profesor vea el proceso.
	GroupID string `json:"group_id"`
	AssignmentID string `json:"assignment_id"`
}

// CompileResult holds the outcome of compiling and running C++ code.
type CompileResult struct {
	Success bool `json:"success"`
	CompilerOutput string `json:"compiler_output"`
	ProgramOutput string `json:"program_output"`
	ExitCode int `json:"exit_code"`
	CompileTimeMs int64 `json:"compile_time_ms"`
	RunTimeMs int64 `json:"run_time_ms"`
	Error string `json:"error,omitempty"`
	Tests []TestResult `json:"tests,omitempty"`
}

const (
	MatchContains = "contiene"
	MatchExact = "exacta"
)

type TestCase struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Stdin string `json:"stdin"`
	Expected string `json:"expected"`
	Match string `json:"match"`
	Hidden bool `json:"hidden"`
}

type TestResult struct {
	TestID string `json:"test_id"`
	Name string `json:"name"`
	Hidden bool `json:"hidden"`
	Passed bool `json:"passed"`
	Stdin string `json:"stdin,omitempty"`
	Expected string `json:"expected,omitempty"`
	Output string `json:"output,omitempty"`
	Error string `json:"error,omitempty"`
}

func (r TestResult) Public() TestResult {
	if !r.Hidden {
		return r
	}
	return TestResult{TestID: r.TestID, Name: r.Name, Hidden: true, Passed: r.Passed}
}

func PublicResults(results []TestResult) []TestResult {
	out := make([]TestResult, len(results))
	for i, r := range results {
		out[i] = r.Public()
	}
	return out
}

// ErrorResponse is returned for client or server errors.
type ErrorResponse struct {
	Error string `json:"error"`
	Details string `json:"details,omitempty"`
}
