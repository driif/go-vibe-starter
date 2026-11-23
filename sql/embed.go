// Package sql embeds the goose migrations so the binary can migrate without the source tree.
package sql

import "embed"

// Migrations holds the migrations directory. The all: prefix keeps the directive valid when the
// directory holds only .gitkeep.
//
//go:embed all:migrations
var Migrations embed.FS
