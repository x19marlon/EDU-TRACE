package compiler

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/marlon/edu-trace/internal/config"
	"github.com/marlon/edu-trace/internal/model"
)

func TestOutputMatches(t *testing.T) {
	cases := []struct {
		expected, output, match string
		want bool
	}{
		{"Suma: 15", "Ingrese n: Suma: 15\n", model.MatchContains, true},
		{"Suma: 15", "Ingrese n: Suma:   15\r\n", model.MatchContains, true},
		{"Suma: 15", "Suma: 16", model.MatchContains, false},
		{"Suma: 15", "Ingrese n: Suma: 15", model.MatchExact, false},
		{"a\n\nb", "  a  \nb\n\n", model.MatchExact, true},
		{"", "algo", model.MatchContains, false},
	}
	for _, c := range cases {
		if got := OutputMatches(c.expected, c.output, c.match); got != c.want {
			t.Errorf("OutputMatches(%q, %q, %s) = %v", c.expected, c.output, c.match, got)
		}
	}
}

func TestCompileAndTest(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ no está instalado")
	}
	c := New(&config.Config{GppPath: "g++", Sandbox: "none", CompileTimeout: 30 * time.Second, RunTimeout: 2 * time.Second, MaxOutputSize: 10240})
	code := `#include <iostream>
using namespace std;
int main() {
    int n, suma = 0;
    cin >> n;
    if (n < 0) { while (true) {} }
    for (int i = 1; i < n; i++) suma += i;
    cout << "Suma: " << suma << endl;
    return 0;
}`
	tests := []model.TestCase{
		{ID: "a", Name: "Uno", Stdin: "1", Expected: "Suma: 1"},
		{ID: "b", Name: "Cero", Stdin: "0", Expected: "Suma: 0", Hidden: true},
		{ID: "c", Name: "Cinco", Stdin: "5", Expected: "Suma: 15", Match: model.MatchExact},
		{ID: "d", Name: "Negativo", Stdin: "-1", Expected: "Suma: 0"},
		{ID: "e", Name: "Negativo 2", Stdin: "-2", Expected: "Suma: 0"},
		{ID: "f", Name: "Después", Stdin: "3", Expected: "Suma: 6"},
	}
	res := c.CompileAndTest(context.Background(), code, "4", tests)
	if !res.Success || res.ProgramOutput != "Suma: 6\n" || len(res.Tests) != len(tests) {
		t.Fatalf("resultado: %+v", res)
	}
	passed := []bool{false, true, false, false, false, false}
	for i, r := range res.Tests {
		if r.Passed != passed[i] || r.TestID != tests[i].ID {
			t.Errorf("caso %s: %+v", tests[i].Name, r)
		}
	}
	if res.Tests[3].Error == "" || res.Tests[5].Error == "" || res.Tests[5].Output != "" {
		t.Fatalf("tras dos tiempos agotados no debe seguir ejecutando: %+v", res.Tests[3:])
	}
	if pub := model.PublicResults(res.Tests); pub[1].Stdin != "" || pub[1].Expected != "" || pub[1].Output != "" || pub[0].Stdin != "1" {
		t.Fatalf("resultados públicos: %+v", pub[:2])
	}

	broken := c.CompileAndTest(context.Background(), "int main() { return x; }", "", tests[:2])
	if broken.Success || len(broken.Tests) != 2 || broken.Tests[0].Passed || broken.Tests[0].Error == "" {
		t.Fatalf("si no compila, los casos deben fallar: %+v", broken.Tests)
	}
}

