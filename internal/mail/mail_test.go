package mail

import (
	"errors"
	"io"
	"mime/quotedprintable"
	"net"
	"net/textproto"
	"strings"
	"testing"
)

func fakeSMTP(t *testing.T, rcptCode int) (int, <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		tp.PrintfLine("220 prueba")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			switch cmd := strings.ToUpper(strings.Fields(line + " x")[0]); cmd {
			case "EHLO", "HELO":
				tp.PrintfLine("250 prueba")
			case "RCPT":
				if rcptCode >= 500 {
					tp.PrintfLine("%d buzón inexistente", rcptCode)
					continue
				}
				tp.PrintfLine("250 ok")
			case "DATA":
				tp.PrintfLine("354 adelante")
				data, _ := io.ReadAll(tp.DotReader())
				got <- string(data)
				tp.PrintfLine("250 recibido")
			case "QUIT":
				tp.PrintfLine("221 adiós")
				return
			default:
				tp.PrintfLine("250 ok")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, got
}

func TestSend(t *testing.T) {
	port, got := fakeSMTP(t, 250)
	cfg := Config{Host: "127.0.0.1", Port: port, From: "EduTrace <no-responder@u.edu>"}
	if !cfg.Enabled() || (Config{Host: "x"}).Enabled() {
		t.Fatal("Enabled necesita servidor y remitente")
	}
	err := cfg.Send(Message{To: "ana@u.edu", Subject: "Revisión de «Taller 1»\r\nBcc: espia@x.com", Body: "Hola, Ana:\n\nTu profesor devolvió la revisión."})
	if err != nil {
		t.Fatal(err)
	}
	raw := <-got
	header, body, ok := strings.Cut(raw, "\n\n")
	if !ok {
		t.Fatalf("mensaje sin separación de cabeceras: %q", raw)
	}
	for _, line := range strings.Split(header, "\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Fatalf("se inyectó una cabecera: %q", header)
		}
	}
	if !strings.Contains(header, "Subject: =?utf-8?q?") || !strings.Contains(header, "To: <ana@u.edu>") || !strings.Contains(header, "Auto-Submitted: auto-generated") {
		t.Fatalf("cabeceras: %q", header)
	}
	decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
	if err != nil || !strings.Contains(string(decoded), "devolvió la revisión") {
		t.Fatalf("cuerpo: %q %v", decoded, err)
	}
}

func TestSendRejected(t *testing.T) {
	port, _ := fakeSMTP(t, 550)
	err := Config{Host: "127.0.0.1", Port: port, From: "no-responder@u.edu"}.Send(Message{To: "nadie@u.edu", Subject: "x", Body: "x"})
	if err == nil || !Permanent(err) {
		t.Fatalf("un 550 es un rechazo definitivo: %v", err)
	}
	if Permanent(errors.New("conexión rechazada")) {
		t.Fatal("un error de red se reintenta")
	}
	if _, err := (Config{From: "no es correo"}).build(Message{To: "a@u.edu"}); err == nil {
		t.Fatal("aceptó un remitente inválido")
	}
}
