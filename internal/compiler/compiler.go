package compiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/config"
	"github.com/marlon/edu-trace/internal/model"
)

// Compiler handles C++ compilation and execution.
type Compiler struct {
	cfg *config.Config
}

// New creates a Compiler with the given configuration.
func New(cfg *config.Config) *Compiler {
	return &Compiler{cfg: cfg}
}

// Compile compiles and runs C++ source code in a temporary sandbox.
func (c *Compiler) Compile(ctx context.Context, sourceCode string, stdin string) *model.CompileResult {
	result := &model.CompileResult{}

	// Create isolated temp directory.
	tmpDir, err := os.MkdirTemp("", "edutrace-*")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create temp directory: %v", err)
		return result
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "main.cpp")
	binPath := filepath.Join(tmpDir, "main")

	// Write source code.
	if err := os.WriteFile(srcPath, []byte(sourceCode), 0600); err != nil {
		result.Error = fmt.Sprintf("failed to write source file: %v", err)
		return result
	}

	// --- Compilation phase ---
	compileStart := time.Now()
	compileCtx, compileCancel := context.WithTimeout(ctx, c.cfg.CompileTimeout)
	defer compileCancel()

	compileCmd := exec.CommandContext(compileCtx, c.cfg.GppPath,
		"-o", binPath,
		"-Wall",
		"-Wextra",
		"-std=c++17",
		srcPath,
	)

	var compileStdout, compileStderr bytes.Buffer
	compileCmd.Stdout = &compileStdout
	compileCmd.Stderr = &compileStderr

	compileErr := compileCmd.Run()
	result.CompileTimeMs = time.Since(compileStart).Milliseconds()

	// Merge stdout+stderr from compiler (g++ writes diagnostics to stderr).
	compilerOutput := compileStderr.String() + compileStdout.String()
	if len(compilerOutput) > c.cfg.MaxOutputSize {
		compilerOutput = compilerOutput[:c.cfg.MaxOutputSize] + "\n... (output truncated)"
	}
	result.CompilerOutput = compilerOutput

	if compileErr != nil {
		// Compilation failed — return errors to the student.
		if compileCtx.Err() == context.DeadlineExceeded {
			result.Error = "compilation timed out"
		}
		result.ExitCode = exitCodeFrom(compileCmd)
		return result
	}

	// --- Execution phase ---
	result.Success = true

	runStart := time.Now()
	runCtx, runCancel := context.WithTimeout(ctx, c.cfg.RunTimeout)
	defer runCancel()

	runCmd := exec.CommandContext(runCtx, binPath)
	runCmd.Dir = tmpDir

	// Pipe stdin to the program if provided.
	if stdin != "" {
		runCmd.Stdin = strings.NewReader(stdin)
	}

	var runStdout, runStderr bytes.Buffer
	runCmd.Stdout = &runStdout
	runCmd.Stderr = &runStderr

	runErr := runCmd.Run()
	result.RunTimeMs = time.Since(runStart).Milliseconds()

	programOutput := runStdout.String() + runStderr.String()
	if len(programOutput) > c.cfg.MaxOutputSize {
		programOutput = programOutput[:c.cfg.MaxOutputSize] + "\n... (output truncated)"
	}
	result.ProgramOutput = programOutput

	if runErr != nil {
		result.ExitCode = exitCodeFrom(runCmd)
		if runCtx.Err() == context.DeadlineExceeded {
			result.Error = "program execution timed out"
			result.Success = false
		}
		return result
	}

	result.ExitCode = 0
	return result
}

// exitCodeFrom extracts the exit code from a finished command.
func exitCodeFrom(cmd *exec.Cmd) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return -1
}
