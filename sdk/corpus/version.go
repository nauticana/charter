package corpus

import "github.com/nauticana/charter/sdk/model"

// definitionVersions names the property through which each versioned definition kind declares the version its
// instances and references name; a changed definition is a new version, never an in-place edit.
var definitionVersions = map[model.Kind]string{
	model.KindAgentDefinition:    "definitionVersion",
	model.KindBusinessProcess:    "processDefinitionVersion",
	model.KindTask:               "taskDefinitionVersion",
	model.KindCapabilityContract: "contractVersion",
	model.KindCapabilityBinding:  "bindingVersion",
	model.KindDataBinding:        "bindingVersion",
	model.KindAuthorityBinding:   "bindingVersion",
	model.KindEventBinding:       "bindingVersion",
}

// DefinitionVersion returns the version a definition document declares; versioned is false for kinds that declare none.
func DefinitionVersion(d *model.Document) (version string, versioned bool) {
	property, versioned := definitionVersions[d.Kind]
	if !versioned {
		return "", false
	}
	value, _ := d.Value.(map[string]any)
	version, _ = value[property].(string)
	return version, true
}
