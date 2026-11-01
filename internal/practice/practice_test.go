package practice

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/config"
)

var solutions = map[string]string{
	"suma-1-n": `#include <iostream>
int main() { long long n, s = 0; std::cin >> n; for (long long i = 1; i <= n; i++) s += i; std::cout << s << "\n"; }`,
	"par-impar": `#include <iostream>
int main() { int n; std::cin >> n; std::cout << (n % 2 == 0 ? "par" : "impar") << "\n"; }`,
	"mayor-de-tres": `#include <iostream>
#include <algorithm>
int main() { int a, b, c; std::cin >> a >> b >> c; std::cout << std::max(a, std::max(b, c)) << "\n"; }`,
	"celsius-fahrenheit": `#include <iostream>
#include <iomanip>
int main() { double c; std::cin >> c; std::cout << std::fixed << std::setprecision(1) << c * 9 / 5 + 32 << "\n"; }`,
	"factorial": `#include <iostream>
int main() { int n; long long f = 1; std::cin >> n; for (int i = 2; i <= n; i++) f *= i; std::cout << f << "\n"; }`,
	"tabla-multiplicar": `#include <iostream>
int main() { int n; std::cin >> n; for (int i = 1; i <= 10; i++) std::cout << n << " x " << i << " = " << n * i << "\n"; }`,
	"bisiesto": `#include <iostream>
int main() { int y; std::cin >> y; bool b = (y % 4 == 0 && y % 100 != 0) || y % 400 == 0; std::cout << (b ? "si" : "no") << "\n"; }`,
	"contar-digitos": `#include <iostream>
int main() { long long n; int d = 1; std::cin >> n; while (n >= 10) { n /= 10; d++; } std::cout << d << "\n"; }`,
	"invertir-numero": `#include <iostream>
int main() { long long n, r = 0; std::cin >> n; while (n > 0) { r = r * 10 + n % 10; n /= 10; } std::cout << r << "\n"; }`,
	"es-primo": `#include <iostream>
int main() { int n; std::cin >> n; bool p = n > 1; for (int i = 2; i * i <= n; i++) if (n % i == 0) p = false; std::cout << (p ? "si" : "no") << "\n"; }`,
	"fibonacci": `#include <iostream>
int main() { int n; long long a = 0, b = 1; std::cin >> n; for (int i = 0; i < n; i++) { std::cout << a << " "; long long t = a + b; a = b; b = t; } std::cout << "\n"; }`,
	"promedio-notas": `#include <iostream>
#include <iomanip>
int main() { int n; double s = 0, x; std::cin >> n; for (int i = 0; i < n; i++) { std::cin >> x; s += x; } std::cout << std::fixed << std::setprecision(2) << s / n << "\n"; }`,
	"maximo-arreglo": `#include <iostream>
int main() { int n, x, m = 0, p = 0; std::cin >> n; for (int i = 1; i <= n; i++) { std::cin >> x; if (i == 1 || x > m) { m = x; p = i; } } std::cout << m << " " << p << "\n"; }`,
	"palindromo": `#include <iostream>
#include <string>
int main() { std::string s; std::cin >> s; bool p = true; for (size_t i = 0; i < s.size() / 2; i++) if (s[i] != s[s.size() - 1 - i]) p = false; std::cout << (p ? "si" : "no") << "\n"; }`,
}

func TestBank(t *testing.T) {
	all := All()
	if len(all) != len(solutions) {
		t.Fatalf("hay %d ejercicios y %d soluciones de referencia", len(all), len(solutions))
	}
	for _, e := range all {
		for _, test := range e.Tests {
			if test.Hidden && (test.Stdin != "" || test.Expected != "") {
				t.Fatalf("%s: la lista pública muestra un caso oculto", e.ID)
			}
		}
	}
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ no está instalado")
	}
	c := compiler.New(&config.Config{GppPath: "g++", Sandbox: "none", CompileTimeout: 30 * time.Second, RunTimeout: 3 * time.Second, MaxOutputSize: 10240})
	for id, code := range solutions {
		e, ok := Get(id)
		if !ok {
			t.Fatalf("falta el ejercicio %s", id)
		}
		res := c.CompileAndTest(context.Background(), code, "", e.Tests)
		for _, r := range res.Tests {
			if !r.Passed {
				t.Errorf("%s, caso %q: la solución de referencia no pasa (salida %q, esperada %q, error %q)", id, r.Name, r.Output, r.Expected, r.Error)
			}
		}
	}
}
