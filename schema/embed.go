// Package schema embeds the Charter JSON Schemas and catalog for in-process validation.
package schema

import "embed"

//go:embed catalog.json common.schema.json */*.schema.json
var FS embed.FS
