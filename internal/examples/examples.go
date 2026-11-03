package examples

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed ejemplos.json
var raw []byte

// Example es un ejemplo de sintaxis del catálogo. Output es la salida real del programa
// con Stdin como entrada: la prueba del paquete comprueba que siga siendo cierta.
type Example struct {
	ID          string `json:"id"`
	Topic       string `json:"topic"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Code        string `json:"code"`
	Stdin       string `json:"stdin"`
	Output      string `json:"output"`
}

var catalog, index = load()

func load() ([]Example, map[string]bool) {
	var list []Example
	if err := json.Unmarshal(raw, &list); err != nil {
		panic(fmt.Sprintf("ejemplos.json: %v", err))
	}
	index := map[string]bool{}
	for _, e := range list {
		index[e.ID] = true
	}
	return list, index
}

// All devuelve el catálogo en el orden en que se ven los temas en el curso.
func All() []Example {
	return append([]Example(nil), catalog...)
}

// Exists indica si el id corresponde a un ejemplo del catálogo.
func Exists(id string) bool {
	return index[id]
}

// Filter devuelve, en el orden del catálogo, los ejemplos con esos ids. Ignora los
// que ya no estén en el catálogo.
func Filter(ids []string) []Example {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	out := []Example{}
	for _, e := range catalog {
		if wanted[e.ID] {
			out = append(out, e)
		}
	}
	return out
}
