// DEMO: notes

// Package notes is the demo slice: a note belongs to the subject that created it, and every
// statement is scoped by that owner, so another owner's note is not found rather than forbidden.
//
// The slice is these files, plus the mountNotes line in internal/server/routes.go:
//
//	proto/api/v1/notes.proto           API contract and field rules
//	sql/migrations/00001_notes.sql     the notes table
//	sql/queries/notes.sql              the four statements
//	internal/domain/notes/             this package: Store over the pool, its test
//	internal/rpc/notes/                the Connect handler and its test
//	internal/server/routes_notes.go    the mount file
//
// Marker forms, read by scripts/init.sh:
//
//   - line 1 exactly "// DEMO: notes" (Go, proto; in Go line 2 is blank) or "-- DEMO: notes"
//     (SQL): the file is deleted;
//   - "// DEMO:line" at the end of a line in a shared file: the line is deleted;
//   - generated output under internal/gen and internal/db/gen is never scanned, only regenerated.
//
// To remove the slice by hand: delete the marked files and the marked hub line, run make gen and
// go mod tidy, then recreate the database or add a migration that drops the notes table.
package notes
