package migrations

import "embed"

// FS holds all schema migrations, applied in filename order by
// infrastructure/postgres. Kept top-level (not under infrastructure/)
// so schema history reads as product history, not adapter detail.
//
//go:embed *.sql
var FS embed.FS
