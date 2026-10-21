package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/marlon/edu-trace/internal/model"
)

const systemPromptChat = `Eres el tutor pedagógico inteligente de EduTrace para el curso universitario Introducción a la Programación en C++.
Tu propósito es guiar al estudiante mediante preguntas orientadoras, pistas conceptuales y analogías claras para que él mismo descubra cómo resolver su duda o problema.

REGLAS PEDAGÓGICAS ESTRICTAS:
1. NUNCA resuelvas directamente el problema del estudiante ni escribas la solución específica para su ejercicio o tarea.
2. Si tienes que mostrar código, DEBE SER ÚNICAMENTE UN EJEMPLO GENÉRICO o una analogía simplificada sobre una situación totalmente diferente.
   - Ejemplo PROHIBIDO: Si el estudiante pregunta cómo sumar los números impares en un arreglo en su tarea, no le des el for con arreglo[i]%%2 != 0.
   - Ejemplo PERMITIDO: Mostrar cómo funciona el operador módulo '%%' en un número entero aislado o cómo se itera un ciclo básico genérico.
3. No des la corrección inmediata. Haz preguntas que lleven al estudiante a reflexionar (método socrático).
4. Usa conceptos adecuados al nivel de principiante: variables, tipos primitivos, condicionales (if/else, switch), ciclos (while, for, do-while), arreglos unidimensionales/matrices y funciones. No utilices temas avanzados (punteros complejos, memoria dinámica, clases avanzadas o plantillas) a menos que sean estrictamente necesarios y acordes.
5. Sé conciso, empático, motivador y claro en tus respuestas en español.

CRITERIOS DEL CURSO:
%s`

// StreamChat envía la conversación del chat al modelo con el contexto del código y enunciado,
// garantizando la pedagogía orientada a ejemplos.
func (o *OllamaClient) StreamChat(ctx context.Context, req model.ChatRequest, statement string, onToken func(string)) error {
	var chatMsgs []chatMessage

	// 1. System Prompt
	sysPrompt := fmt.Sprintf(systemPromptChat, o.syllabus)
	chatMsgs = append(chatMsgs, chatMessage{
		Role:    "system",
		Content: sysPrompt,
	})

	// 2. Context Prompt (Código actual, stdin y enunciado si existen)
	var ctxBuilder strings.Builder
	ctxBuilder.WriteString("CONTEXTO ACTUAL DEL ESTUDIANTE EN EL EDITOR:\n")
	if strings.TrimSpace(statement) != "" {
		ctxBuilder.WriteString("\n--- ENUNCIADO DEL TALLER ---\n" + statement + "\n")
	}
	if strings.TrimSpace(req.Code) != "" {
		codeNumbered := numberLines(req.Code)
		if len(codeNumbered) > 10000 {
			codeNumbered = codeNumbered[:10000] + "\n... [código truncado]"
		}
		ctxBuilder.WriteString("\n--- CÓDIGO FUENTE ACTUAL DEL ESTUDIANTE (C++) ---\n" + codeNumbered + "\n")
	}
	if strings.TrimSpace(req.Stdin) != "" {
		stdinVal := req.Stdin
		if len(stdinVal) > maxStdinChars {
			stdinVal = stdinVal[:maxStdinChars] + "..."
		}
		ctxBuilder.WriteString("\n--- ENTRADA ESTÁNDAR (STDIN) ACTUAL ---\n" + stdinVal + "\n")
	}
	ctxBuilder.WriteString("\nRecuerda: Si el estudiante te pide resolver su código o corregir su error, NO le des la respuesta directa; guíalo con pistas y sólo con ejemplos genéricos distintos a su ejercicio.")

	chatMsgs = append(chatMsgs, chatMessage{
		Role:    "system",
		Content: ctxBuilder.String(),
	})

	// 3. Conversation Messages
	for _, m := range req.Messages {
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		chatMsgs = append(chatMsgs, chatMessage{
			Role:    role,
			Content: m.Content,
		})
	}

	return o.streamChat(ctx, chatMsgs, onToken)
}
