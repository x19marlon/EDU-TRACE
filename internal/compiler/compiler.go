package compiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/config"
	"github.com/marlon/edu-trace/internal/model"
)

// Compiler handles C++ compilation and execution.
type Compiler struct {
	cfg *config.Config
	slots chan struct{}
}

const slotWait = 60 * time.Second

// New creates a Compiler with the given configuration.
func New(cfg *config.Config) *Compiler {
	n := cfg.MaxSandboxes
	if n <= 0 {
		n = runtime.NumCPU()
	}
	return &Compiler{cfg: cfg, slots: make(chan struct{}, n)}
}

// Compile compiles and runs C++ source code in a temporary sandbox.
func (c *Compiler) Compile(ctx context.Context, sourceCode string, stdin string) *model.CompileResult {
	return c.CompileAndTest(ctx, sourceCode, stdin, nil)
}

func (c *Compiler) CompileAndTest(ctx context.Context, sourceCode string, stdin string, tests []model.TestCase) *model.CompileResult {
	result := &model.CompileResult{}

	wait := time.NewTimer(slotWait)
	defer wait.Stop()
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-wait.C:
		result.Error = "el servidor está ocupado compilando otros programas, intenta de nuevo en un minuto"
		return result
	case <-ctx.Done():
		result.Error = "se canceló la compilación"
		return result
	}

	// Create isolated temp directory.
	tmpDir, err := os.MkdirTemp("", "edutrace-*")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create temp directory: %v", err)
		return result
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "main.cpp")

	// Write source code.
	if err := os.WriteFile(srcPath, []byte(sourceCode), 0600); err != nil {
		result.Error = fmt.Sprintf("failed to write source file: %v", err)
		return result
	}

	// --- Compilation phase ---
	compileStart := time.Now()
	compileCtx, compileCancel := context.WithTimeout(ctx, c.cfg.CompileTimeout)
	defer compileCancel()

	name, args := c.compileCommand(tmpDir)
	compileCmd := exec.CommandContext(compileCtx, name, args...)
	compileCmd.Dir = tmpDir
	compileCmd.WaitDelay = time.Second

	compileStdout := newLimitedBuffer(c.cfg.MaxOutputSize)
	compileStderr := newLimitedBuffer(c.cfg.MaxOutputSize)
	compileCmd.Stdout = compileStdout
	compileCmd.Stderr = compileStderr

	compileErr := compileCmd.Run()
	result.CompileTimeMs = time.Since(compileStart).Milliseconds()

	// Merge stdout+stderr from compiler (g++ writes diagnostics to stderr).
	compilerOutput := compileStderr.String() + compileStdout.String()
	if len(compilerOutput) > c.cfg.MaxOutputSize {
		compilerOutput = compilerOutput[:c.cfg.MaxOutputSize] + "\n... (salida recortada)"
	}
	result.CompilerOutput = compilerOutput

	if compileErr != nil {
		// Compilation failed — return errors to the student.
		if compileCtx.Err() == context.DeadlineExceeded {
			result.Error = "la compilación tardó demasiado"
		}
		result.ExitCode = exitCodeFrom(compileCmd)
		for _, t := range tests {
			result.Tests = append(result.Tests, testResult(t, runOutcome{}, "el programa no compiló"))
		}
		result.Explanations = Explain(result)
		return result
	}

	// --- Execution phase ---
	result.Success = true

	run := c.run(ctx, tmpDir, stdin)
	result.ProgramOutput, result.ExitCode, result.Error, result.RunTimeMs = run.output, run.exitCode, run.err, run.ms
	result.Tests = c.runTests(ctx, tmpDir, tests)
	result.Explanations = Explain(result)
	return result
}

type runOutcome struct {
	output string
	exitCode int
	err string
	ms int64
	timedOut bool
}

