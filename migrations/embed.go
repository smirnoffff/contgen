// Package migrations embeds the ordered database schema in the application.
package migrations

import "embed"

// Files contains each immutable, forward-only SQL migration.
//
//go:embed *.sql
var Files embed.FS
