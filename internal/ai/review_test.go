package ai

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestReviewSchemaFor(t *testing.T) {
	if !bytes.Equal(reviewSchemaFor(0), reviewSchema) {
		t.Fatal("sin RAE debe usar el esquema base")
	}
	var schema struct {
		Properties map[string]struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(reviewSchemaFor(2), &schema); err != nil {
		t.Fatal(err)
	}
	levels := schema.Properties["niveles_rae"]
	if len(levels.Required) != 2 || levels.Required[1] != "rae_2" || len(levels.Properties["rae_1"].Enum) != 3 {
		t.Fatalf("niveles_rae: %+v", levels)
	}
	if schema.Required[len(schema.Required)-1] != "niveles_rae" || len(schema.Properties) != 5 {
		t.Fatalf("esquema: %+v", schema)
	}
	if _, ok := schema.Properties["nota"]; !ok {
		t.Fatal("se perdió la nota")
	}
}
