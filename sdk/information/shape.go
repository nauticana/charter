package information

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nauticana/charter/sdk/model"
)

// CheckShape reports names declared twice and identity names that are not required properties (CHR-INFO-010).
func CheckShape(shape model.InformationShape) []string {
	var out []string
	seen := map[string]bool{}
	names := make([]string, 0, len(shape.Properties)+len(shape.Relationships))
	for _, p := range shape.Properties {
		names = append(names, p.Name)
	}
	for _, r := range shape.Relationships {
		names = append(names, r.Name)
	}
	for _, n := range names {
		if seen[n] {
			out = append(out, fmt.Sprintf("%s is declared twice", n))
		}
		seen[n] = true
	}
	for _, n := range shape.Identity {
		if k := slices.IndexFunc(shape.Properties, func(p model.ShapeProperty) bool { return p.Name == n }); k < 0 || !shape.Properties[k].Required {
			out = append(out, fmt.Sprintf("identity %s is not a required property", n))
		}
	}
	return out
}

// Resolve follows a dot-separated path through relationships to a property of the definition's shape (CHR-INFO-010).
func Resolve(ctx context.Context, defs Provider, def model.InformationDefinition, path string) (model.ShapeProperty, error) {
	segments := strings.Split(path, ".")
	for _, name := range segments[:len(segments)-1] {
		if def.Shape == nil {
			return model.ShapeProperty{}, fmt.Errorf("%s: %s has no shape", path, def.ID)
		}
		k := slices.IndexFunc(def.Shape.Relationships, func(r model.ShapeRelationship) bool { return r.Name == name })
		if k < 0 {
			return model.ShapeProperty{}, fmt.Errorf("%s: %s has no relationship %s", path, def.ID, name)
		}
		target, err := defs.Definition(ctx, def.Namespace, def.Shape.Relationships[k].TargetInformationID)
		if err != nil {
			return model.ShapeProperty{}, fmt.Errorf("%s: relationship %s: %w", path, name, err)
		}
		def = target
	}
	if def.Shape == nil {
		return model.ShapeProperty{}, fmt.Errorf("%s: %s has no shape", path, def.ID)
	}
	last := segments[len(segments)-1]
	k := slices.IndexFunc(def.Shape.Properties, func(p model.ShapeProperty) bool { return p.Name == last })
	if k < 0 {
		return model.ShapeProperty{}, fmt.Errorf("%s: %s has no property %s", path, def.ID, last)
	}
	return def.Shape.Properties[k], nil
}

// CheckMappings reports every charterPath that does not resolve, every relationship target that does not resolve, and
// every required property, identity included, that the mappings of a binding to def, or of a target they reach, leave
// unmapped (CHR-INFO-010).
func CheckMappings(ctx context.Context, defs Provider, def model.InformationDefinition, mappings []model.FieldMapping) []string {
	if def.Shape == nil {
		return nil
	}
	var out []string
	mapped := map[string]bool{}
	for _, m := range mappings {
		mapped[m.CharterPath] = true
		if _, err := Resolve(ctx, defs, def, m.CharterPath); err != nil {
			out = append(out, err.Error())
		}
	}
	checkRequired(ctx, defs, def, "", mapped, map[string]bool{}, &out)
	return out
}

// checkRequired reports the required properties of def, and of each relationship target a mapping reaches, left unmapped.
func checkRequired(ctx context.Context, defs Provider, def model.InformationDefinition, prefix string, mapped, seen map[string]bool, out *[]string) {
	key := def.Namespace + "\x00" + def.ID
	if def.Shape == nil || seen[key] {
		return
	}
	seen[key] = true
	defer delete(seen, key)
	for _, p := range def.Shape.Properties {
		path := p.Name
		if prefix != "" {
			path = prefix + "." + path
		}
		if p.Required && !mapped[path] {
			*out = append(*out, fmt.Sprintf("required property %s of %s is not mapped", path, def.ID))
		}
	}
	for _, r := range def.Shape.Relationships {
		target, err := defs.Definition(ctx, def.Namespace, r.TargetInformationID)
		if err != nil {
			*out = append(*out, fmt.Sprintf("%s: relationship %s: %v", def.ID, r.Name, err))
			continue
		}
		path := r.Name
		if prefix != "" {
			path = prefix + "." + path
		}
		if traversed(mapped, path) {
			checkRequired(ctx, defs, target, path, mapped, seen, out)
		}
	}
}

func traversed(mapped map[string]bool, relationship string) bool {
	for p := range mapped {
		if strings.HasPrefix(p, relationship+".") {
			return true
		}
	}
	return false
}

// JSONSchema is the JSON Schema an instance of the shape satisfies; relationships are objects, or arrays of objects
// when many, and undeclared members are refused (CHR-INFO-003).
func JSONSchema(shape model.InformationShape) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, p := range shape.Properties {
		props[p.Name] = map[string]any{"type": p.Type}
		if p.Required {
			required = append(required, p.Name)
		}
	}
	for _, r := range shape.Relationships {
		if r.Cardinality == model.CardinalityMany {
			props[r.Name] = map[string]any{"type": "array", "items": map[string]any{"type": "object"}}
		} else {
			props[r.Name] = map[string]any{"type": "object"}
		}
	}
	return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
}
