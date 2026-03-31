package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/driif/go-vibe-starter/internal/server/config"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate <subcommand>",
	Short: "Run database migrations",
	Long:  `Run database migrations using goose (up, down, status, create, etc.).`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	},
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Migrate the DB to the most recent version",
	Run:   runMigrate("up"),
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back the version by 1",
	Run:   runMigrate("down"),
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Dump the migration status for the current DB",
	Run:   runMigrate("status"),
}

var migrateResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Roll back all migrations",
	Run:   runMigrate("reset"),
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateResetCmd)
	dbCmd.AddCommand(migrateCmd)
}

func runMigrate(command string) func(*cobra.Command, []string) {
	return func(_ *cobra.Command, _ []string) {
		conf := config.DefaultServiceConfigFromEnv()

		db, err := sql.Open("postgres", conf.Database.ConnectionString())
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		goose.SetTableName(config.DatabaseMigrationTable)

		if err := goose.Run(command, db, config.DatabaseMigrationFolder); err != nil {
			fmt.Printf("Migration %s failed: %v\n", command, err)
			os.Exit(1)
		}
	}
}
