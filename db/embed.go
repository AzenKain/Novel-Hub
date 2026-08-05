// Package db carries the SQL schema into the binary. The embed has to live here rather than
// in pkg/database because go:embed cannot reach outside its own package directory, and the
// schema must ship inside the binary: the release artifact is a single file.
package db

import "embed"

//go:embed schema/*.sql
var SchemaFS embed.FS
