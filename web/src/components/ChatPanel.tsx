"use client";

import { useState, useRef, useEffect } from "react";
import { ChatMessage, streamChat } from "@/lib/api";
import { Spinner } from "@/components/ui";
import { formatFeedback } from "@/lib/format";

interface ChatPanelProps {
  code: string;
  stdin: string;
  assignmentId?: string;
}

export default function ChatPanel({ code, stdin, assignmentId }: ChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading]);

  const handleSend = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    const trimmed = input.trim();
    if (!trimmed || isLoading) return;

    const userMsg: ChatMessage = { role: "user", content: trimmed };
    const updatedMessages = [...messages, userMsg];
    setMessages(updatedMessages);
    setInput("");
    setError(null);
    setIsLoading(true);

    let assistantReply = "";
    // Insert empty placeholder assistant message
    setMessages((prev) => [...prev, { role: "assistant", content: "" }]);

    try {
      await streamChat(
        {
          assignment_id: assignmentId,
          code,
          stdin,
          messages: updatedMessages,
        },
        (token) => {
          assistantReply += token;
          setMessages((prev) => {
            const next = [...prev];
            if (next.length > 0 && next[next.length - 1].role === "assistant") {
              next[next.length - 1] = { role: "assistant", content: assistantReply };
            }
            return next;
          });
        }
      );
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : "Error al conectar con la IA";
      setError(errMsg);
      // Remove empty assistant message if it failed before generating anything
      if (!assistantReply) {
        setMessages((prev) => prev.slice(0, -1));
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleClear = () => {
    if (isLoading) return;
    setMessages([]);
    setError(null);
  };

  return (
    <div className="flex h-[480px] flex-col bg-white">
      {/* Encabezado pedagógico */}
      <div className="flex items-center justify-between border-b border-lavender-100 px-4 py-2 bg-lavender-50/50">
        <div className="flex items-center gap-2">
          <span className="flex h-2 w-2 rounded-full bg-mint-500 animate-pulse" />
          <span className="text-xs font-bold text-lavender-900">Tutor Pedagógico</span>
          <span className="text-[11px] text-ink-faint">
            (Pistas conceptuales y ejemplos ilustrativos)
          </span>
        </div>
        {messages.length > 0 && (
          <button
            onClick={handleClear}
            disabled={isLoading}
            className="text-xs font-semibold text-ink-faint hover:text-ink hover:underline disabled:opacity-50"
          >
            Limpiar chat
          </button>
        )}
      </div>

      {/* Lista de mensajes */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center text-center p-6 text-ink-soft">
            <div className="rounded-full bg-lavender-100 p-3 mb-3 text-lavender-600">
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
            </div>
            <h3 className="text-sm font-bold text-ink">¿Tienes dudas con tu código?</h3>
            <p className="text-xs text-ink-faint mt-1 max-w-xs">
              Pregúntame sobre conceptos, errores o lógica. Recuerda que no te daré la solución directa de tu tarea, sino pistas y ejemplos guía para que aprendas a resolverla.
            </p>
          </div>
        ) : (
          messages.map((msg, idx) => {
            const isUser = msg.role === "user";
            return (
              <div
                key={idx}
                className={`flex flex-col ${isUser ? "items-end" : "items-start"}`}
              >
                <div
                  className={`max-w-[85%] rounded-2xl px-4 py-2.5 text-sm ${
                    isUser
                      ? "bg-lavender-600 text-white rounded-br-none whitespace-pre-wrap"
                      : "bg-lavender-50 text-ink rounded-bl-none border border-lavender-100 leading-relaxed"
                  }`}
                >
                  {isUser ? (
                    msg.content
                  ) : msg.content ? (
                    <div
                      className="chat-markdown"
                      dangerouslySetInnerHTML={{
                        __html:
                          formatFeedback(msg.content) +
                          (isLoading && idx === messages.length - 1
                            ? '<span class="inline-block w-1.5 h-3 ml-1 bg-lavender-500 animate-pulse align-middle"></span>'
                            : ""),
                      }}
                    />
                  ) : (
                    <span className="flex items-center gap-1.5 text-lavender-500 py-1">
                      <Spinner className="h-3.5 w-3.5 border-current" />
                      <span className="text-xs">Pensando...</span>
                    </span>
                  )}
                </div>
              </div>
            );
          })
        )}

        {error && (
          <div className="rounded-2xl bg-peach-50 p-3 text-xs text-peach-700 font-medium">
            {error}
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input del Chat */}
      <form
        onSubmit={handleSend}
        className="border-t border-lavender-100 p-3 bg-white flex items-center gap-2"
      >
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Haz una pregunta sobre tu código o conceptos de C++..."
          disabled={isLoading}
          className="flex-1 rounded-full border border-lavender-200 bg-white px-4 py-2 text-sm text-ink placeholder-ink-faint focus:border-lavender-400 focus:outline-none focus:ring-2 focus:ring-lavender-100 disabled:opacity-60"
        />
        <button
          type="submit"
          disabled={isLoading || !input.trim()}
          className="flex h-9 w-9 items-center justify-center rounded-full bg-lavender-600 text-white transition hover:bg-lavender-700 disabled:opacity-50"
          title="Enviar mensaje"
        >
          {isLoading ? (
            <Spinner className="h-4 w-4 border-white" />
          ) : (
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
            </svg>
          )}
        </button>
      </form>
    </div>
  );
}
