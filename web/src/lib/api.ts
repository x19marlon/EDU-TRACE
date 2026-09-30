export interface CompileResult {
  success: boolean;
  compiler_output: string;
  program_output: string;
  exit_code: number;
  compile_time_ms: number;
  run_time_ms: number;
  error?: string;
}

export interface FeedbackMeta {
  model: string;
  time_ms: number;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function compileCode(code: string, stdin: string = ""): Promise<CompileResult> {
  const response = await fetch(`${API_BASE}/api/compile`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code, stdin }),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(
      errorData?.error || `Server error: ${response.status}`
    );
  }

  return response.json();
}

/**
 * Streams AI feedback via Server-Sent Events.
 * onToken is called for each chunk of text as it arrives.
 * Returns metadata (model, time) when the stream completes.
 */
export async function streamFeedback(
  code: string,
  compilerOutput: string,
  success: boolean,
  onToken: (token: string) => void
): Promise<FeedbackMeta> {
  const response = await fetch(`${API_BASE}/api/feedback`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      code,
      compiler_output: compilerOutput,
      success,
    }),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    throw new Error(
      errorData?.error || `Server error: ${response.status}`
    );
  }

  const reader = response.body?.getReader();
  if (!reader) throw new Error("No response stream");

  const decoder = new TextDecoder();
  let meta: FeedbackMeta = { model: "", time_ms: 0 };
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });

    // Parse SSE events from buffer.
    const lines = buffer.split("\n");
    buffer = lines.pop() || ""; // Keep incomplete line in buffer.

    let eventType = "message";

    for (const line of lines) {
      if (line.startsWith("event: ")) {
        eventType = line.slice(7).trim();
      } else if (line.startsWith("data: ")) {
        const data = line.slice(6);

        if (eventType === "done") {
          try {
            meta = JSON.parse(data);
          } catch { /* ignore */ }
        } else if (eventType === "error") {
          try {
            const errObj = JSON.parse(data);
            throw new Error(errObj.error || data);
          } catch {
            throw new Error(data);
          }
        } else {
          try {
            const parsed = JSON.parse(data);
            if (parsed && typeof parsed.token === "string") {
              onToken(parsed.token);
            } else if (typeof parsed === "string") {
              onToken(parsed);
            }
          } catch {
            onToken(data);
          }
        }

        eventType = "message";
      }
    }
  }

  return meta;
}
