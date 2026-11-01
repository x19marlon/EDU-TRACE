package similarity

import "testing"

const original = `#include <iostream>
using namespace std;
int main() {
    int n, suma = 0, mayor = -1;
    cin >> n;
    for (int i = 1; i <= n; i++) {
        int x;
        cin >> x;
        suma += x;
        if (x > mayor) mayor = x;
    }
    cout << "Suma: " << suma << endl;
    cout << "Mayor: " << mayor << endl;
    if (suma % 2 == 0) cout << "par" << endl;
    else cout << "impar" << endl;
    return 0;
}`

const renamed = `#include <iostream>
using namespace std;
// copiado y con otros nombres
int main() {
    int cantidad, total = 0, maximo = -1;
    cin >> cantidad;
    for (int k = 1; k <= cantidad; k++) {
        int valor;
        cin >> valor;
        total += valor;
        if (valor > maximo) maximo = valor;
    }
    cout << "Total: " << total << endl;
    cout << "Maximo: " << maximo << endl;
    if (total % 2 == 0) cout << "par" << endl;
    else cout << "impar" << endl;
    return 0;
}`

const different = `#include <iostream>
#include <vector>
using namespace std;
int main() {
    vector<int> datos;
    int x;
    while (cin >> x) datos.push_back(x);
    long long acumulado = 0;
    int pos = 0;
    for (size_t j = 0; j < datos.size(); j++) {
        acumulado = acumulado + datos[j];
        if (datos[j] > datos[pos]) pos = j;
    }
    cout << acumulado << " " << (datos.empty() ? 0 : datos[pos]) << "\n";
    cout << (acumulado % 2 ? "impar" : "par") << "\n";
}`

func TestCompare(t *testing.T) {
	pairs := Compare(map[string]string{"ana": original, "beto": renamed, "carla": different}, 0.5)
	if len(pairs) != 1 || pairs[0].A != "ana" || pairs[0].B != "beto" || pairs[0].Score < 0.9 {
		t.Fatalf("pares: %+v", pairs)
	}
	if got := Compare(map[string]string{"a": "int main(){}", "b": "int main(){}"}, 0.1); len(got) != 0 {
		t.Fatalf("programas demasiado cortos no se comparan: %+v", got)
	}
}

func TestTokenizeIgnoresNamesAndComments(t *testing.T) {
	a := Tokenize("int x = 5; // uno\ncout << \"hola\";")
	b := Tokenize("int valor = 99; /* dos */ cout << \"chao\";")
	if len(a) != len(b) {
		t.Fatalf("%v\n%v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("token %d: %q vs %q", i, a[i], b[i])
		}
	}
}
