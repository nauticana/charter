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
	if problems := CheckMappings(ctx, defs, order, mappings); len(problems) != 4 || !strings.Contains(problems[1], "lineId") ||
		!strings.Contains(problems[2], "creditStatus.customerId") || !strings.Contains(problems[3], "creditStatus.creditStatus") {
		t.Errorf("unresolved path, unmapped identity, and unmapped target properties: %v", problems)
	}
	if problems := CheckMappings(ctx, defs, order, []model.FieldMapping{{CharterPath: "orderId"}, {CharterPath: "lineId"}}); len(problems) != 0 {
		t.Errorf("a relationship no path reaches requires nothing: %v", problems)
	}
	dangling := order
	dangling.Shape = &model.InformationShape{Properties: order.Shape.Properties, Identity: order.Shape.Identity,
		Relationships: []model.ShapeRelationship{{Name: "lost", TargetInformationID: model.Ref{ID: "INFO-NONE"}, Cardinality: model.CardinalityOne}}}
	if problems := CheckMappings(ctx, defs, dangling, []model.FieldMapping{{CharterPath: "orderId"}, {CharterPath: "lineId"}}); len(problems) != 1 || !strings.Contains(problems[0], "relationship lost") {
		t.Errorf("unresolved relationship target: %v", problems)
	}
	if problems := CheckMappings(ctx, defs, model.InformationDefinition{}, mappings); problems != nil {
		t.Errorf("a definition without a shape constrains no mapping: %v", problems)
	}
	schema := JSONSchema(*order.Shape)
	if schema["additionalProperties"] != false || len(schema["required"].([]string)) != 2 || schema["properties"].(map[string]any)["creditStatus"].(map[string]any)["type"] != "object" {
		t.Errorf("schema: %v", schema)
	}
}
