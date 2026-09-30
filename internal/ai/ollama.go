package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/config"
)

// OllamaClient communicates with a local Ollama instance.
type OllamaClient struct {
	cfg      *config.Config
	client   *http.Client
	syllabus string
}

// chatRequest is the JSON body sent to Ollama /api/chat.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  chatOptions   `json:"options,omitempty"`
}

type chatMessage struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	Thinking string `json:"thinking,omitempty"`
}

type chatOptions struct {
	NumCtx int `json:"num_ctx,omitempty"`
}

// chatResponse is the non-streaming response from Ollama.
type chatResponse struct {
	Message chatMessage `json:"message"`
}

// NewOllamaClient creates a client and loads syllabus documents.
func NewOllamaClient(cfg *config.Config) (*OllamaClient, error) {
	syllabus, err := loadSyllabusDocuments()
	if err != nil {
		return nil, fmt.Errorf("loading syllabus: %w", err)
	}

	slog.Info("syllabus loaded",
		"length", len(syllabus),
		"model", cfg.OllamaModel,
		"url", cfg.OllamaBaseURL,
	)

	return &OllamaClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.OllamaTimeout) * time.Second,
		},
		syllabus: syllabus,
	}, nil
}

// loadSyllabusDocuments reads all .md files from the syllabus/ directory.
func loadSyllabusDocuments() (string, error) {
	// Find syllabus directory relative to the binary or working directory.
	syllabusDir := "syllabus"

	// Try from executable location if not found in cwd.
	if _, err := os.Stat(syllabusDir); os.IsNotExist(err) {
		execPath, _ := os.Executable()
		syllabusDir = filepath.Join(filepath.Dir(execPath), "syllabus")
	}

	entries, err := os.ReadDir(syllabusDir)
	if err != nil {
		return "", fmt.Errorf("reading syllabus directory %q: %w", syllabusDir, err)
	}

	var parts []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(syllabusDir, entry.Name()))
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		parts = append(parts, fmt.Sprintf("--- %s ---\n%s", entry.Name(), string(content)))
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("no .md files found in %q", syllabusDir)
	}

	return strings.Join(parts, "\n\n"), nil
}

const systemPrompt = `Eres un tutor de Introducción a la Programación en C++. Tu rol es dar retroalimentación pedagógica constructiva a estudiantes universitarios.

REGLAS ESTRICTAS:
1. NO eres un calificador. NUNCA asignes puntajes, notas numéricas ni calificaciones.
2. Tu objetivo es AYUDAR al estudiante a mejorar, no juzgarlo.
3. Responde SIEMPRE en español.
4. Sé específico: menciona líneas o secciones concretas del código.
5. Estructura tu respuesta en estas secciones:
   - **✅ Lo que está bien**: Reconoce los aciertos del estudiante.
   - **💡 Sugerencias de mejora**: Explica qué puede mejorar y POR QUÉ.
   - **📚 Conceptos a repasar**: Si aplica, menciona temas que el estudiante debería revisar.
6. Usa un tono amigable y motivador, como un tutor paciente.
7. Si el código no compila, enfócate en los errores de compilación y cómo corregirlos.
8. Basa tu retroalimentación en los criterios del syllabus proporcionado.

CRITERIOS DEL SYLLABUS:
%s`

// streamChunk is one JSON line from Ollama's streaming response.
type streamChunk struct {
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// BuildMessages constructs the chat messages for a feedback request.
func (o *OllamaClient) BuildMessages(code string, compilerOutput string, success bool) []chatMessage {
	var userMsg strings.Builder
	userMsg.WriteString("Analiza el siguiente código C++ de un estudiante y da retroalimentación pedagógica.\n\n")

	if !success && compilerOutput != "" {
		userMsg.WriteString("⚠️ El código NO compiló. Errores del compilador:\n```\n")
		userMsg.WriteString(compilerOutput)
		userMsg.WriteString("\n```\n\n")
	} else if compilerOutput != "" {
		userMsg.WriteString("⚠️ Advertencias del compilador:\n```\n")
		userMsg.WriteString(compilerOutput)
		userMsg.WriteString("\n```\n\n")
	}

	userMsg.WriteString("Código del estudiante:\n```cpp\n")
	userMsg.WriteString(code)
	userMsg.WriteString("\n```")

	return []chatMessage{
		{
			Role:    "system",
			Content: fmt.Sprintf(systemPrompt, o.syllabus),
		},
		{
			Role:    "user",
			Content: userMsg.String(),
		},
	}
}

// StreamFeedback sends student code to Ollama and streams the response token by token.
// The onToken callback is called for each chunk of text as it arrives.
func (o *OllamaClient) StreamFeedback(ctx context.Context, code string, compilerOutput string, success bool, onToken func(token string)) error {
	messages := o.BuildMessages(code, compilerOutput, success)

	reqBody := chatRequest{
		Model:    o.cfg.OllamaModel,
		Messages: messages,
		Stream:   true,
		Options: chatOptions{
			NumCtx: o.cfg.OllamaContextSize,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	url := o.cfg.OllamaBaseURL + "/api/chat"
	slog.Info("sending streaming feedback request to Ollama",
		"url", url,
		"model", o.cfg.OllamaModel,
		"code_length", len(code),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming NDJSON response line by line.
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var chunk streamChunk
		if err := decoder.Decode(&chunk); err != nil {
			return fmt.Errorf("decoding stream chunk: %w", err)
		}

		if chunk.Message.Content != "" {
			onToken(chunk.Message.Content)
		}

		if chunk.Done {
			break
		}
	}

	return nil
}

// Ping checks if Ollama is reachable and the model is available.
func (o *OllamaClient) Ping(ctx context.Context) error {
	url := o.cfg.OllamaBaseURL + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach Ollama at %s: %w", o.cfg.OllamaBaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	return nil
}

