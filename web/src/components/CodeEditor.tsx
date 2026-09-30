"use client";

import { useRef, useEffect } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { basicSetup } from "codemirror";
import { cpp } from "@codemirror/lang-cpp";
import { oneDark } from "@codemirror/theme-one-dark";

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
}

export default function CodeEditor({ value, onChange, readOnly = false }: CodeEditorProps) {
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  useEffect(() => {
    if (!editorRef.current) return;

    const state = EditorState.create({
      doc: value,
      extensions: [
        basicSetup,
        cpp(),
        oneDark,
        EditorView.editable.of(!readOnly),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            onChange(update.state.doc.toString());
          }
        }),
        EditorView.theme({
          "&": { height: "100%", fontSize: "14px" },
          ".cm-scroller": { overflow: "auto" },
          ".cm-content": { fontFamily: "'JetBrains Mono', 'Fira Code', monospace" },
          ".cm-gutters": { fontFamily: "'JetBrains Mono', 'Fira Code', monospace" },
        }),
      ],
    });

    const view = new EditorView({
      state,
      parent: editorRef.current,
    });

    viewRef.current = view;

    return () => {
      view.destroy();
    };
    // Only run on mount/unmount — value changes are handled by the editor itself.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div
      ref={editorRef}
      className="w-full h-full min-h-[400px] rounded-lg overflow-hidden border border-gray-700"
    />
  );
}
