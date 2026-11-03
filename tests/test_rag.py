"""Regression tests that do not download models or contact external services."""

import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import AsyncMock, patch

from src.rag import retrieve as retrieval
from src.rag import service


class RetrievalTests(unittest.IsolatedAsyncioTestCase):
    async def test_invalid_input_does_not_open_store(self):
        with patch.object(retrieval.lancedb, "connect_async", new_callable=AsyncMock) as connect:
            for question, top_k in [("  ", 5), ("test", 0), ("test", 21), ("x" * 2001, 5)]:
                with self.assertRaises(ValueError):
                    await retrieval.retrieve(question, top_k)
            connect.assert_not_called()

    async def test_missing_index_has_actionable_error(self):
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(retrieval, "LANCEDB_URI", Path(directory) / "missing"):
                with self.assertRaisesRegex(RuntimeError, "cocoindex update"):
                    await retrieval.retrieve("rúbrica")

    async def test_empty_table_does_not_load_model(self):
        table = SimpleNamespace(count_rows=AsyncMock(return_value=0))
        connection = SimpleNamespace(open_table=AsyncMock(return_value=table))
        with tempfile.TemporaryDirectory() as directory:
            with (
                patch.object(retrieval, "LANCEDB_URI", Path(directory)),
                patch.object(retrieval.lancedb, "connect_async", AsyncMock(return_value=connection)),
                patch.object(retrieval, "get_embedder") as embedder,
            ):
                self.assertEqual(await retrieval.retrieve("rúbrica"), [])
                embedder.assert_not_called()

    async def test_store_failure_is_explained(self):
        with tempfile.TemporaryDirectory() as directory:
            with (
                patch.object(retrieval, "LANCEDB_URI", Path(directory)),
                patch.object(retrieval.lancedb, "connect_async", AsyncMock(side_effect=OSError("bad store"))),
            ):
                with self.assertRaisesRegex(RuntimeError, "índice local"):
                    await retrieval.retrieve("rúbrica")


class ServiceTests(unittest.IsolatedAsyncioTestCase):
    async def test_empty_retrieval_does_not_call_llm(self):
        with (
            patch.object(service, "retrieve", AsyncMock(return_value=[])),
            patch.object(service, "_complete") as complete,
        ):
            self.assertIn("No encontré contexto", await service.ask_rag("otra cosa"))
            complete.assert_not_called()

    async def test_answer_contains_traceable_sources(self):
        chunk = retrieval.RetrievedChunk(1, "src/classroom/rubrica.md", "Máximo 10 puntos", 0, 16, 0.7)
        with (
            patch.object(service, "retrieve", AsyncMock(return_value=[chunk])),
            patch.object(service, "_complete", return_value="El máximo es 10 puntos [1].") as complete,
        ):
            answer = await service.ask_rag("¿Puntaje máximo?")
            self.assertIn("10 puntos [1]", answer)
            self.assertIn("src/classroom/rubrica.md (caracteres 0–16)", answer)
            self.assertIn(chunk.text, complete.call_args.args[1])

    def test_rag_does_not_require_deepseek_key(self):
        with patch.dict(os.environ, {"DEEPSEEK_API_KEY": ""}):
            with patch.object(service, "chat", return_value="respuesta") as chat:
                self.assertEqual(service._complete("pregunta", "contexto"), "respuesta")
                self.assertIn("contexto", chat.call_args.args[0][1]["content"])


if __name__ == "__main__":
    unittest.main()
