// Package test is the integration harness: fresh Postgres databases cloned from a migrated
// template, the real handler behind httptest, and tokens from the in-process identity provider.
package test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/driif/go-vibe-starter/internal/db"
)

// postgresImage has the same major version as the dev compose file.
const postgresImage = "postgres:17-alpine"

const templateDB = "app_template"

var (
	startOnce sync.Once
	hostPort  string
	startErr  error
	sequence  atomic.Int64

	notIdentifier = regexp.MustCompile(`[^a-z0-9]`)
)

// Database returns the URL of a fresh database cloned from the migrated template. It is dropped
// when the test ends. Safe for parallel tests.
func Database(t *testing.T) string {
	t.Helper()
	return createDatabase(t, " TEMPLATE "+templateDB)
}

// EmptyDatabase returns the URL of a fresh database with no migration applied. It is dropped
// when the test ends.
func EmptyDatabase(t *testing.T) string {
	t.Helper()
	return createDatabase(t, "")
}

func createDatabase(t *testing.T, suffix string) string {
	t.Helper()
	if err := start(); err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	name := fmt.Sprintf("t%d_%s", sequence.Add(1), notIdentifier.ReplaceAllString(strings.ToLower(t.Name()), "_"))
	if len(name) > 63 {
		name = name[:63]
	}
	if err := exec(context.Background(), fmt.Sprintf("CREATE DATABASE %q%s", name, suffix)); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		if err := exec(context.Background(), fmt.Sprintf("DROP DATABASE %q WITH (FORCE)", name)); err != nil {
			t.Errorf("drop database: %v", err)
		}
	})
	return dsn(name)
}

// start runs one container per test process and migrates the template database. A failure is
// returned to every later caller. Ryuk removes the container when the process ends.
func start() error {
	startOnce.Do(func() { startErr = startPostgres(context.Background()) })
	return startErr
}

func startPostgres(ctx context.Context) error {
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        postgresImage,
			Env:          map[string]string{"POSTGRES_PASSWORD": "postgres"},
			ExposedPorts: []string{"5432/tcp"},
			Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "full_page_writes=off"},
			WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		},
		Started: true,
	})
	if err != nil {
		return err
	}
	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		return err
	}
	hostPort = host + ":" + port.Port()
	if err := exec(ctx, "CREATE DATABASE "+templateDB); err != nil {
		return err
	}
	return db.Migrate(ctx, dsn(templateDB), "up", io.Discard)
}

func dsn(database string) string {
	return fmt.Sprintf("postgres://postgres:postgres@%s/%s?sslmode=disable", hostPort, database)
}

func exec(ctx context.Context, statement string) error {
	admin, err := pgx.Connect(ctx, dsn("postgres"))
	if err != nil {
		return err
	}
	defer admin.Close(ctx)
	_, err = admin.Exec(ctx, statement)
	return err
}
