package practice

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/config"
)

var solutions = map[string]string{
	"saludo": `#include <iostream>
#include <string>
int main() { std::string n; std::cin >> n; std::cout << "Hola, " << n << "!\n"; }`,
	"orden-inverso": `#include <iostream>
int main() { int a, b, c; std::cin >> a >> b >> c; std::cout << c << " " << b << " " << a << "\n"; }`,
	"datos-estudiante": `#include <iostream>
int main() { int e; double h; std::cin >> e >> h; std::cout << "Edad: " << e << "\nEstatura: " << h << "\n"; }`,
	"intercambiar": `#include <iostream>
int main() { int a, b; std::cin >> a >> b; int t = a; a = b; b = t; std::cout << a << " " << b << "\n"; }`,
	"parte-entera": `#include <iostream>
int main() { double d; std::cin >> d; int i = d; std::cout << i << "\n"; }`,
	"saldo": `#include <iostream>
int main() { int s, d, r; std::cin >> s >> d >> r; s += d; s -= r; std::cout << s << "\n"; }`,
	"division-residuo": `#include <iostream>
int main() { int a, b; std::cin >> a >> b; std::cout << a / b << " " << a % b << "\n"; }`,
	"area-circulo": `#include <iostream>
#include <iomanip>
int main() { double r; std::cin >> r; std::cout << std::fixed << std::setprecision(2) << 3.14159 * r * r << "\n"; }`,
	"segundos-hms": `#include <iostream>
int main() { int s; std::cin >> s; std::cout << s / 3600 << " " << s % 3600 / 60 << " " << s % 60 << "\n"; }`,
	"aprobado": `#include <iostream>
int main() { double n; std::cin >> n; std::cout << (n >= 3.0 ? "aprobado" : "reprobado") << "\n"; }`,
	"tipo-triangulo": `#include <iostream>
int main() { int a, b, c; std::cin >> a >> b >> c; if (a == b && b == c) std::cout << "equilatero"; else if (a == b || b == c || a == c) std::cout << "isosceles"; else std::cout << "escaleno"; std::cout << "\n"; }`,
	"cuadrado-funcion": `#include <iostream>
int cuadrado(int x) { return x * x; }
int main() { int n; std::cin >> n; std::cout << cuadrado(n) << "\n"; }`,
	"potencia-funcion": `#include <iostream>
long long potencia(int b, int e) { long long r = 1; for (int i = 0; i < e; i++) r *= b; return r; }
int main() { int b, e; std::cin >> b >> e; std::cout << potencia(b, e) << "\n"; }`,
	"mcd-funcion": `#include <iostream>
int mcd(int a, int b) { while (b) { int t = a % b; a = b; b = t; } return a; }
int main() { int a, b; std::cin >> a >> b; std::cout << mcd(a, b) << "\n"; }`,
	"primos-rango": `#include <iostream>
bool esPrimo(int n) { if (n < 2) return false; for (int i = 2; i * i <= n; i++) if (n % i == 0) return false; return true; }
int main() { int a, b; std::cin >> a >> b; for (int i = a; i <= b; i++) if (esPrimo(i)) std::cout << i << " "; std::cout << "\n"; }`,
	"invertir-arreglo": `#include <iostream>
int main() { int n, d[100]; std::cin >> n; for (int i = 0; i < n; i++) std::cin >> d[i]; for (int i = n - 1; i >= 0; i--) std::cout << d[i] << " "; std::cout << "\n"; }`,
	"cuantos-aprobaron": `#include <iostream>
int main() { int n, c = 0; double x; std::cin >> n; for (int i = 0; i < n; i++) { std::cin >> x; if (x >= 3.0) c++; } std::cout << c << "\n"; }`,
	"suma-arreglo-funcion": `#include <iostream>
int suma(int d[], int n) { int s = 0; for (int i = 0; i < n; i++) s += d[i]; return s; }
int main() { int n, d[100]; std::cin >> n; for (int i = 0; i < n; i++) std::cin >> d[i]; std::cout << suma(d, n) << "\n"; }`,
	"buscar-elemento": `#include <iostream>
int buscar(int d[], int n, int x) { for (int i = 0; i < n; i++) if (d[i] == x) return i + 1; return -1; }
int main() { int n, x, d[100]; std::cin >> n; for (int i = 0; i < n; i++) std::cin >> d[i]; std::cin >> x; std::cout << buscar(d, n, x) << "\n"; }`,
	"ordenar-arreglo": `#include <iostream>
void ordenar(int d[], int n) { for (int i = 0; i < n; i++) for (int j = 0; j + 1 < n - i; j++) if (d[j] > d[j + 1]) { int t = d[j]; d[j] = d[j + 1]; d[j + 1] = t; } }
int main() { int n, d[100]; std::cin >> n; for (int i = 0; i < n; i++) std::cin >> d[i]; ordenar(d, n); for (int i = 0; i < n; i++) std::cout << d[i] << " "; std::cout << "\n"; }`,
	"suma-matriz": `#include <iostream>
int main() { int f, c, x, s = 0; std::cin >> f >> c; for (int i = 0; i < f * c; i++) { std::cin >> x; s += x; } std::cout << s << "\n"; }`,
	"diagonal-principal": `#include <iostream>
int main() { int n, m[20][20], s = 0; std::cin >> n; for (int i = 0; i < n; i++) for (int j = 0; j < n; j++) std::cin >> m[i][j]; for (int i = 0; i < n; i++) s += m[i][i]; std::cout << s << "\n"; }`,
	"transpuesta": `#include <iostream>
int main() { int f, c, m[20][20]; std::cin >> f >> c; for (int i = 0; i < f; i++) for (int j = 0; j < c; j++) std::cin >> m[i][j]; for (int j = 0; j < c; j++) { for (int i = 0; i < f; i++) std::cout << m[i][j] << " "; std::cout << "\n"; } }`,
	"rectangulo-struct": `#include <iostream>
struct Rectangulo { int ancho, alto; };
int main() { Rectangulo r; std::cin >> r.ancho >> r.alto; std::cout << r.ancho * r.alto << " " << 2 * (r.ancho + r.alto) << "\n"; }`,
	"distancia-puntos": `#include <iostream>
#include <iomanip>
#include <cmath>
struct Punto { double x, y; };
int main() { Punto a, b; std::cin >> a.x >> a.y >> b.x >> b.y; std::cout << std::fixed << std::setprecision(2) << std::sqrt((a.x - b.x) * (a.x - b.x) + (a.y - b.y) * (a.y - b.y)) << "\n"; }`,
	"mejor-estudiante": `#include <iostream>
#include <string>
struct Estudiante { std::string nombre; double nota; };
int main() { int n; std::cin >> n; Estudiante m{"", -1}; for (int i = 0; i < n; i++) { Estudiante e; std::cin >> e.nombre >> e.nota; if (e.nota > m.nota) m = e; } std::cout << m.nombre << "\n"; }`,
	"guardar-y-sumar": `#include <iostream>
#include <fstream>
int main() { int n, x, s = 0; std::cin >> n; std::ofstream o("/tmp/numeros.txt"); for (int i = 0; i < n; i++) { std::cin >> x; o << x << "\n"; } o.close(); std::ifstream f("/tmp/numeros.txt"); while (f >> x) s += x; std::cout << s << "\n"; }`,
	"contar-lineas": `#include <iostream>
#include <fstream>
#include <string>
int main() { std::string l; std::ofstream o("/tmp/lineas.txt"); while (std::getline(std::cin, l)) o << l << "\n"; o.close(); std::ifstream f("/tmp/lineas.txt"); int c = 0; while (std::getline(f, l)) c++; std::cout << c << "\n"; }`,
	"palabra-mas-larga": `#include <iostream>
#include <fstream>
#include <string>
int main() { std::string p, m; std::ofstream o("/tmp/palabras.txt"); while (std::cin >> p) o << p << "\n"; o.close(); std::ifstream f("/tmp/palabras.txt"); while (f >> p) if (p.size() > m.size()) m = p; std::cout << m << "\n"; }`,
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
	topics := []string{}
	for _, e := range all {
		if len(topics) == 0 || topics[len(topics)-1] != e.Topic {
			topics = append(topics, e.Topic)
		}
	}
	want := "Entrada y salida,Variables,Aritmética,Condicionales,Ciclos,Funciones,Arreglos,Arreglos y funciones,Matrices,Estructuras,Archivos de texto"
	if got := strings.Join(topics, ","); got != want {
		t.Fatalf("orden de los temas: %s", got)
	}
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
