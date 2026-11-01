package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/marlon/edu-trace/internal/model"
)

const (
	ModeHint = "pista"
	MaxHintLevel = 3
	maxHintOutput = 2000
)

const systemPromptHint = `Eres el tutor de EduTrace para el curso Introducción a la Programación en C++. El estudiante pidió UNA pista para avanzar por su cuenta. Nunca escribas la solución ni el código corregido de su ejercicio.

Niveles de pista:
1. General: dile qué parte del problema o del programa conviene revisar, sin decir cuál es el error.
2. Más concreta: señala la zona del código (con números de línea) y la idea que falla, con una pregunta que lo guíe.
3. Muy concreta: explica con palabras qué debe cambiar, sin escribir el código corregido. Si ayuda, usa un ejemplo genérico distinto de su ejercicio.

Responde en español, de "tú", en máximo 80 palabras, sin títulos ni listas. Da solo la pista del nivel pedido y no repitas las pistas anteriores.`

var hintLevels = [MaxHintLevel + 1]string{"", "1 (general)", "2 (más concreta)", "3 (muy concreta)"}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "") + "\n[recortado]"
}

func (o *OllamaClient) hintMessages(req model.FeedbackRequest, statement string) []chatMessage {
	var b strings.Builder
	if strings.TrimSpace(statement) != "" {
		b.WriteString("ENUNCIADO:\n" + clip(statement, 3000) + "\n\n")
	}
	b.WriteString("CÓDIGO DEL ESTUDIANTE:\n" + clip(numberLines(req.Code), 10000) + "\n\n")
	switch {
	case !req.Compiled:
		b.WriteString("Todavía no compiló esta versión del código.\n")
	case !req.Success:
		b.WriteString("NO COMPILA. Mensajes de g++:\n" + clip(req.CompilerOutput, maxHintOutput) + "\n")
	default:
		if req.Stdin != "" {
			b.WriteString("Entrada:\n" + clip(req.Stdin, maxStdinChars) + "\n")
		}
		b.WriteString("Salida:\n" + clip(req.ProgramOutput, maxHintOutput) + "\n")
		if req.RunError != "" {
			b.WriteString("Error al ejecutar: " + req.RunError + "\n")
		}
		writeTests(&b, req.Tests)
	}
	for i, p := range req.Previous {
		fmt.Fprintf(&b, "\nPista %d que ya recibió: %s\n", i+1, clip(p, 600))
	}
	fmt.Fprintf(&b, "\nDa ahora la pista de nivel %s.", hintLevels[req.Level])
	return []chatMessage{{Role: "system", Content: systemPromptHint}, {Role: "user", Content: b.String()}}
}

func (o *OllamaClient) StreamHint(ctx context.Context, req model.FeedbackRequest, statement string, onToken func(string)) error {
	if req.Level < 1 || req.Level > MaxHintLevel {
		return fmt.Errorf("nivel de pista inválido: %d", req.Level)
	}
	messages := o.hintMessages(req, statement)
	if !o.fits(messages) {
		return ErrPromptTooLong
	}
	guard := NewSolutionGuard(onToken, req.Code, statement)
	err := o.streamChat(ctx, messages, guard.Write)
	guard.Close()
	if n := guard.Omitted(); n > 0 {
		slog.Info("filtro anti-solución: código omitido en una pista", "fragmentos", n)
	}
	return err
}
