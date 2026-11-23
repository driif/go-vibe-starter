package cmd

import (
	"errors"

	"github.com/driif/go-vibe-starter/internal/db"
	"github.com/spf13/cobra"
)

func databaseCommand() *cobra.Command {
	database := &cobra.Command{Use: "db", Short: "Manage the database"}
	migrations := &cobra.Command{Use: "migrate", Short: "Apply or inspect embedded migrations"}
	for _, name := range []string{"up", "down", "status"} {
		migrations.AddCommand(&cobra.Command{Use: name, Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if cfg.DatabaseURL == "" {
				return errors.New("DATABASE_URL is required")
			}
			return db.Migrate(c.Context(), cfg.DatabaseURL, c.Name(), c.OutOrStdout())
		}})
	}
	database.AddCommand(migrations)
	return database
}
