package binding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/nauticana/charter/schema"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var ErrPackageManifest = errors.New("invalid binding package manifest")

const packageManifestSchema = "binding/package_manifest.schema.json"

const maxPackageInputBytes = 1 << 20

// PackageManifest describes one reusable realization of a capability contract. Adapter and observer references are
// opaque composition keys; credentials and provider configuration do not belong in the manifest.
type PackageManifest struct {
	FormatVersion    int                    `json:"formatVersion"`
	ID               string                 `json:"id"`
	Version          string                 `json:"version"`
	CapabilityID     model.Ref              `json:"capabilityId"`
	ContractVersion  string                 `json:"contractVersion"`
	AdapterRef       string                 `json:"adapterRef"`
	ObserverRef      string                 `json:"observerRef,omitempty"`
	ParameterSchema  json.RawMessage        `json:"parameterSchema"`
	BindingTemplates []BindingTemplate      `json:"bindingTemplates"`
	Features         []model.FeatureSupport `json:"features"`
	LossyMappings    []model.LossyMapping   `json:"lossyMappings"`
	ConformanceCases []string               `json:"conformanceCases"`
}

// BindingTemplate is a binding document plus JSON Pointer locations replaced by named installation parameters.
type BindingTemplate struct {
	Document   json.RawMessage   `json:"document"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// ParsePackageManifest validates and decodes the portable JSON format.
func ParsePackageManifest(raw []byte) (PackageManifest, error) {
	var manifest PackageManifest
	if len(raw) > maxPackageInputBytes {
		return manifest, fmt.Errorf("%w: manifest exceeds %d bytes", ErrPackageManifest, maxPackageInputBytes)
	}
	if err := validateJSON(packageManifestSchema, nil, raw); err != nil {
		return manifest, fmt.Errorf("%w: %v", ErrPackageManifest, err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return manifest, fmt.Errorf("%w: %v", ErrPackageManifest, err)
	}
	return manifest, nil
}

// Materialize resolves parameters and pins every template to the supplied customer system profile. Charter owns the
// resulting documents; downstream composition resolves AdapterRef and ObserverRef to executable implementations.
func Materialize(manifest PackageManifest, profile model.SystemProfile, params map[string]any) ([]*model.Document, error) {
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPackageManifest, err)
	}
	if len(manifestJSON) > maxPackageInputBytes {
		return nil, fmt.Errorf("%w: manifest exceeds %d bytes", ErrPackageManifest, maxPackageInputBytes)
	}
	if err := validateJSON(packageManifestSchema, nil, manifestJSON); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPackageManifest, err)
	}
	if profile.Namespace == "" || profile.ID == "" || profile.CharterSpecVersion == "" {
		return nil, fmt.Errorf("%w: profile namespace, id, and specification version are required", ErrPackageManifest)
	}
	parameterJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("%w: parameters: %v", ErrPackageManifest, err)
	}
	if len(parameterJSON) > maxPackageInputBytes {
		return nil, fmt.Errorf("%w: parameters exceed %d bytes", ErrPackageManifest, maxPackageInputBytes)
	}
	if err := validateJSONBytes(manifest.ParameterSchema, parameterJSON); err != nil {
		return nil, fmt.Errorf("%w: parameters: %v", ErrPackageManifest, err)
	}

	out := make([]*model.Document, 0, len(manifest.BindingTemplates))
	for index, template := range manifest.BindingTemplates {
		var value map[string]any
		decoder := json.NewDecoder(bytes.NewReader(template.Document))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("%w: template %d: %v", ErrPackageManifest, index, err)
		}
		pointers := make([]string, 0, len(template.Parameters))
		for pointer := range template.Parameters {
			pointers = append(pointers, pointer)
		}
		sort.Strings(pointers)
		for position, pointer := range pointers {
			for _, prior := range pointers[:position] {
				if strings.HasPrefix(pointer, prior+"/") {
					return nil, fmt.Errorf("%w: template %d parameter paths %q and %q overlap", ErrPackageManifest, index, prior, pointer)
				}
			}
			name := template.Parameters[pointer]
			parameter, ok := params[name]
			if !ok {
				return nil, fmt.Errorf("%w: template %d parameter %q is absent", ErrPackageManifest, index, name)
			}
			if err := setPointer(value, pointer, parameter); err != nil {
				return nil, fmt.Errorf("%w: template %d: %v", ErrPackageManifest, index, err)
			}
		}
		kind, _ := value["kind"].(string)
		if !isBindingKind(model.Kind(kind)) {
			return nil, fmt.Errorf("%w: template %d has unsupported kind %q", ErrPackageManifest, index, kind)
		}
		value["charterSpecVersion"] = profile.CharterSpecVersion
		value["namespace"] = profile.Namespace
		value["systemProfileId"] = profile.ID
		if kind == string(model.KindCapabilityBinding) || kind == string(model.KindAuthorityBinding) {
			value["capabilityId"] = refValue(manifest.CapabilityID)
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("%w: template %d: %v", ErrPackageManifest, index, err)
		}
		if err := validateJSON(bindingSchema(model.Kind(kind)), nil, raw); err != nil {
			return nil, fmt.Errorf("%w: template %d result: %v", ErrPackageManifest, index, err)
		}
		document, err := (corpus.Parser{}).Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: template %d: %v", ErrPackageManifest, index, err)
		}
		out = append(out, document)
	}
	return out, nil
}

func validateJSON(path string, schemaJSON, instanceJSON []byte) error {
	if schemaJSON == nil {
		var err error
		schemaJSON, err = fs.ReadFile(schema.FS, path)
		if err != nil {
			return err
		}
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	common, err := fs.ReadFile(schema.FS, "common.schema.json")
	if err != nil {
		return err
	}
	commonValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(common))
	if err != nil {
		return err
	}
	if err := compiler.AddResource("https://nauticana.github.io/charter/schema/common.schema.json", commonValue); err != nil {
		return err
	}
	schemaValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return err
	}
	if path == "" {
		return validateCompiled(compiler, "urn:charter:binding-package-parameters", schemaValue, instanceJSON)
	}
	var metadata struct {
		ID string `json:"$id"`
	}
	if err := json.Unmarshal(schemaJSON, &metadata); err != nil {
		return err
	}
	if metadata.ID == "" {
		return fmt.Errorf("schema %s has no $id", path)
	}
	return validateCompiled(compiler, metadata.ID, schemaValue, instanceJSON)
}

func validateJSONBytes(schemaJSON, instanceJSON []byte) error {
	if len(schemaJSON) == 0 {
		return errors.New("parameter schema is required")
	}
	return validateJSON("", schemaJSON, instanceJSON)
}

func validateCompiled(compiler *jsonschema.Compiler, id string, schemaValue any, instanceJSON []byte) error {
	if err := compiler.AddResource(id, schemaValue); err != nil {
		return err
	}
	compiled, err := compiler.Compile(id)
	if err != nil {
		return err
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(instanceJSON))
	if err != nil {
		return err
	}
	return compiled.Validate(instance)
}

func isBindingKind(kind model.Kind) bool {
	switch kind {
	case model.KindCapabilityBinding, model.KindDataBinding, model.KindAuthorityBinding, model.KindEventBinding:
		return true
	default:
		return false
	}
}

func bindingSchema(kind model.Kind) string {
	switch kind {
	case model.KindCapabilityBinding:
		return "binding/capability_binding.schema.json"
	case model.KindDataBinding:
		return "binding/data_binding.schema.json"
	case model.KindAuthorityBinding:
		return "binding/authority_binding.schema.json"
	case model.KindEventBinding:
		return "binding/event_binding.schema.json"
	default:
		return ""
	}
}

func refValue(ref model.Ref) any {
	if ref.Namespace == "" {
		return ref.ID
	}
	return map[string]any{"namespace": ref.Namespace, "id": ref.ID}
}

func setPointer(document map[string]any, pointer string, value any) error {
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if pointer == "" || !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("parameter path %q is not a JSON Pointer", pointer)
	}
	for i := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(parts[i], "~1", "/"), "~0", "~")
	}
	if protectedParameterPath(parts[0]) {
		return fmt.Errorf("parameter path %q targets a package-owned field", pointer)
	}
	var current any = document
	for _, part := range parts[:len(parts)-1] {
		switch node := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = node[part]
			if !ok {
				return fmt.Errorf("parameter path %q does not exist", pointer)
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(node) {
				return fmt.Errorf("parameter path %q has invalid array index", pointer)
			}
			current = node[index]
		default:
			return fmt.Errorf("parameter path %q crosses a scalar", pointer)
		}
	}
	last := parts[len(parts)-1]
	switch node := current.(type) {
	case map[string]any:
		if _, ok := node[last]; !ok {
			return fmt.Errorf("parameter path %q does not exist", pointer)
		}
		node[last] = value
	case []any:
		index, err := strconv.Atoi(last)
		if err != nil || index < 0 || index >= len(node) {
			return fmt.Errorf("parameter path %q has invalid array index", pointer)
		}
		node[index] = value
	default:
		return fmt.Errorf("parameter path %q crosses a scalar", pointer)
	}
	return nil
}

func protectedParameterPath(name string) bool {
	switch name {
	case "charterSpecVersion", "namespace", "kind", "id", "bindingVersion", "systemProfileId", "capabilityId":
		return true
	default:
		return false
	}
}