func TestNormalizeAndExplainErrors(t *testing.T) {
	cases := []struct{ in, norm, explains string }{
		{"‘suma’ was not declared in this scope", "'…' was not declared in this scope", "no está declarada"},
		{"'cout' was not declared in this scope", "'cout' was not declared in this scope", "#include <iostream>"},
		{"expected ‘;’ before ‘return’", "expected ';' before 'return'", "punto y coma"},
		{"no match for ‘operator>>’ (operand types are ‘std::ostream’ and ‘int’)", "no match for 'operator>>' (operand types are '…' and 'int')", "cin >> x"},
		{"expected ‘,’ or ‘;’ before ‘for’", "expected ',' or ';' before 'for'", "punto y coma"},
		{"algo raro", "algo raro", ""},
	}
	for _, c := range cases {
		got := NormalizeError(c.in)
		if got != c.norm {
			t.Errorf("NormalizeError(%q) = %q", c.in, got)
		}
		if exp := ExplainError(got); (c.explains == "") != (exp == "") || !strings.Contains(exp, c.explains) {
			t.Errorf("ExplainError(%q) = %q", got, exp)
		}
	}
	if msg, _ := DescribeRuntime("", 136); msg != "División entera por cero" {
		t.Errorf("136: %q", msg)
	}
	if msg, _ := DescribeRuntime("se agotó el tiempo de ejecución", -1); !strings.Contains(msg, "tiempo") {
		t.Errorf("tiempo: %q", msg)
	}
}

func TestSandboxLimits(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ no está instalado")
	}
	if err := CheckSandbox("bwrap"); err != nil {
		t.Skipf("sin sandbox: %v", err)
	}
	c := New(&config.Config{GppPath: "g++", Sandbox: "bwrap", CompileTimeout: 30 * time.Second, RunTimeout: 5 * time.Second, MaxOutputSize: 10240, MaxSandboxes: 2})
	code := `#include <csignal>
#include <cstdio>
#include <sys/wait.h>
#include <unistd.h>
int main() {
    signal(SIGXFSZ, SIG_IGN);
    int hijos = 0;
    for (int i = 0; i < 200; i++) {
        pid_t p = fork();
        if (p == 0) { sleep(1); _exit(0); }
        if (p < 0) break;
        hijos++;
    }
    while (wait(nullptr) > 0) {}
    static char buf[65536];
    long total = 0;
    for (int f = 0; f < 32; f++) {
        char name[32];
        snprintf(name, sizeof name, "/tmp/f%d", f);
        FILE* out = fopen(name, "w");
        if (!out) break;
        for (int i = 0; i < 16; i++) {
            size_t n = fwrite(buf, 1, sizeof buf, out);
            total += n;
            if (n < sizeof buf) break;
        }
        fclose(out);
    }
    printf("%d %ld\n", hijos, total);
}`
	res := c.CompileAndTest(context.Background(), code, "", nil)
	if !res.Success {
		t.Fatalf("no compiló o no corrió: %+v", res)
	}
	var hijos int
	var total int64
	if _, err := fmt.Sscan(res.ProgramOutput, &hijos, &total); err != nil {
		t.Fatalf("salida %q: %v", res.ProgramOutput, err)
	}
	t.Logf("procesos hijos: %d, bytes en /tmp: %d", hijos, total)
	if hijos >= runMaxProcs {
		t.Errorf("creó %d procesos; el límite es %d", hijos, runMaxProcs)
	}
	if TmpLimited() && total > runTmpBytes {
		t.Errorf("escribió %d bytes en /tmp; el límite es %d", total, runTmpBytes)
	}
}

func TestExplain(t *testing.T) {
	res := &model.CompileResult{CompilerOutput: "main.cpp: In function 'int main()':\nmain.cpp:5:5: error: 'cout' was not declared in this scope\nmain.cpp:5:5: error: 'cout' was not declared in this scope\nmain.cpp:7:1: error: expected ';' before '}' token\nmain.cpp:9:3: error: algo que no conocemos"}
	got := Explain(res)
	if len(got) != 2 || got[0].Line != 5 || !strings.Contains(got[0].Explanation, "#include <iostream>") || got[1].Line != 7 {
		t.Fatalf("explicaciones: %+v", got)
	}
	run := Explain(&model.CompileResult{Success: true, ExitCode: 136})
	if len(run) != 1 || run[0].Message != "División entera por cero" {
		t.Fatalf("error al ejecutar: %+v", run)
	}
	if len(Explain(&model.CompileResult{Success: true})) != 0 {
		t.Fatal("un programa que funciona no necesita explicaciones")
	}
}
