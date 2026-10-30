package compiler

import (
	"context"
	"os/exec"
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
