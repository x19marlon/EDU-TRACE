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
		return result
	}

	// --- Execution phase ---
	result.Success = true

	runStart := time.Now()
	runCtx, runCancel := context.WithTimeout(ctx, c.cfg.RunTimeout)
	defer runCancel()

	name, args = c.runCommand(tmpDir)
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
	result.RunTimeMs = time.Since(runStart).Milliseconds()

	programOutput := runStdout.String() + runStderr.String()
	if len(programOutput) > c.cfg.MaxOutputSize {
		programOutput = programOutput[:c.cfg.MaxOutputSize] + "\n... (salida recortada)"
	}
	result.ProgramOutput = programOutput

	if runErr != nil {
		result.ExitCode = exitCodeFrom(runCmd)
		if runCtx.Err() == context.DeadlineExceeded {
			// Success sigue en true: compiló bien, el problema fue al ejecutar.
			result.Error = "se agotó el tiempo de ejecución (¿un ciclo infinito o esperando entrada?)"
		}
		return result
	}

	result.ExitCode = 0
	return result
}

// limitedBuffer guarda como máximo max bytes y descarta el resto sin fallar,
// para que el proceso hijo no se bloquee al escribir.
type limitedBuffer struct {
	buf       bytes.Buffer
	max       int
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
