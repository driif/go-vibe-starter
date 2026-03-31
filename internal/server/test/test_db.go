package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/driif/go-vibe-starter/internal/server/config"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB is a test database instance
type TestDB struct {
	*sql.DB
	Container testcontainers.Container
	t         *testing.T
}

// NewDBInstance creates a new test database instance
func NewDBInstance(t *testing.T, conf config.App) *TestDB {
	t.Helper()

	db := &TestDB{t: t}

	req := testcontainers.ContainerRequest{
		Image: "postgres",
		Env: map[string]string{
			"POSTGRES_PASSWORD": conf.Database.Password,
			"POSTGRES_USER":     conf.Database.Username,
			"POSTGRES_DB":       conf.Database.Database,
		},
		NetworkMode:  "bridge",
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}

	var err error
	db.Container, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	mappedPort, err := db.Container.MappedPort(context.Background(), "5432")
	if err != nil {
		t.Fatalf("failed to get mapped port: %s", err)
	}

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		conf.Database.Username,
		conf.Database.Password,
		conf.Database.Host,
		mappedPort.Port(),
		conf.Database.Database,
	)

	db.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		t.Fatalf("failed to open database: %s", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %s", err)
	}

	return db
}

// Close closes the test database instance
func (db *TestDB) Close() {
	db.t.Helper()

	err := db.Container.Terminate(context.TODO())
	if err != nil {
		db.t.Fatalf("failed to terminate container: %s", err)
	}

	err = db.DB.Close()
	if err != nil {
		db.t.Fatalf("failed to close database: %s", err)
	}

	db = nil
}

// ApplyFixtures applies the migrations and test fixtures to the test database instance
func (db *TestDB) ApplyFixtures(t *testing.T) {
	db.t.Helper()

	goose.SetTableName(config.DatabaseMigrationTable)

	if err := goose.Up(db.DB, config.DatabaseMigrationFolder); err != nil {
		db.t.Fatal(err)
	}

	// ctx := context.Background()

	// inserts := Inserts()

	// // insert test fixtures in an auto-managed db transaction
	// err = dbutils.WithTransaction(ctx, db.DB, func(tx boil.ContextExecutor) error {
	// 	t.Helper()
	// 	for _, fixture := range inserts {
	// 		if err = fixture.Insert(ctx, tx, boil.Infer()); err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }
}

// WithTestDB creates a new test database instance and applies the migrations and test fixtures to it
func WithTestDB(t *testing.T, closure func(db *sql.DB)) {
	t.Helper()

	conf := config.DefaultServiceConfigFromEnv()

	testDB := NewDBInstance(t, conf)

	testDB.ApplyFixtures(t)

	closure(testDB.DB)

	require.NotPanics(t, func() { testDB.Close() })

}
