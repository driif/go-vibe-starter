package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/driif/go-vibe-starter/internal/server/config/env"
)

// DatabaseMigrationTable name is baked into the binary
// This setting should always be in sync with dbconfig.yml, sqlboiler.toml and the live database (e.g. to be able to test producation dumps locally)
const DatabaseMigrationTable = "migrations"

// DatabaseMigrationFolder (folder with all *.sql migrations).
// This settings should always be in sync with dbconfig.yaml and Dockerfile (the final app stage).
// It's expected that the migrations folder lives at the root of this project or right next to the app binary.
var DatabaseMigrationFolder = filepath.Join(env.GetProjectRootDir(), "migrations")

// Database represents the database configuration.
type Database struct {
	Host             string
	Port             int
	Username         string
	Password         string            `json:"-"`          // sensitive
	Database         string
	AdditionalParams map[string]string `json:",omitempty"` // Optional additional connection parameters mapped into the connection string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
}

// ConnectionString generates a DSN for sql.Open or equivalents, assuming Postgres keyword/value syntax.
func (c Database) ConnectionString() string {
	return c.buildDSN(c.Database)
}

// ConnectionSpecString generates a DSN pointing to the "spec" database (used for testing against production schema dumps).
func (c Database) ConnectionSpecString() string {
	return c.buildDSN("spec")
}

// buildDSN constructs a PostgreSQL keyword/value connection string for the given dbname.
// Values are quoted so that passwords or usernames containing spaces or special characters are handled correctly.
func (c Database) buildDSN(dbname string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "host=%s port=%d user=%s password=%s dbname=%s",
		escapeValue(c.Host),
		c.Port,
		escapeValue(c.Username),
		escapeValue(c.Password),
		escapeValue(dbname),
	)

	if _, ok := c.AdditionalParams["sslmode"]; !ok {
		b.WriteString(" sslmode=disable")
	}

	if len(c.AdditionalParams) > 0 {
		keys := make([]string, 0, len(c.AdditionalParams))
		for k := range c.AdditionalParams {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Fprintf(&b, " %s=%s", k, escapeValue(c.AdditionalParams[k]))
		}
	}

	return b.String()
}

// escapeValue quotes a PostgreSQL DSN value if it contains whitespace, single quotes, or backslashes.
// Per https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-KEYWORD-VALUE
func escapeValue(v string) string {
	if !strings.ContainsAny(v, " \t\n\\'") {
		return v
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}
