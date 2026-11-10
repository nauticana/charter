// Package schema embeds the keel schema modules of Charter's relational model: one directory of YAML table
// definitions per module dependency.yml lists, and each module's code-list seed in seed/.
package schema

import "embed"

// Modules holds dependency.yml, every module's ab_meta.yml and table files, and the seed files.
//
//go:embed dependency.yml */*.yml
var Modules embed.FS
