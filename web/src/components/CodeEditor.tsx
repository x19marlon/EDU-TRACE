"use client";

import { useRef, useEffect } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { basicSetup } from "codemirror";
import { cpp } from "@codemirror/lang-cpp";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags as t } from "@lezer/highlight";

// Tema claro a juego con la paleta pastel (colores con contraste suficiente sobre blanco).
const pastelTheme = EditorView.theme({
  "&": { height: "100%", fontSize: "14px", backgroundColor: "#FFFFFF", color: "#2E2A3F" },
  ".cm-scroller": { overflow: "auto" },
  ".cm-content": { fontFamily: "'JetBrains Mono', 'Fira Code', monospace", caretColor: "#7154CC" },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "#7154CC", borderLeftWidth: "2px" },
  ".cm-gutters": {
    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
    backgroundColor: "#F6F3FE",
    color: "#A29DB3",
    border: "none",
  },
  ".cm-activeLine": { backgroundColor: "#F6F3FE80" },
  ".cm-activeLineGutter": { backgroundColor: "#EDE7FC", color: "#5C42AE" },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
    backgroundColor: "#DCD2F8 !important",
  },
  ".cm-matchingBracket": { backgroundColor: "#DDF3EA", outline: "1px solid #9AD6BE" },
  "&.cm-focused": { outline: "none" },
});

const pastelHighlight = HighlightStyle.define([
  { tag: [t.keyword, t.controlKeyword, t.modifier], color: "#7154CC", fontWeight: "600" },
  { tag: [t.typeName, t.standard(t.typeName)], color: "#2D7258" },
  { tag: [t.string, t.character], color: "#A4522C" },
  { tag: [t.number, t.bool, t.null], color: "#C9683A" },
  { tag: [t.comment, t.lineComment, t.blockComment], color: "#A29DB3", fontStyle: "italic" },
  { tag: [t.processingInstruction, t.meta], color: "#3A8F6F" },
  { tag: [t.function(t.variableName), t.function(t.propertyName)], color: "#5C42AE" },
  { tag: [t.operator, t.punctuation], color: "#6B6680" },
  { tag: t.variableName, color: "#2E2A3F" },
]);

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
        pastelTheme,
        syntaxHighlighting(pastelHighlight),
        EditorView.editable.of(!readOnly),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            onChange(update.state.doc.toString());
          }
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
      className="w-full h-full min-h-[400px] overflow-hidden"
    />
  );
}
