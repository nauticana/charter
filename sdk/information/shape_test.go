package information

import (
	"context"
	"strings"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestShapes(t *testing.T) {
	ctx := context.Background()
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	defs := NewBaseProvider(c)
	order, err := defs.Definition(ctx, "harbor.example", model.Ref{ID: "INFO-ORDER-EXCEPTION"})
	if err != nil || order.Shape == nil {
		t.Fatalf("order exception shape: %v", err)
	}
	if problems := CheckShape(*order.Shape); len(problems) != 0 {
		t.Errorf("harbor shape: %v", problems)
	}
	bad := model.InformationShape{Properties: []model.ShapeProperty{{Name: "id", Type: "string", Required: true}, {Name: "note", Type: "string"}},
		Identity: []string{"note", "missing"}, Relationships: []model.ShapeRelationship{{Name: "id", TargetInformationID: model.Ref{ID: "X"}, Cardinality: model.CardinalityOne}}}
	if problems := CheckShape(bad); len(problems) != 3 {
		t.Errorf("duplicate name and two identity defects: %v", problems)
	}
	for path, want := range map[string]string{
		"orderId":                   "",
		"creditStatus.creditStatus": "",
		"missing":                   "has no property missing",
		"creditStatus.missing":      "has no property missing",
		"nowhere.orderId":           "has no relationship nowhere",
		"orderId.creditStatus":      "has no relationship orderId",
	} {
		_, err := Resolve(ctx, defs, order, path)
		if (want == "") != (err == nil) || err != nil && !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", path, err, want)
		}
	}
	mappings := []model.FieldMapping{{CharterPath: "orderId"}, {CharterPath: "creditStatus.rating"}}
	if problems := CheckMappings(ctx, defs, order, mappings); len(problems) != 2 || !strings.Contains(problems[1], "lineId") {
		t.Errorf("unresolved path and unmapped identity: %v", problems)
	}
	if problems := CheckMappings(ctx, defs, model.InformationDefinition{}, mappings); problems != nil {
		t.Errorf("a definition without a shape constrains no mapping: %v", problems)
	}
	schema := JSONSchema(*order.Shape)
	if schema["additionalProperties"] != false || len(schema["required"].([]string)) != 2 || schema["properties"].(map[string]any)["creditStatus"].(map[string]any)["type"] != "object" {
		t.Errorf("schema: %v", schema)
	}
}