func (c *Compiler) run(ctx context.Context, tmpDir, stdin string) runOutcome {
	var out runOutcome
	runStart := time.Now()
	runCtx, runCancel := context.WithTimeout(ctx, c.cfg.RunTimeout)
	defer runCancel()

	name, args := c.runCommand(tmpDir)
	runCmd := exec.CommandContext(runCtx, name, args...)
	runCmd.Dir = tmpDir
	runCmd.WaitDelay = time.Second

	// Pipe stdin to the program if provided.
	if stdin != "" {
		runCmd.Stdin = strings.NewReader(stdin)
	}

	// Buffers acotados: un programa que imprime en bucle no puede agotar la memoria del servidor.
	runStdout := newLimitedBuffer(c.cfg.MaxOutputSize)
	runStderr := newLimitedBuffer(c.cfg.MaxOutputSize)
	runCmd.Stdout = runStdout
	runCmd.Stderr = runStderr

	runErr := runCmd.Run()
	out.ms = time.Since(runStart).Milliseconds()

	programOutput := runStdout.String() + runStderr.String()
	if len(programOutput) > c.cfg.MaxOutputSize {
		programOutput = programOutput[:c.cfg.MaxOutputSize] + "\n... (salida recortada)"
	}
	out.output = programOutput

	if runErr != nil {
		out.exitCode = exitCodeFrom(runCmd)
		if runCtx.Err() == context.DeadlineExceeded {
			// Success sigue en true: compiló bien, el problema fue al ejecutar.
			out.err = "se agotó el tiempo de ejecución (¿un ciclo infinito o esperando entrada?)"
			out.timedOut = true
		}
	}
	return out
}

const (
	maxTestOutput = 2000
	maxTimeouts = 2
)

func (c *Compiler) runTests(ctx context.Context, tmpDir string, tests []model.TestCase) []model.TestResult {
	var results []model.TestResult
	timeouts := 0
	for _, t := range tests {
		if timeouts >= maxTimeouts || ctx.Err() != nil {
			results = append(results, testResult(t, runOutcome{}, "no se ejecutó: el programa no terminó en los casos anteriores"))
			continue
		}
		out := c.run(ctx, tmpDir, t.Stdin)
		if out.timedOut {
			timeouts++
		}
		results = append(results, testResult(t, out, ""))
	}
	return results
}

func testResult(t model.TestCase, out runOutcome, skipped string) model.TestResult {
	r := model.TestResult{TestID: t.ID, Name: t.Name, Hidden: t.Hidden, Stdin: t.Stdin, Expected: t.Expected, Own: t.Own}
	switch {
	case skipped != "":
		r.Error = skipped
		return r
	case out.err != "":
		r.Error = out.err
	case out.exitCode != 0:
		r.Error = fmt.Sprintf("el programa terminó con código de salida %d", out.exitCode)
	}
	r.Output = out.output
	if len(r.Output) > maxTestOutput {
		r.Output = strings.ToValidUTF8(r.Output[:maxTestOutput], "") + "\n... (salida recortada)"
	}
	r.Passed = r.Error == "" && OutputMatches(t.Expected, out.output, t.Match)
	return r
}

func normalizeOutput(s string) string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			lines = append(lines, strings.Join(f, " "))
		}
	}
	return strings.Join(lines, "\n")
}

func OutputMatches(expected, output, match string) bool {
	exp, got := normalizeOutput(expected), normalizeOutput(output)
	if match == model.MatchExact {
		return exp == got
	}
	return exp != "" && strings.Contains(got, exp)
}

// limitedBuffer guarda como máximo max bytes y descarta el resto sin fallar,
// para que el proceso hijo no se bloquee al escribir.
type limitedBuffer struct {
	buf bytes.Buffer
	max int
	truncated bool
}

func newLimitedBuffer(max int) *limitedBuffer { return &limitedBuffer{max: max} }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.max + 1 - b.buf.Len(); room > 0 {
		if len(p) > room {
			b.buf.Write(p[:room])
			b.truncated = true
		} else {
			b.buf.Write(p)
		}
	} else {
		b.truncated = true
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string { return b.buf.String() }

// exitCodeFrom extracts the exit code from a finished command.
func exitCodeFrom(cmd *exec.Cmd) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return -1
}
