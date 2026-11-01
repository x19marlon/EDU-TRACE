package practice

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/marlon/edu-trace/internal/model"
)

//go:embed ejercicios.json
var raw []byte

type Exercise struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Topic string `json:"topic"`
	Level int `json:"level"`
	Statement string `json:"statement"`
	Tests []model.TestCase `json:"tests"`
}

var bank, index = load()

func load() ([]Exercise, map[string]int) {
	var list []Exercise
	if err := json.Unmarshal(raw, &list); err != nil {
		panic(fmt.Sprintf("ejercicios.json: %v", err))
	}
	index := map[string]int{}
	for i := range list {
		index[list[i].ID] = i
		for j := range list[i].Tests {
			list[i].Tests[j].ID = fmt.Sprintf("%s-%d", list[i].ID, j+1)
			list[i].Tests[j].Match = model.MatchExact
		}
	}
	return list, index
}

func Get(id string) (Exercise, bool) {
	i, ok := index[id]
	if !ok {
		return Exercise{}, false
	}
	return bank[i], true
}

func (e Exercise) Public() Exercise {
	tests := make([]model.TestCase, len(e.Tests))
	for i, t := range e.Tests {
		if t.Hidden {
			t = model.TestCase{ID: t.ID, Name: t.Name, Hidden: true}
		}
		tests[i] = t
	}
	e.Tests = tests
	return e
}

func All() []Exercise {
	out := make([]Exercise, len(bank))
	for i, e := range bank {
		out[i] = e.Public()
	}
	return out
}
