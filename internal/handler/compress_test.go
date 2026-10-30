package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompressMiddleware(t *testing.T) {
	h := compressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			writeJSON(w, http.StatusOK, map[string]string{"texto": strings.Repeat("retroalimentación ", 200)})
		case "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte("data: hola\n\n"))
			w.(http.Flusher).Flush()
		case "/vacio":
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	get := func(path, encoding string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if encoding != "" {
			req.Header.Set("Accept-Encoding", encoding)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/json", "gzip, deflate")
	if rec.Header().Get("Content-Encoding") != "gzip" || rec.Body.Len() > 500 {
		t.Fatalf("JSON sin comprimir: %v %d bytes", rec.Header(), rec.Body.Len())
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if !strings.Contains(string(body), "retroalimentación") {
		t.Fatalf("contenido descomprimido: %q", body[:40])
	}
	if rec := get("/json", ""); rec.Header().Get("Content-Encoding") != "" {
		t.Fatal("comprimió sin que el cliente lo aceptara")
	}
	if rec := get("/sse", "gzip"); rec.Header().Get("Content-Encoding") != "" || rec.Body.String() != "data: hola\n\n" || !rec.Flushed {
		t.Fatalf("el streaming no debe comprimirse: %v %q", rec.Header(), rec.Body.String())
	}
	if rec := get("/vacio", "gzip"); rec.Code != http.StatusNoContent || rec.Header().Get("Content-Encoding") != "" || rec.Body.Len() != 0 {
		t.Fatalf("204: %v %d", rec.Header(), rec.Body.Len())
	}
}
